//go:build it

package sqlexec

// 回滚 SQL 生成的真实数据库集成测试。
//
// 这一组用例钉的不是「能不能拼出字符串」，而是**把生成的 SQL 真跑一遍，数据是否回到改前**。
// 回滚链路的失效形态全是静默的：列值错位、主键丢位、把二进制列的 hex 文本误当值解码写回、按不存在的键
// WHERE 命中 0 行——每一种都能「执行成功」，只有把数据读回来才知道错。
//
// 因此每条用例都走生产采集链路（doUpdate/doDelete 采旧值）→ 生成 → 执行生成结果 → 读回比对。
// 拒绝类用例（主键被改、无主键、列已被删）断言的是「不给生成」，因为那种情形下生成的语句
// 看着能跑却会改错数据。
//
// 运行：cd server && go test -tags it -count=1 -run TestITRollback ./internal/db/application/

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dbi/value"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

const rbTableName = "it_sql_rollback"

// rbRowCount 表当前行数
func rbRowCount(t *testing.T, conn *dbi.DbConn) int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.QueryContext(context.Background(), "SELECT COUNT(*) AS cnt FROM "+quote(rbTableName))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	cnt, ok := value.ValToInt64(rows[0]["cnt"])
	require.True(t, ok, "计数取值形态异常: %T", rows[0]["cnt"])
	return cnt
}

// rbReq 组装一次交互式执行请求（与 appRunAtomic 同源，但本文件自带表与库定位）
func rbReq(conn *dbi.DbConn, sql string) *dto.DbSQLExecReq {
	return &dto.DbSQLExecReq{DbConn: conn, DbId: conn.Info.InstanceId, Db: conn.Info.Database, SQL: sql}
}

// rbExec 走生产执行链路，返回被捕获的执行记录（含当时采到的旧值）
func rbExec(t *testing.T, conn *dbi.DbConn, sql string) *entity.DbSQLExec {
	t.Helper()
	repo := &captureSqlExecRepo{RepoImpl: &base.RepoImpl[*entity.DbSQLExec]{}}
	app := &DbSQLExecAppImpl{}
	app.dbSQLExecRepo = repo
	_, err := app.exec(context.Background(), rbReq(conn, sql), nil)
	require.NoError(t, err, "执行链路本身失败: %s", sql)
	require.NotEmpty(t, repo.records, "执行记录未被写入，拿不到旧值")
	record := repo.records[len(repo.records)-1]
	require.Equal(t, entity.DbSQLExecStatusSuccess, record.Status, "执行未成功，旧值不可信")
	return record
}

// rbPrepare 建表并写入若干行，ddl 为列定义（主键形态由用例决定）
func rbPrepare(t *testing.T, conn *dbi.DbConn, ddl string, inserts ...string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	rbMustExec(t, conn, "DROP TABLE IF EXISTS "+quote(rbTableName))
	rbMustExec(t, conn, "CREATE TABLE "+quote(rbTableName)+" "+ddl)
	for _, insert := range inserts {
		rbMustExec(t, conn, fmt.Sprintf(ddlInsertFormat(conn), insert))
	}
}

// ddlInsertFormat 让用例只写值列表，表名由本函数补齐（各方言引号形态不同）
func ddlInsertFormat(conn *dbi.DbConn) string {
	return "INSERT INTO " + conn.GetDialect().Quoter().QuoteIdent(rbTableName) + " VALUES %s"
}

func rbMustExec(t *testing.T, conn *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := conn.Exec(sql); err != nil {
		t.Fatalf("exec failed [%s]: %s", sql, err.Error())
	}
}

// rbColumn 读回某列在 id=? 上的值
func rbColumn(t *testing.T, conn *dbi.DbConn, column string, id any) string {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.QueryContext(context.Background(),
		fmt.Sprintf("SELECT %s FROM %s WHERE %s = %v", quote(column), quote(rbTableName), quote("id"), id))
	require.NoError(t, err)
	require.Len(t, rows, 1, "读回失败：目标行不存在")
	return fmt.Sprint(rows[0][column])
}

// rbGenerateUpdate 用生产同一条路径生成回滚 SQL：直接走 genUpdateRollback（GenRollback 的
// UPDATE 分支同一入口），确保「无主键/主键被改」守卫、并发校验与 CAS 条件都被真实覆盖，
// 而不是绕过它们只测拼串。
func rbGenerateUpdate(t *testing.T, conn *dbi.DbConn, record *entity.DbSQLExec) ([]string, string, error) {
	t.Helper()
	outcome, err := rbGenerateUpdateOutcome(t, conn, record)
	if err != nil {
		return nil, "", err
	}
	return outcome.sqls, outcome.note, nil
}

// rbGenerateUpdateOutcome 同 rbGenerateUpdate，但交回完整结论（含 verified/conflicts），
// 供需要断言并发校验结果的用例使用
func rbGenerateUpdateOutcome(t *testing.T, conn *dbi.DbConn, record *entity.DbSQLExec) (*updateRollbackOutcome, error) {
	t.Helper()
	table, err := loadRollbackColumns(context.Background(), conn, record.Table)
	if err != nil {
		return nil, err
	}
	rows, err := decodeOldValueRows(context.Background(), record.OldValue)
	if err != nil {
		return nil, err
	}
	require.NotEmpty(t, rows, "生产采集链路必须留下旧值，否则本用例验证不了任何东西")
	return genUpdateRollback(context.Background(), conn, table, record, rows)
}

