//go:build it

package application

// SQL文件执行链路（DbSqlExecApp.ExecReader）真实数据库集成测试。
//
// 与 transfer 包 importDumpStream 的分工：迁移/备份恢复链路会**过滤**脚本内事务控制语句（自己管批级提交），
// 而用户在「SQL编辑器 → 执行SQL文件」提交的文件按脚本原样执行事务控制语句（与 mysql CLI、psql 一致的
// 脚本自治语义）：真实提交点由数据库决定，失败时原样返回底层数据库错误，**不猜测也不改写失败原因**
//（从语句文本推提交点不可靠：mysql的DDL同样隐式提交，而pg事务内BEGIN不提交、sqlite事务内BEGIN直接报错）。
// 该差异只有在真实数据库上才可验证。
//
// 覆盖：复杂切割值在真实库的落地一致性 / 纯DML文件失败整体回滚 /
//
//	含事务控制语句文件的真实执行行为与残留 / ctx取消
//
// 运行：cd server && go test -tags it -count=1 -run TestITExecReader ./internal/db/application/

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/i18n"
)

// appExecReader 驱动生产 ExecReader 执行一段SQL文件内容
func appExecReader(ctx context.Context, conn *dbi.DbConn, filename, content string) error {
	app := &dbSqlExecAppImpl{}
	return app.ExecReader(ctx, &dto.SqlReaderExec{
		DbConn:   conn,
		Reader:   strings.NewReader(content),
		Filename: filename,
	})
}

// appExecComplexFile 各方言一致的复杂SQL文件：值内含分号/引号/注释符/换行，注释内含分号与块注释
func appExecComplexFile(quote func(string) string) string {
	return "-- 头部注释; 内含分号不应切分\n" +
		fmt.Sprintf("INSERT INTO %s VALUES (1, 'a;b''c');\n", quote(appExecTable)) +
		fmt.Sprintf("INSERT INTO %s VALUES (2, '中文🙂;emoji;半角分号');\n", quote(appExecTable)) +
		"/* 块注释\n   跨行; 分号; 不应切分 */\n" +
		fmt.Sprintf("INSERT INTO %s VALUES (3, 'line1\nline2');\n", quote(appExecTable)) +
		fmt.Sprintf("INSERT INTO %s VALUES (4, 'semi; /* 不是块注释 */; -- 不是行注释');\n", quote(appExecTable)) +
		fmt.Sprintf("INSERT INTO %s VALUES (5, '');\n", quote(appExecTable))
}

// TestITExecReaderComplexSqlFileRealExec 复杂SQL文件在三方言真实库上被正确切割并完整执行
func TestITExecReaderComplexSqlFileRealExec(t *testing.T) {
	for _, node := range appDialectNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := appPrepareTable(t, node)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent

			require.NoError(t, appExecReader(context.Background(), conn, "complex.sql", appExecComplexFile(quote)),
				"复杂SQL文件执行失败")

			_, rows, err := conn.Query("SELECT id, val FROM " + quote(appExecTable) + " ORDER BY id")
			require.NoError(t, err)
			require.Len(t, rows, 5, "应执行5条INSERT（注释内的分号不得造成伪切分）")

			want := map[int64]string{
				1: "a;b'c",
				2: "中文🙂;emoji;半角分号",
				3: "line1\nline2",
				4: "semi; /* 不是块注释 */; -- 不是行注释",
				5: "",
			}
			for _, r := range rows {
				id, ok := dbi.ValToInt64(r["id"])
				require.True(t, ok, "id取值形态异常: %T", r["id"])
				val := strings.TrimSpace(fmt.Sprintf("%v", r["val"]))
				if b, isBytes := r["val"].([]byte); isBytes {
					val = string(b)
				}
				assert.Equal(t, want[id], val, "第%d行的值未原样落库", id)
			}
		})
	}
}

// TestITExecReaderFailureRollbackNoResidue 纯DML文件末尾失败：整个文件必须整体回滚，且不得误报提前提交
func TestITExecReaderFailureRollbackNoResidue(t *testing.T) {
	for _, node := range appDialectNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := appPrepareTable(t, node)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent

			ctx := context.Background()
			file := fmt.Sprintf("INSERT INTO %s VALUES (1, 'a');\nINSERT INTO %s VALUES (2, 'b');\nINSERT INTO it_no_such_table VALUES (3);\n",
				quote(appExecTable), quote(appExecTable))
			err := appExecReader(ctx, conn, "fail.sql", file)
			require.Error(t, err, "末尾坏语句应导致执行失败")

			assert.Equal(t, int64(0), appRowCount(t, conn), "纯DML文件失败后不得残留部分数据")
			// 失败原因必须是底层数据库原始错误（含不存在的表名），便于运维定位失败语句
			assert.Contains(t, err.Error(), "it_no_such_table", "失败提示应包含底层数据库错误: %s", err.Error())
		})
	}
}

// TestITExecReaderTxnControlStmtExecutedPerFile 文件内的 BEGIN 按脚本原样执行，失败时返回底层数据库原始错误。
//
// 提交点与中断位置完全由数据库决定，本机实测（见appDialectNodes）：
//   - mysql：BEGIN 隐式提交在途写入 → 末尾失败的回滚覆盖不到该提交点，残留1行；
//   - postgres：事务内 BEGIN 仅告警不提交 → 整体回滚，残留0行；
//   - sqlite：事务内 BEGIN 直接报错 → 在该语句即中断，残留0行。
//
// 所以失败原因必须是数据库自身的错误（各方言期望片段见 errOnTxnBegin），而不是平台代数据库推断的事务后果；
// 数据库版本行为变化会让本用例变红，提醒重审该矩阵
func TestITExecReaderTxnControlStmtExecutedPerFile(t *testing.T) {
	for _, node := range appDialectNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := appPrepareTable(t, node)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent

			ctx := context.Background()
			file := fmt.Sprintf("INSERT INTO %s VALUES (1, 'before-begin');\nBEGIN;\nINSERT INTO %s VALUES (2, 'after-begin');\nINSERT INTO it_no_such_table VALUES (3);\n",
				quote(appExecTable), quote(appExecTable))
			err := appExecReader(ctx, conn, "txn.sql", file)
			require.Error(t, err, "末尾坏语句应导致执行失败")

			assert.Contains(t, err.Error(), node.errOnTxnBegin,
				"[%s] 失败原因应为底层数据库原始错误（期望片段 %q）: %s", node.name, node.errOnTxnBegin, err.Error())
			assert.Equal(t, node.survivedOnTxnBegin, appRowCount(t, conn),
				"[%s] 事务控制语句的真实提交/残留行为与本机实测不符，需重审该矩阵", node.name)
		})
	}
}

// TestITExecReaderCancelled ctx取消时必须报取消而非"执行成功"
func TestITExecReaderCancelled(t *testing.T) {
	for _, node := range appDialectNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := appPrepareTable(t, node)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			file := fmt.Sprintf("INSERT INTO %s VALUES (1, 'a');\nINSERT INTO %s VALUES (2, 'b');\n",
				quote(appExecTable), quote(appExecTable))
			err := appExecReader(ctx, conn, "cancel.sql", file)
			require.Error(t, err, "已取消的ctx必须返回错误")

			assert.Contains(t, err.Error(), i18n.TC(ctx, imsg.ErrSqlExecCancelled),
				"取消应报明确的取消错误: %s", err.Error())
			assert.Equal(t, int64(0), appRowCount(t, conn), "取消后不得残留已执行数据")
		})
	}
}
