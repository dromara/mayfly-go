package dbtool

import (
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/dialect/mysql"

	"github.com/stretchr/testify/require"
)

// TestIsReadOnlyStatement 守住 DbQueryData 的安全边界：它是「只读」工具且不经人工审批，
// 任何能改数据/有副作用的语句都不能从这条通道溜过去。
//
// 旧实现把整个 OtherStmt 兜底大类当只读放行，CALL/DO/SET/VACUUM/EXPLAIN ANALYZE 得以绕过
// ExecSql 的审批门——本测试正是钉这些形态，并确认纯只读语句仍放行。
func TestIsReadOnlyStatement(t *testing.T) {
	dialect := dbi.GetDialect(dbi.ToDbType("mysql"))
	require.NotNil(t, dialect, "mysql 方言未注册，测试无法进行")
	parser := dialect.GetSQLParser()
	splitter := dialect.GetSQLSplitter()

	// 确认只读：走查询通道回结果集、无副作用
	readOnly := []string{
		"SELECT * FROM t",
		"  select 1",
		"WITH c AS (SELECT 1) SELECT * FROM c",
		"SHOW TABLES",
		"SHOW CREATE TABLE t",
		"EXPLAIN SELECT * FROM t",
		"DESC t",
		"DESCRIBE t",
		// 前导注释不得影响判定（切割器保留注释原文）
		"-- 查一下\nSELECT * FROM t",
	}
	for _, sql := range readOnly {
		require.Truef(t, isReadOnlyStatement(parser, splitter, sql), "应判只读: %s", sql)
	}

	// 非只读：DML/DDL 本就拒；重点是这些在旧实现里被 OtherStmt 兜底误判为只读、可绕过审批
	notReadOnly := []string{
		"INSERT INTO t VALUES (1)",
		"UPDATE t SET a = 1",
		"DELETE FROM t WHERE id = 1",
		"DROP TABLE t",
		"CREATE TABLE t (id INT)",
		"ALTER TABLE t ADD COLUMN c INT",
		"TRUNCATE TABLE t",
		"CALL my_proc()",
		"SET @x = 1",
		"VACUUM",
		"DO SLEEP(1)",
		// MERGE（upsert）是写：裸 MERGE 与 WITH 前置 MERGE 都不得从只读通道溜过
		"MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.a = 1",
		"WITH c AS (SELECT id FROM t) MERGE INTO u USING c ON (u.id = c.id) WHEN MATCHED THEN UPDATE SET u.a = 1",
		// WITH 开头但实为写：主语句是 DML，或数据修改型 CTE（体内 DELETE）
		"WITH c AS (SELECT id FROM t) DELETE FROM u WHERE id IN (SELECT id FROM c)",
		"WITH c AS (DELETE FROM t RETURNING id) SELECT * FROM c",
		// SELECT ... INTO 建表/导出是写（Classify→insert），不得从只读通道溜过
		"SELECT id INTO bak_users FROM users WHERE id > 1",
		"SELECT a FROM t INTO OUTFILE '/tmp/x.txt'",
		// EXPLAIN ANALYZE 会真正执行底层语句（这里是 DELETE），只读通道不得放行
		"EXPLAIN ANALYZE DELETE FROM t WHERE id = 1",
		"EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM t",
	}
	for _, sql := range notReadOnly {
		require.Falsef(t, isReadOnlyStatement(parser, splitter, sql), "应判非只读: %s", sql)
	}
}

// TestDbQueryData_MultiStatementHidesWrite 单条元素里塞分号多条时，判定必须以拆分后的单语句为单位。
// Classify 只解析到首个 ';'，若对整串判定，`SELECT 1; DELETE FROM t` 会按首句判只读、而执行整串
// （PG/SQLite 驱动会把 DELETE 一并跑掉）——只读通道藏写。DbQueryData 修复后先 SplitSQL 再逐条判。
func TestDbQueryData_MultiStatementHidesWrite(t *testing.T) {
	dialect := dbi.GetDialect(dbi.ToDbType("mysql"))
	require.NotNil(t, dialect)
	parser, splitter := dialect.GetSQLParser(), dialect.GetSQLSplitter()

	stmts, err := splitStatements(splitter, "SELECT 1; DELETE FROM t")
	require.NoError(t, err)
	require.Len(t, stmts, 2, "分号分隔的多语句必须拆开判定，不能整串当一个")

	allReadOnly := true
	for _, single := range stmts {
		if !isReadOnlyStatement(parser, splitter, single) {
			allReadOnly = false
		}
	}
	require.False(t, allReadOnly, "SELECT;DELETE 含写语句，只读通道必须整体拒绝，否则执行整串会跑掉 DELETE")
}