// rbGenerateInsert 走 INSERT 回滚的生产入口（GenRollback 的 INSERT 分支直接调 genInsertRollback），
// 行数据源是解析出的插入值快照（存于 NewValue）
func rbGenerateInsert(t *testing.T, conn *dbi.DbConn, record *entity.DbSQLExec) ([]string, []dto.SqlRollbackConflict, error) {
	t.Helper()
	table, err := loadRollbackColumns(context.Background(), conn, record.Table)
	if err != nil {
		return nil, nil, err
	}
	rows, err := decodeOldValueRows(context.Background(), record.NewValue)
	if err != nil {
		return nil, nil, err
	}
	require.NotEmpty(t, rows, "INSERT 必须采集到插入值快照，否则本用例验证不了任何东西")
	return genInsertRollback(context.Background(), conn, table, rows)
}

// rbAutoIncrementDDL 各方言的自增主键建表列定义（用于验证「插入时主键缺位」的定位回查路径）
func rbAutoIncrementDDL(node appDialectNode) string {
	switch node.name {
	case "mysql":
		return "(id INT AUTO_INCREMENT PRIMARY KEY, val VARCHAR(200))"
	case "pg":
		return "(id SERIAL PRIMARY KEY, val VARCHAR(200))"
	default: // sqlite
		return "(id INTEGER PRIMARY KEY AUTOINCREMENT, val TEXT)"
	}
}

// rbBinaryDDL 各方言的二进制列类型名（mysql/sqlite 用 BLOB，pg 用 BYTEA）
func rbBinaryDDL(node appDialectNode) string {
	if node.name == "pg" {
		return "(id INT PRIMARY KEY, payload BYTEA)"
	}
	return "(id INT PRIMARY KEY, payload BLOB)"
}

// rbHexLiteral 把 hex 文本渲染成该方言的二进制字面量（与生产 codec 同形态：mysql/sqlite X'..'、pg '\x..'）
func rbHexLiteral(node appDialectNode, hexText string) string {
	if node.name == "pg" {
		return "'\\x" + hexText + "'"
	}
	return "X'" + hexText + "'"
}

// rollbackNodes 参与互测的方言（与 appDialectNodes 同源）
func rollbackNodes() []appDialectNode {
	return appDialectNodes
}

// TestITRollbackUpdateRestoresOldValues UPDATE 的回滚必须把数据真的改回去。
//
// 这是整条链路唯一有意义的验收：生成的语句执行成功而值没变，等于零
func TestITRollbackUpdateRestoresOldValues(t *testing.T) {
	for _, node := range rollbackNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			rbPrepare(t, conn, node.ddl, "(1, 'a')", "(2, 'b')")
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 'changed' WHERE %s <= 2", quote(rbTableName), quote("val"), quote("id")))
			require.NotEmpty(t, record.OldValue, "UPDATE 必须采集到旧值")
			assert.Equal(t, "changed", rbColumn(t, conn, "val", 1), "前置条件：数据确实被改过")

			statements, _, err := rbGenerateUpdate(t, conn, record)
			require.NoError(t, err)
			require.Len(t, statements, 2, "两行改动必须生成两条回滚语句")
			for _, statement := range statements {
				rbMustExec(t, conn, statement)
			}

			assert.Equal(t, "a", rbColumn(t, conn, "val", 1), "回滚后必须还原第一行")
			assert.Equal(t, "b", rbColumn(t, conn, "val", 2), "回滚后必须还原第二行")
		})
	}
}

// TestITRollbackUpdateRestoresBinaryVerbatim 二进制列的回滚必须逐字节还原。
//
// 二进制旧值经 dbi 的 ValuerBytes 归一为 hex 文本，渲染回滚字面量时必须原样包成方言的十六进制
// 字面量（mysql/sqlite X'..'、pg '\x..'），绝不能按 base64 解码——hex 字符集是 base64 子集，
// 长度为 4 的倍数的 hex 串（偶数字节值极常见）会被 base64 静默解成完全不同的字节，回滚「执行成功」
// 而二进制数据被悄悄改写。用 cafebabe/deadbeef 这类 4 字节值正是为踩中该形态
func TestITRollbackUpdateRestoresBinaryVerbatim(t *testing.T) {
	const originalHex = "cafebabe"
	const changedHex = "deadbeef"
	for _, node := range rollbackNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent
			rbPrepare(t, conn, rbBinaryDDL(node))
			rbMustExec(t, conn, fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (1, %s)",
				quote(rbTableName), quote("id"), quote("payload"), rbHexLiteral(node, originalHex)))

			// 走生产采集链路把二进制列改成另一个值（doUpdate 采旧值 + captureUpdatePostImage 采执行后快照）
			record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = %s WHERE %s = 1",
				quote(rbTableName), quote("payload"), rbHexLiteral(node, changedHex), quote("id")))
			require.NotEmpty(t, record.OldValue, "UPDATE 必须采集到二进制旧值")
			// 读回经 ValuerBytes 归一为小写 hex 文本（hex.EncodeToString 恒小写）
			assert.Equal(t, changedHex, rbColumn(t, conn, "payload", 1), "前置条件：二进制列确实被改过")

			statements, _, err := rbGenerateUpdate(t, conn, record)
			require.NoError(t, err)
			require.Len(t, statements, 1)
			rbMustExec(t, conn, statements[0])

			assert.Equal(t, originalHex, rbColumn(t, conn, "payload", 1),
				"回滚后二进制列必须逐字节还原（base64 误解码会在此写回错误字节）")
		})
	}
}

