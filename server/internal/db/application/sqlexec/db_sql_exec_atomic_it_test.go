//go:build it

package sqlexec

// 单请求内多语句原子执行（Atomic）的真实数据库集成测试。
//
// 钉死四条与「用户以为在事务里操作」直接相关的行为：
//  1. 整批共用一个事务：后面一条失败，前面已「执行成功」的语句不得落库；
//  2. 不开原子开关时保持逐条自动提交的既有语义（同一批数据必须留在库里），
//     否则「原子」成了默认行为，改掉了所有人的执行习惯；
//  3. 旧值采集必须发生在事务内：同批第二条语句的旧值应是第一条改完的值，
//     若走连接池读到的是事务外快照，据此生成的回滚 SQL 会把数据改错；
//  4. SELECT/DDL 声明为原子批次时直接拒绝，且不执行任何一条（不给「可回滚」的假承诺）。
//
// 与 ExecReader 用例的分工：ExecReader 是「脚本自治」语义（事务控制由脚本与数据库决定），
// 本文件是「平台代管批次」语义（由请求显式声明要不要原子），两者不可互相替代。
//
// 运行：cd server && go test -tags it -count=1 -run TestITAtomicExec ./internal/db/application/

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

// captureSqlExecRepo 捕获写入的执行记录，用于断言旧值采集结果（不查库，避免引入额外依赖）
type captureSqlExecRepo struct {
	*base.RepoImpl[*entity.DbSQLExec]

	records []*entity.DbSQLExec
}

func (c *captureSqlExecRepo) GetPageList(*entity.DbSQLExecQuery, ...string) (*model.PageResult[*entity.DbSQLExec], error) {
	return nil, nil
}

func (c *captureSqlExecRepo) Insert(_ context.Context, e *entity.DbSQLExec) error {
	c.records = append(c.records, e)
	return nil
}

// appAtomicSetup 建表并写入 (1,'a')，返回连接与捕获型仓储
func appAtomicSetup(t *testing.T, node appDialectNode) (*dbi.DbConn, *captureSqlExecRepo) {
	t.Helper()
	conn := appPrepareTable(t, node)
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec(fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (1, 'a')", quote(appExecTable), quote("id"), quote("val")))
	require.NoError(t, err)
	return conn, &captureSqlExecRepo{RepoImpl: &base.RepoImpl[*entity.DbSQLExec]{}}
}

// appRunAtomic 走生产执行链路（procdef 传 nil，即未绑定流程定义不做策略判定）
func appRunAtomic(ctx context.Context, conn *dbi.DbConn, repo *captureSqlExecRepo, sql string, atomic bool) ([]*dto.DbSQLExecRes, error) {
	app := &DbSQLExecAppImpl{}
	app.dbSQLExecRepo = repo
	return app.exec(ctx, &dto.DbSQLExecReq{
		DbConn: conn,
		DbId:   conn.Info.InstanceId,
		Db:     conn.Info.Database,
		SQL:    sql,
		Atomic: atomic,
	}, nil)
}

// appVal 读回 id=1 的 val（事务是否真的回滚，只能由库侧事实证明）
func appVal(t *testing.T, conn *dbi.DbConn) string {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query("SELECT " + quote("val") + " FROM " + quote(appExecTable) + " WHERE " + quote("id") + " = 1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	return fmt.Sprint(rows[0]["val"])
}

// TestITAtomicExecRollbackWholeBatch 后面一条失败 → 整批不得落库
func TestITAtomicExecRollbackWholeBatch(t *testing.T) {
	for _, node := range appDialectNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn, repo := appAtomicSetup(t, node)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent

			sql := fmt.Sprintf("UPDATE %s SET %s = 'x' WHERE %s = 1; INSERT INTO it_no_such_table VALUES (9);",
				quote(appExecTable), quote("val"), quote("id"))
			res, err := appRunAtomic(context.Background(), conn, repo, sql, true)
			require.Error(t, err, "原子批次里有语句失败时必须整批报错")
			assert.Nil(t, res, "整批回滚不得返回「部分成功」的结果数组")
			// 断言取渲染结果里的稳定内容而不是提示文案：文案属 i18n，按语言会变，
			// 而「第几条/共几条」与底层数据库原始错误才是这条守卫真正要钉的东西
			assert.Contains(t, err.Error(), "2/2", "错误必须指明是批次里第几条失败")
			assert.Contains(t, err.Error(), "it_no_such_table", "原始数据库错误不得被吞掉")
			assert.Equal(t, "a", appVal(t, conn), "第一条 UPDATE 必须随整批回滚，val 仍是原值")
			// 执行记录写在平台自己的元数据库里，**不随目标库事务回滚**：若在提交前逐条落库，这里会
			// 留下一条「成功」记录——语句其实一条没生效，而回滚 SQL 入口还能据它当时采到的旧值
			// 把陈旧数据写回现网（非原子路径由下一个用例作对照）
			assert.Empty(t, repo.records, "整批回滚不得留下任何执行记录，否则作废的语句会被当成已生效的变更")
		})
	}
}