// TestITRollbackUpdateRestoresBitValue 位列（mysql BIT(8)）的回滚必须按数值还原，不能被塌缩成布尔。
//
// BIT(8) 存 255，回滚若把 255 渲染成 true（mysql 里 true=1）就会写回 1，且 CAS 条件 flags=true
// 也匹配不上真实值——两种失效都「执行成功」而数据没还原。必须真跑一遍读回 255
// （pg 的 BIT 是位串、sqlite 无原生 BIT，数值语义以 mysql 最典型，故本用例只跑 mysql）
func TestITRollbackUpdateRestoresBitValue(t *testing.T) {
	node := rollbackNodes()[0] // mysql
	conn := node.conn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent
	rbPrepare(t, conn, "(id INT PRIMARY KEY, flags BIT(8))")
	rbMustExec(t, conn, fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (1, 255)", quote(rbTableName), quote("id"), quote("flags")))

	// 走生产采集链路把位列改成 0（doUpdate 采旧值 255 + captureUpdatePostImage 采执行后快照 0）
	record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 0 WHERE %s = 1", quote(rbTableName), quote("flags"), quote("id")))
	require.NotEmpty(t, record.OldValue, "UPDATE 必须采集到位列旧值")
	assert.Equal(t, "0", rbColumn(t, conn, "flags", 1), "前置条件：位列确实被改成 0")

	statements, _, err := rbGenerateUpdate(t, conn, record)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	rbMustExec(t, conn, statements[0])

	// 读回经 ValuerBit 归一为整数：必须是 255 而非塌缩后的 1
	assert.Equal(t, "255", rbColumn(t, conn, "flags", 1), "回滚后 BIT(8) 必须还原为 255（塌缩成 true 会写回 1）")
}

// TestITRollbackDeleteReinsertsRows DELETE 的回滚是按旧值重新插回整行：
// 少插一行就是数据丢失，且不会有任何报错
func TestITRollbackDeleteReinsertsRows(t *testing.T) {
	for _, node := range rollbackNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			rbPrepare(t, conn, node.ddl, "(1, 'a')", "(2, 'b')")
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("DELETE FROM %s WHERE %s = 2", quote(rbTableName), quote("id")))
			require.NotEmpty(t, record.OldValue, "DELETE 必须采集到被删行的整行数据")

			table, err := loadRollbackColumns(context.Background(), conn, record.Table)
			require.NoError(t, err)
			rows, err := decodeOldValueRows(context.Background(), record.OldValue)
			require.NoError(t, err)
			statements, err := rollbackInserts(context.Background(), conn, table, rows)
			require.NoError(t, err)
			require.Len(t, statements, 1)

			rbMustExec(t, conn, statements[0])
			assert.Equal(t, "b", rbColumn(t, conn, "val", 2), "被删行必须原样回来")
		})
	}
}

// TestITRollbackDeleteRestoresAliasedWhere 表别名限定的 DELETE 必须采集回被删行并原样回滚。
//
// 走的是 doDelete 的另一条采集拼接：FROM 子句拼上别名（`FROM t x`）、WHERE 逐字回放（`x.id = 2`），
// 只有别名在 FROM 与 WHERE 两侧同时成立才 SELECT 得回被删行；解析器若把别名丢了，回查会因未定义
// 别名报错→旧值采空→静默丢回滚。pg 的 `DELETE FROM t AS x WHERE x.id` 语法无歧义（sqlite 单表 DELETE
// 不支持表别名，mysql 单表 DELETE 别名写法跨版本不一致），故以 pg 为代表钉死这条共享拼接路径
func TestITRollbackDeleteRestoresAliasedWhere(t *testing.T) {
	node := rollbackNodes()[1] // pg
	conn := node.conn(t)
	defer conn.Close()
	rbPrepare(t, conn, node.ddl, "(1, 'a')", "(2, 'b')")
	quote := conn.GetDialect().Quoter().QuoteIdent

	record := rbExec(t, conn, fmt.Sprintf("DELETE FROM %s AS x WHERE x.%s = 2", quote(rbTableName), quote("id")))
	require.NotEmpty(t, record.OldValue, "别名 DELETE 必须采集到被删行的整行数据")

	table, err := loadRollbackColumns(context.Background(), conn, record.Table)
	require.NoError(t, err)
	rows, err := decodeOldValueRows(context.Background(), record.OldValue)
	require.NoError(t, err)
	statements, err := rollbackInserts(context.Background(), conn, table, rows)
	require.NoError(t, err)
	require.Len(t, statements, 1)

	rbMustExec(t, conn, statements[0])
	assert.Equal(t, "b", rbColumn(t, conn, "val", 2), "别名 DELETE 的被删行必须原样回来")
}

// TestITRollbackRefusesPrimaryKeyUpdate 主键被改过时必须拒绝生成。
//
// 旧值里只有主键的原值，改后的行号没人知道：按原值 WHERE 命中 0 行，
// 界面上是「执行成功」，数据却留在改错的状态——这种「成功的空操作」比报错危险得多
func TestITRollbackRefusesPrimaryKeyUpdate(t *testing.T) {
	for _, node := range []appDialectNode{rollbackNodes()[0], rollbackNodes()[1]} {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			rbPrepare(t, conn, node.ddl, "(1, 'a')")
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 99 WHERE %s = 1", quote(rbTableName), quote("id"), quote("id")))
			statements, _, err := rbGenerateUpdate(t, conn, record)
			require.Error(t, err, "主键被改过时必须拒绝生成")
			assert.Empty(t, statements, "拒绝时不得交回任何语句")
			// 必须说出是「主键被改过」。不钉这一层的话，同一个用例会因为 MySQL 把重复列名
			// 回成 id/id1 而提前撞进「列已不存在」分支，错误原因错了用例却是绿的
			assert.Contains(t, err.Error(), "主键", "错误必须点名主键: "+err.Error())
			assert.NotContains(t, err.Error(), "已不存在", "不得把主键改名误报成「列已被删除」")
		})
	}
}

// TestITRollbackRefusesTableWithoutPrimaryKey 无主键表上不能生成回滚 UPDATE：
// 没有定位条件就等于全表改，那会把没被改过的行一起改掉
func TestITRollbackRefusesTableWithoutPrimaryKey(t *testing.T) {
	for _, node := range []appDialectNode{rollbackNodes()[0], rollbackNodes()[1]} {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			rbPrepare(t, conn, "(id INT, val VARCHAR(200))", "(1, 'a')")
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 'x' WHERE %s = 1", quote(rbTableName), quote("val"), quote("id")))
			statements, _, err := rbGenerateUpdate(t, conn, record)
			require.Error(t, err, "无主键表不得生成回滚 UPDATE")
			assert.Empty(t, statements)
		})
	}
}

// TestITRollbackRefusesAfterColumnDropped 列被删掉后必须拒绝：
// 继续按旧值生成会让值与列错位，把已删列的值写进别的列
func TestITRollbackRefusesAfterColumnDropped(t *testing.T) {
	for _, node := range []appDialectNode{rollbackNodes()[0], rollbackNodes()[1]} {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			rbPrepare(t, conn, node.ddl, "(1, 'a')")
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 'x' WHERE %s = 1", quote(rbTableName), quote("val"), quote("id")))
			rbMustExec(t, conn, "ALTER TABLE "+quote(rbTableName)+" DROP COLUMN "+quote("val"))

			statements, _, err := rbGenerateUpdate(t, conn, record)
			require.Error(t, err, "旧值里的列已不存在时必须拒绝生成")
			assert.Empty(t, statements)
		})
	}
}

// TestITRollbackBigIntKeyKeepsEveryDigit BIGINT 主键必须逐位保留并真的命中原来那行。
//
// 用 2^53 之外的奇数：默认 JSON 解析走 float64，末位会被改写成邻近的偶数，
// 生成的 WHERE 谁也不命中——回滚「执行成功」而数据一动不动
func TestITRollbackBigIntKeyKeepsEveryDigit(t *testing.T) {
	for _, node := range []appDialectNode{rollbackNodes()[0], rollbackNodes()[1]} {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			const bigKey = 9007199254740993
			rbPrepare(t, conn, "(id BIGINT PRIMARY KEY, val VARCHAR(200))", fmt.Sprintf("(%d, 'a')", bigKey))
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 'x' WHERE %s = %d", quote(rbTableName), quote("val"), quote("id"), bigKey))
			statements, _, err := rbGenerateUpdate(t, conn, record)
			require.NoError(t, err)
			require.Len(t, statements, 1)
			assert.Contains(t, statements[0], "9007199254740993", "主键字面量必须逐位保留: %s", statements[0])

			rbMustExec(t, conn, statements[0])
			assert.Equal(t, "a", rbColumn(t, conn, "val", bigKey), "按精确主键回滚后原行必须还原")
		})
	}
}

// TestITRollbackTruncationIsDeclared 旧值采集被截断时必须声明「只覆盖已采集行」。
//
// 采集侧到上限就停止扫描而不报错，所以旧值看着齐备、实际少行：不声明的话用户以为
// 点一下就能全改回去，而超出上限的那些行当时就没人记过，谁也回不来
func TestITRollbackTruncationIsDeclared(t *testing.T) {
	conn := rollbackNodes()[0].conn(t)
	defer conn.Close()

	quote := conn.GetDialect().Quoter().QuoteIdent
	rbPrepare(t, conn, "(id INT PRIMARY KEY, val VARCHAR(200))")
	insert := ddlInsertFormat(conn)
	total := sqlExecAuditRows + 3
	for i := 1; i <= total; i++ {
		rbMustExec(t, conn, fmt.Sprintf(insert, fmt.Sprintf("(%d, 'v%d')", i, i)))
	}

	record := rbExec(t, conn, fmt.Sprintf("DELETE FROM %s WHERE %s > 0", quote(rbTableName), quote("id")))
	rows, err := decodeOldValueRows(context.Background(), record.OldValue)
	require.NoError(t, err)
	require.Len(t, rows, sqlExecAuditRows, "采集上限必须在生产链路上真实生效，否则本用例无意义")

	table, err := loadRollbackColumns(context.Background(), conn, record.Table)
	require.NoError(t, err)
	statements, err := rollbackInserts(context.Background(), conn, table, rows)
	require.NoError(t, err)
	require.Len(t, statements, sqlExecAuditRows)

	assert.NotEmpty(t, rollbackTruncationNote(context.Background(), rows), "到过上限必须给出声明")

	for _, statement := range statements {
		rbMustExec(t, conn, statement)
	}
	// 回滚后仍少了 3 行：这就是那句声明要告知用户的事实
	assert.Equal(t, int64(sqlExecAuditRows), rbRowCount(t, conn), "只能找回已采集的行，多出的行当时就没记录")
}

// TestITRollbackInsertDeletesNewRow INSERT 的回滚是「删掉刚插入的行」。
//
// 主键齐备（在插入列内）时直接按「主键 + 全部插入值」生成 DELETE：
// 删多一行是数据丢失，删少一行则回滚无效，都必须靠真跑一遍再数行来钉死
func TestITRollbackInsertDeletesNewRow(t *testing.T) {
	for _, node := range rollbackNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			rbPrepare(t, conn, node.ddl)
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (1, 'a')", quote(rbTableName), quote("id"), quote("val")))
			require.NotEmpty(t, record.NewValue, "INSERT 必须采集到插入值快照")
			assert.Equal(t, int64(1), rbRowCount(t, conn), "前置条件：行确实被插入")

			statements, conflicts, err := rbGenerateInsert(t, conn, record)
			require.NoError(t, err)
			assert.Empty(t, conflicts, "主键齐备的单行插入不该有定位冲突")
			require.Len(t, statements, 1, "一行插入必须生成一条 DELETE")

			rbMustExec(t, conn, statements[0])
			assert.Equal(t, int64(0), rbRowCount(t, conn), "回滚后刚插入的行必须消失")
		})
	}
}