// TestITAtomicExecNonAtomicKeepsPartialSuccess 不开原子开关时保持既有的逐条自动提交语义
func TestITAtomicExecNonAtomicKeepsPartialSuccess(t *testing.T) {
	for _, node := range appDialectNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn, repo := appAtomicSetup(t, node)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent

			sql := fmt.Sprintf("UPDATE %s SET %s = 'x' WHERE %s = 1; INSERT INTO it_no_such_table VALUES (9);",
				quote(appExecTable), quote("val"), quote("id"))
			res, err := appRunAtomic(context.Background(), conn, repo, sql, false)
			require.NoError(t, err, "非原子批次逐条执行，失败信息写在各自结果里而不整批报错")
			require.NotEmpty(t, res)
			assert.Equal(t, "x", appVal(t, conn), "未声明原子时第一条已生效的数据不得被撤销")
			// 与原子路径的对照面：不开原子时成功的语句必须当场留下记录（不能因为
			// 原子批次改成「提交后统一落库」而把两条路径一起推迟）
			require.Len(t, repo.records, 1, "非原子批次里已生效的语句必须当场落执行记录")
			assert.Contains(t, repo.records[0].SQL, "UPDATE")
		})
	}
}

// TestITAtomicExecOldValueReadInsideTx 旧值必须在事务内采集：
// 第二条 UPDATE 记录的旧值应是第一条改完后的 'x'，而不是事务外的 'a'
func TestITAtomicExecOldValueReadInsideTx(t *testing.T) {
	for _, node := range appDialectNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn, repo := appAtomicSetup(t, node)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent

			sql := fmt.Sprintf("UPDATE %s SET %s = 'x' WHERE %s = 1; UPDATE %s SET %s = 'z' WHERE %s = 1;",
				quote(appExecTable), quote("val"), quote("id"), quote(appExecTable), quote("val"), quote("id"))
			_, err := appRunAtomic(context.Background(), conn, repo, sql, true)
			require.NoError(t, err)
			assert.Equal(t, "z", appVal(t, conn), "整批成功后改动必须落库")

			require.Len(t, repo.records, 2, "两条语句各应有一条执行记录")
			assert.Contains(t, repo.records[0].OldValue, "a", "第一条的旧值是原值")
			// 这条断言就是 WalkQueryRowsInTx 的存在理由：若旧值采集走连接池（事务外快照），
			// 这里读到的仍是 'a'，据此生成的回滚 SQL 会把 val 改回 'a' 而不是 'x'
			assert.Contains(t, repo.records[1].OldValue, "x", "第二条的旧值必须是事务内已改成的值")
			assert.NotContains(t, repo.records[1].OldValue, "\"a\"", "第二条的旧值不得取到事务外的原值")
		})
	}
}

// TestITAtomicExecRejectsNonDmlStatements SELECT 与 DDL 不得进原子批次，且拒绝时一条都不执行
func TestITAtomicExecRejectsNonDmlStatements(t *testing.T) {
	for _, node := range appDialectNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn, repo := appAtomicSetup(t, node)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent

			for _, sql := range []string{
				fmt.Sprintf("UPDATE %s SET %s = 'x' WHERE %s = 1; SELECT 1;", quote(appExecTable), quote("val"), quote("id")),
				fmt.Sprintf("UPDATE %s SET %s = 'x' WHERE %s = 1; CREATE TABLE it_atomic_ddl (id int);", quote(appExecTable), quote("val"), quote("id")),
			} {
				want := "SELECT 1"
				if strings.Contains(sql, "CREATE TABLE") {
					want = "it_atomic_ddl"
				}
				_, err := appRunAtomic(context.Background(), conn, repo, sql, true)
				require.Error(t, err, "含非 DML 语句的原子批次必须被拒绝: %s", sql)
				// 拒绝必须落到「是哪条语句不属于这三类」，否则用户只会看到一句无法行动的规则说明
				assert.Contains(t, err.Error(), want, "拒绝提示必须带上被拒语句")
				var bizErr *errorx.BizError
				require.True(t, errors.As(err, &bizErr), "非原子类别必须是业务拒绝错误，而不是数据库报错")
				assert.Equal(t, "a", appVal(t, conn), "拒绝发生在执行前，同批第一条也不得生效")
			}
			assert.Empty(t, repo.records, "被拒绝的批次不应留下执行记录")
		})
	}
}

// TestITAtomicExecEmptyAndSingle 单条语句也可声明原子（等价于给它包一个事务），空批次不误判为原子失败
func TestITAtomicExecEmptyAndSingle(t *testing.T) {
	node := appDialectNodes[0]
	conn, repo := appAtomicSetup(t, node)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent

	only := fmt.Sprintf("UPDATE %s SET %s = 'x' WHERE %s = 1;", quote(appExecTable), quote("val"), quote("id"))
	res, err := appRunAtomic(context.Background(), conn, repo, only, true)
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "x", appVal(t, conn))

	// 只有空白与注释：切不出语句时不进入事务，也不报错
	res, err = appRunAtomic(context.Background(), conn, repo, strings.TrimSpace("-- nothing to do"), true)
	require.NoError(t, err)
	assert.Empty(t, res)
}