// TestITRollbackInsertAutoIncrementFindsRow 自增主键不在插入列内时，回滚必须按插入值回查到唯一行、
// 补上它真正的主键再生成 DELETE。少了这一步就只能凭值删，遇到同值行会删错对象
func TestITRollbackInsertAutoIncrementFindsRow(t *testing.T) {
	for _, node := range rollbackNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			rbPrepare(t, conn, rbAutoIncrementDDL(node))
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("INSERT INTO %s (%s) VALUES ('a')", quote(rbTableName), quote("val")))
			require.NotEmpty(t, record.NewValue, "INSERT 必须采集到插入值快照")

			statements, conflicts, err := rbGenerateInsert(t, conn, record)
			require.NoError(t, err)
			assert.Empty(t, conflicts, "唯一候选行不该有冲突")
			require.Len(t, statements, 1)
			// 回查补进的真实主键必须出现在条件里（自增出的 id=1）
			assert.Contains(t, statements[0], quote("id"), "必须把回查到的真实主键补进删除条件: %s", statements[0])

			rbMustExec(t, conn, statements[0])
			assert.Equal(t, int64(0), rbRowCount(t, conn), "回滚后自增行必须被删掉")
		})
	}
}

// TestITRollbackInsertAmbiguousMatchRefuses 插入值命中多行时必须拒绝生成：
// 凭值删行在这种情况下会删掉「本来就在」或「后来又插入」的同值行，宁可停下也不误删
func TestITRollbackInsertAmbiguousMatchRefuses(t *testing.T) {
	for _, node := range rollbackNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			rbPrepare(t, conn, rbAutoIncrementDDL(node))
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("INSERT INTO %s (%s) VALUES ('dup')", quote(rbTableName), quote("val")))
			// 再插一行同值：回查时 'dup' 命中两行，无法唯一定位
			rbMustExec(t, conn, fmt.Sprintf("INSERT INTO %s (%s) VALUES ('dup')", quote(rbTableName), quote("val")))

			statements, conflicts, err := rbGenerateInsert(t, conn, record)
			require.NoError(t, err, "多行命中是「该行不生成」而非整体报错")
			assert.Empty(t, statements, "无法唯一定位时不得交回任何 DELETE")
			require.Len(t, conflicts, 1)
			assert.Equal(t, dto.RollbackConflictAmbiguousMatch, conflicts[0].Reason, "必须点名是多行命中")
		})
	}
}

// TestITRollbackUpdateDetectsConcurrentModification 变更之后又被他人改过的行必须被判为并发冲突、
// 不生成回滚。CAS 的意义正在于此：把陈旧旧值盖回去会悄悄抹掉他人的修改，而界面显示「执行成功」
func TestITRollbackUpdateDetectsConcurrentModification(t *testing.T) {
	for _, node := range rollbackNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			rbPrepare(t, conn, node.ddl, "(1, 'a')")
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 'changed' WHERE %s = 1", quote(rbTableName), quote("val"), quote("id")))
			require.NotEmpty(t, record.NewValue, "UPDATE 必须采集执行后快照，否则并发校验无从谈起")

			// 模拟他人在那次变更之后又改了同一行
			rbMustExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 'other' WHERE %s = 1", quote(rbTableName), quote("val"), quote("id")))

			outcome, err := rbGenerateUpdateOutcome(t, conn, record)
			require.NoError(t, err, "并发修改是「该行不生成」而非整体报错")
			assert.Empty(t, outcome.sqls, "被并发改过的行不得生成回滚语句")
			require.Len(t, outcome.conflicts, 1)
			assert.Equal(t, dto.RollbackConflictConcurrentModified, outcome.conflicts[0].Reason, "必须点名是并发修改")
		})
	}
}

// TestITRollbackUpdateCasCarriesPostImage 校验通过的行，回滚 WHERE 必须带上执行后快照值充当 CAS，
// 且真的能把值还原：条件里少了快照值就等于放弃了并发保护
func TestITRollbackUpdateCasCarriesPostImage(t *testing.T) {
	for _, node := range rollbackNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			rbPrepare(t, conn, node.ddl, "(1, 'a')")
			quote := conn.GetDialect().Quoter().QuoteIdent

			record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 'changed' WHERE %s = 1", quote(rbTableName), quote("val"), quote("id")))
			outcome, err := rbGenerateUpdateOutcome(t, conn, record)
			require.NoError(t, err)
			require.True(t, outcome.verified, "有执行后快照且未被并发改动时必须判定为已校验")
			assert.Empty(t, outcome.conflicts)
			require.Len(t, outcome.sqls, 1)
			assert.Contains(t, outcome.sqls[0], "changed", "回滚 WHERE 必须带上执行后快照值充当 CAS: %s", outcome.sqls[0])

			rbMustExec(t, conn, outcome.sqls[0])
			assert.Equal(t, "a", rbColumn(t, conn, "val", 1), "CAS 命中后值必须还原")
		})
	}
}

// TestITRollbackUpdateRestoresReservedWordColumn 保留字列名（order）的 UPDATE 必须能采集旧值并回滚还原。
//
// 解析器会剥掉标识符引号，采集回查 SELECT 若把列名裸拼回去，`SELECT order, id FROM ...`
// 在各方言都是语法错（order 是保留字）→ 旧值采空 → 回滚整体不可用。这是「执行成功但记录丢失」
// 的静默失效，必须真跑一遍采集→生成→还原钉死
func TestITRollbackUpdateRestoresReservedWordColumn(t *testing.T) {
	for _, node := range rollbackNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent
			rbPrepare(t, conn, fmt.Sprintf("(%s INT PRIMARY KEY, %s VARCHAR(20))", quote("id"), quote("order")), "(1, 'a')")

			record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 'changed' WHERE %s = 1", quote(rbTableName), quote("order"), quote("id")))
			require.NotEmpty(t, record.OldValue, "保留字列的 UPDATE 必须采集到旧值")
			assert.Equal(t, "changed", rbColumn(t, conn, "order", 1), "前置条件：保留字列确实被改过")

			statements, _, err := rbGenerateUpdate(t, conn, record)
			require.NoError(t, err)
			require.Len(t, statements, 1)
			rbMustExec(t, conn, statements[0])
			assert.Equal(t, "a", rbColumn(t, conn, "order", 1), "回滚后保留字列必须还原")
		})
	}
}

// TestITRollbackUpdateRestoresAliasedSetColumn 带表别名限定的 SET（UPDATE t AS x SET x.val）必须能采集并回滚。
//
// 解析器把 SET 目标原样保留成 `x.val`，采集与生成两侧都必须用 lastSegment 归一为裸列名 `val`：
// 不归一则采集 SELECT 拼出 “ `x.val` “（整体当一个标识符）查不到列、旧值采空。
// 别名限定 SET 是 mysql/oracle/dm/mssql 的语法，pg/sqlite 拒绝（SET 列不得带表名限定），故只跑 mysql 作代表
func TestITRollbackUpdateRestoresAliasedSetColumn(t *testing.T) {
	node := rollbackNodes()[0] // mysql
	conn := node.conn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent
	rbPrepare(t, conn, node.ddl, "(1, 'a')")

	record := rbExec(t, conn, fmt.Sprintf("UPDATE %s AS x SET x.%s = 'changed' WHERE x.%s = 1", quote(rbTableName), quote("val"), quote("id")))
	require.NotEmpty(t, record.OldValue, "别名 SET 的 UPDATE 必须采集到旧值")
	assert.Equal(t, "changed", rbColumn(t, conn, "val", 1), "前置条件：数据确实被改过")

	statements, _, err := rbGenerateUpdate(t, conn, record)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	rbMustExec(t, conn, statements[0])
	assert.Equal(t, "a", rbColumn(t, conn, "val", 1), "回滚后别名列必须还原")
}

// TestITRollbackUpdateRestoresCaseSensitiveIdentifiers 大小写敏感表/列（pg 建表时加引号即大小写敏感）
// 的 UPDATE 必须全链路可用：采集回查若把 "RbCaseSensitive"/"Value" 裸拼，pg 会折叠成小写而查不到，
// 旧值采空、回滚不可用。mysql/sqlite 标识符大小写不敏感，此形态只在 pg（及 oracle/dm）成立，故只跑 pg
func TestITRollbackUpdateRestoresCaseSensitiveIdentifiers(t *testing.T) {
	var pgNode *appDialectNode
	for i := range appDialectNodes {
		if appDialectNodes[i].name == "pg" {
			pgNode = &appDialectNodes[i]
		}
	}
	require.NotNil(t, pgNode, "需要 pg 方言验证大小写敏感标识符")

	conn := pgNode.conn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent

	const tbl = "RbCaseSensitive"
	rbMustExec(t, conn, "DROP TABLE IF EXISTS "+quote(tbl))
	rbMustExec(t, conn, fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY, %s VARCHAR(20))", quote(tbl), quote("Id"), quote("Value")))
	defer rbMustExec(t, conn, "DROP TABLE IF EXISTS "+quote(tbl))
	rbMustExec(t, conn, fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (1, 'a')", quote(tbl), quote("Id"), quote("Value")))

	record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 'changed' WHERE %s = 1", quote(tbl), quote("Value"), quote("Id")))
	require.NotEmpty(t, record.OldValue, "大小写敏感表/列的 UPDATE 必须采集到旧值")

	statements, _, err := rbGenerateUpdate(t, conn, record)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	rbMustExec(t, conn, statements[0])

	_, rows, err := conn.QueryContext(context.Background(), fmt.Sprintf("SELECT %s FROM %s WHERE %s = 1", quote("Value"), quote(tbl), quote("Id")))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "a", fmt.Sprint(rows[0]["Value"]), "回滚后大小写敏感列必须还原")
}

// TestITRollbackUpdateRestoresFoldedCaseIdentifiers 锁定「裸写混合大小写标识符」在折叠方言上的旧值采集。
//
// pg 会把裸写的 UserName 折叠成 username，目录里存的是小写名。采集回查若把「剥掉引号后的用户书写形态」
// 直接 QuoteIdent，就引用到一个不存在的 "UserName"：回查 SELECT 报错 → 旧值静默采空 → 回滚能力整体消失，
// 而 UPDATE 本身照常执行成功（要到用户点回滚时才暴露）。失效形态是静默的，所以必须真跑一遍数据读回来验。
// 不变量：回查用的标识符必须先归一到目录真名再引用。
func TestITRollbackUpdateRestoresFoldedCaseIdentifiers(t *testing.T) {
	var pgNode *appDialectNode
	for i := range appDialectNodes {
		if appDialectNodes[i].name == "pg" {
			pgNode = &appDialectNodes[i]
		}
	}
	require.NotNil(t, pgNode, "需要 pg 方言验证折叠大小写标识符")

	conn := pgNode.conn(t)
	defer conn.Close()

	const tbl = "rb_folded_case"
	rbMustExec(t, conn, "DROP TABLE IF EXISTS "+tbl)
	rbMustExec(t, conn, fmt.Sprintf("CREATE TABLE %s (id INT PRIMARY KEY, username VARCHAR(20))", tbl))
	defer rbMustExec(t, conn, "DROP TABLE IF EXISTS "+tbl)
	rbMustExec(t, conn, fmt.Sprintf("INSERT INTO %s (id, username) VALUES (1, 'a')", tbl))

	// 表名按目录原样书写（元数据可按名定位），变更列裸写成混合大小写：pg 折叠后命中目录里的小写列，
	// 语句合法且确实改了数据。此时回查若引用书写形态 "UserName" 就会报错，旧值静默采空
	record := rbExec(t, conn, "UPDATE rb_folded_case SET UserName = 'changed' WHERE id = 1")
	require.NotEmpty(t, record.OldValue, "折叠大小写的裸写变更列必须采集到旧值（回查须按目录真名引用）")

	statements, _, err := rbGenerateUpdate(t, conn, record)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	rbMustExec(t, conn, statements[0])

	_, rows, err := conn.QueryContext(context.Background(), "SELECT username FROM rb_folded_case WHERE id = 1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "a", fmt.Sprint(rows[0]["username"]), "回滚后必须还原折叠列的原值")
}

// TestITRollbackSkipsSchemaQualifiedTable 锁 schema 限定表的 fail-closed 守卫。
//
// 采集回查与回滚生成都只按连接默认 schema 解析标识符（元数据查询以 current_schema() 过滤）。
// 若跨 schema DML 照常采集：UPDATE 会采到默认 schema 同名表的旧值、生成静默无效的回滚；
// INSERT 更严重——回滚 DELETE 会删掉默认 schema 同名表的行（数据丢失）。守卫必须在采集侧
// 直接跳过（Table/OldValue/NewValue 全空 → 生成侧如实报「无旧值/无插入快照」），且真实目标表
// 正确改动、默认 schema 同名表分毫未动。
//
// pg 一例即代表全部方言：Schema 字段由四方言共用的 parseTableRef 填充，守卫本身与方言无关。
func TestITRollbackSkipsSchemaQualifiedTable(t *testing.T) {
	var pgNode *appDialectNode
	for i := range appDialectNodes {
		if appDialectNodes[i].name == "pg" {
			pgNode = &appDialectNodes[i]
		}
	}
	require.NotNil(t, pgNode, "需要 pg 方言验证 schema 限定表的 fail-closed 守卫")

	conn := pgNode.conn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent

	const tbl = "RbSchemaShadow"
	const secondSchema = "rb_shadow_s2"
	// 默认 schema(public) 与第二 schema 各建同名表：制造「跨 schema 限定 + 默认 schema 有同名影子表」的危险前提
	rbMustExec(t, conn, fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(tbl)))
	rbMustExec(t, conn, fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY, %s VARCHAR(20))", quote(tbl), quote("id"), quote("col")))
	defer rbMustExec(t, conn, fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(tbl)))
	rbMustExec(t, conn, fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (1, 'public')", quote(tbl), quote("id"), quote("col")))

	rbMustExec(t, conn, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", secondSchema))
	defer rbMustExec(t, conn, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", secondSchema))
	rbMustExec(t, conn, fmt.Sprintf("CREATE TABLE %s.%s (%s INT PRIMARY KEY, %s VARCHAR(20))", secondSchema, quote(tbl), quote("id"), quote("col")))
	rbMustExec(t, conn, fmt.Sprintf("INSERT INTO %s.%s (%s, %s) VALUES (1, 'B')", secondSchema, quote(tbl), quote("id"), quote("col")))

	readCol := func(from string, id int) string {
		_, rows, err := conn.QueryContext(context.Background(),
			fmt.Sprintf("SELECT %s FROM %s WHERE %s = %d", quote("col"), from, quote("id"), id))
		require.NoError(t, err)
		require.Len(t, rows, 1)
		return fmt.Sprint(rows[0]["col"])
	}

	// 跨 schema UPDATE：守卫必须跳过采集（Table/OldValue 全空）
	updRecord := rbExec(t, conn, fmt.Sprintf("UPDATE %s.%s SET %s = 'changed' WHERE %s = 1", secondSchema, quote(tbl), quote("col"), quote("id")))
	assert.Empty(t, updRecord.Table, "schema 限定表必须跳过采集：Table 不得写入")
	assert.Empty(t, updRecord.OldValue, "schema 限定表必须跳过采集：不得采到（错误表的）旧值")
	// 真实目标表被正确改动；默认 schema 同名影子表分毫未动（证明未采错/未写错表）
	assert.Equal(t, "changed", readCol(secondSchema+"."+quote(tbl), 1), "真实目标表 s2.t 应被改动")
	assert.Equal(t, "public", readCol(quote(tbl), 1), "默认 schema 同名表必须未被跨 schema 更新触碰")

	// 跨 schema INSERT：守卫必须跳过 NewValue 采集（否则回滚 DELETE 会删错默认 schema 同名表的行）
	insRecord := rbExec(t, conn, fmt.Sprintf("INSERT INTO %s.%s (%s, %s) VALUES (2, 'C')", secondSchema, quote(tbl), quote("id"), quote("col")))
	assert.Empty(t, insRecord.Table, "schema 限定 INSERT 必须跳过采集：Table 不得写入")
	assert.Empty(t, insRecord.NewValue, "schema 限定 INSERT 必须跳过插入快照：否则回滚会删错表")
	assert.Equal(t, "C", readCol(secondSchema+"."+quote(tbl), 2), "真实目标表 s2.t 应插入成功")
}

// TestITRollbackSkipsDeleteWithLimit 锁 DELETE … LIMIT 的行限制 fail-closed 守卫（mysql 形态）。
//
// DELETE … LIMIT 2 只删 WHERE 匹配集（id>0，共 5 行）中由存储引擎选定的 2 行。若照常采集，
// 回查 SELECT … WHERE id>0 会捞回全部 5 行，回滚便为「其实没被删的 3 行」也生成 INSERT——
// 撞主键报错，或在无主键表插出重复行（数据损坏）。守卫必须跳过采集，让生成侧如实报「无旧值」。
func TestITRollbackSkipsDeleteWithLimit(t *testing.T) {
	node := rollbackNodes()[0] // mysql：DELETE … LIMIT 是 mysql 的行限制形态
	conn := node.conn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent
	rbPrepare(t, conn, node.ddl, "(1,'a')", "(2,'b')", "(3,'c')", "(4,'d')", "(5,'e')")

	record := rbExec(t, conn, fmt.Sprintf("DELETE FROM %s WHERE %s > 0 LIMIT 2", quote(rbTableName), quote("id")))
	assert.Empty(t, record.OldValue, "带 LIMIT 的 DELETE 必须跳过旧值采集（fail-closed）")
	assert.Equal(t, int64(3), rbRowCount(t, conn), "DELETE … LIMIT 2 应只删 2 行、剩 3 行")

	// 采集为空 → 生成侧无旧值可用：如实拒绝，而不是生成会重插 3 行幸存数据的错误回滚
	rows, err := decodeOldValueRows(context.Background(), record.OldValue)
	require.NoError(t, err)
	assert.Empty(t, rows, "无旧值快照，回滚生成必须无行可用")
}

// TestITRollbackSkipsUpdateWithLimit 锁 UPDATE … LIMIT 的行限制 fail-closed 守卫（mysql 形态）。
//
// 带 LIMIT 的 UPDATE 只影响 WHERE 匹配集的一个非确定子集，回查按 WHERE 捞回全部匹配行会为
// 未被更新的行生成覆盖式回滚。守卫必须跳过旧值与执行后快照的采集。
func TestITRollbackSkipsUpdateWithLimit(t *testing.T) {
	node := rollbackNodes()[0] // mysql
	conn := node.conn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent
	rbPrepare(t, conn, node.ddl, "(1,'a')", "(2,'b')", "(3,'c')")

	record := rbExec(t, conn, fmt.Sprintf("UPDATE %s SET %s = 'x' WHERE %s > 0 LIMIT 2", quote(rbTableName), quote("val"), quote("id")))
	assert.Empty(t, record.OldValue, "带 LIMIT 的 UPDATE 必须跳过旧值采集（fail-closed）")
	assert.Empty(t, record.NewValue, "带 LIMIT 的 UPDATE 不得采集执行后快照")
}

// TestITRollbackSkipsInsertConflictClause 锁 INSERT 冲突处理形态的 fail-closed 守卫。
//
// 目标行（id=1）已存在时，REPLACE / INSERT IGNORE / ON DUPLICATE KEY UPDATE / ON CONFLICT
// 都不新增行（忽略 / 改为更新 / 先删后插）。INSERT 的回滚是「删掉刚插入的行」，若照常按 VALUES
// 采集快照，生成的 DELETE 会命中并删除那条既有行（数据丢失）。守卫必须对每种形态都跳过采集，
// 让既有行安然留存、生成侧如实报「无插入快照」。
func TestITRollbackSkipsInsertConflictClause(t *testing.T) {
	cases := []struct {
		node string
		name string
		sql  func(q func(string) string) string
	}{
		{"mysql", "insert-ignore", func(q func(string) string) string {
			return fmt.Sprintf("INSERT IGNORE INTO %s (%s,%s) VALUES (1,'new')", q(rbTableName), q("id"), q("val"))
		}},
		{"mysql", "replace", func(q func(string) string) string {
			return fmt.Sprintf("REPLACE INTO %s (%s,%s) VALUES (1,'new')", q(rbTableName), q("id"), q("val"))
		}},
		{"mysql", "on-duplicate", func(q func(string) string) string {
			return fmt.Sprintf("INSERT INTO %s (%s,%s) VALUES (1,'new') ON DUPLICATE KEY UPDATE %s='new'", q(rbTableName), q("id"), q("val"), q("val"))
		}},
		{"pg", "on-conflict-nothing", func(q func(string) string) string {
			return fmt.Sprintf("INSERT INTO %s (%s,%s) VALUES (1,'new') ON CONFLICT (%s) DO NOTHING", q(rbTableName), q("id"), q("val"), q("id"))
		}},
		{"pg", "on-conflict-update", func(q func(string) string) string {
			return fmt.Sprintf("INSERT INTO %s (%s,%s) VALUES (1,'new') ON CONFLICT (%s) DO UPDATE SET %s=EXCLUDED.%s", q(rbTableName), q("id"), q("val"), q("id"), q("val"), q("val"))
		}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.node+"/"+c.name, func(t *testing.T) {
			var node appDialectNode
			for _, n := range rollbackNodes() {
				if n.name == c.node {
					node = n
				}
			}
			require.NotEmpty(t, node.name, "需要 %s 方言节点", c.node)
			conn := node.conn(t)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent
			rbPrepare(t, conn, node.ddl, "(1,'old')")

			record := rbExec(t, conn, c.sql(quote))
			assert.Empty(t, record.NewValue, "冲突形态 INSERT 必须跳过插入快照采集（否则回滚 DELETE 会删既有行）")
			assert.Empty(t, record.Table, "冲突形态 INSERT 不得写入 Table（生成侧据此报无插入快照）")
			// 既有行必须仍在：证明执行未新增行，也没有任何回滚会删它
			assert.Equal(t, int64(1), rbRowCount(t, conn), "冲突形态 INSERT 后仍应只有那 1 行既有数据")
		})
	}
}
