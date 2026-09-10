package itest

// 反向异构数据库迁移集成测试（与正向 TestITMysqlToPgMigration/TestITMysqlToSqliteMigration 对称）：
//   - pg 源 → mysql / sqlite 目标：pg 专属类型（serial/bytea/numeric）转换 + pg 字符串（反斜杠原样语义）
//     经 mysql 方言导出（反斜杠转义语义）后的转义正确性
//   - sqlite 源 → pg / mysql 目标：sqlite 动态类型列经 ConvToTargetDbColumn 的转换与回环
//
// 期望值直接取源库回读结果（readAllRows），由 assertCell 做跨方言值形态宽松归一比对。
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/dbm/

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// setupPgSourceTable 建立pg反向迁移源表并填充数据（含引号/反斜杠/中文/emoji/bytea/numeric）
func setupPgSourceTable(t *testing.T, conn *dbi.DbConn, table string) []map[string]any {
	t.Helper()
	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote(table))
	mustExec(t, conn, "CREATE TABLE "+quote(table)+" (id serial PRIMARY KEY, v_varchar varchar(128), v_num numeric(12,2), v_bytea bytea, v_text text)")
	mustExec(t, conn, "INSERT INTO "+quote(table)+" (v_varchar, v_num, v_bytea, v_text) VALUES "+
		`('it''s a;b', 123.45, '\x48656c6c6f', 'back\slash''quote'),`+
		`('中文🙂', -99.5, '\x00ff10', '多行' || chr(10) || 'text')`)
	return readAllRows(t, conn, table, "id")
}

// TestITPgToMysqlMigration pg源 → mysql目标异构迁移
func TestITPgToMysqlMigration(t *testing.T) {
	pconn := pgConn(t)
	defer pconn.Close()
	mconn := mysqlConn(t)
	defer mconn.Close()

	srcTable := "it_rvmig_pg_src"
	expectRows := setupPgSourceTable(t, pconn, srcTable)
	script := dumpTableScript(t, pconn, srcTable, mconn.GetDialect(), "mysql")

	// 结构断言：serial应转mysql自增
	assert.Contains(t, script, "AUTO_INCREMENT", "pg serial应转mysql AUTO_INCREMENT")

	// 导入真实mysql（脚本含DROP TABLE IF EXISTS，自动清理上次运行残留）
	execStmtsInTx(t, mconn, strings.NewReader(script))
	actualRows := readAllRows(t, mconn, srcTable, "id")
	require.Len(t, actualRows, len(expectRows))
	for i, expect := range expectRows {
		assertRow(t, expect, actualRows[i], i)
	}
}

// TestITPgToSqliteMigration pg源 → sqlite目标异构迁移
func TestITPgToSqliteMigration(t *testing.T) {
	pconn := pgConn(t)
	defer pconn.Close()
	sconn := sqliteConn(t)

	srcTable := "it_rvmig_pg_src"
	expectRows := setupPgSourceTable(t, pconn, srcTable)
	script := dumpTableScript(t, pconn, srcTable, sconn.GetDialect(), "sqlite")

	execStmtsInTx(t, sconn, strings.NewReader(script))
	actualRows := readAllRows(t, sconn, srcTable, "id")
	require.Len(t, actualRows, len(expectRows))
	for i, expect := range expectRows {
		assertRow(t, expect, actualRows[i], i)
	}
}

// setupSqliteSourceTable 建立sqlite反向迁移源表并填充数据
func setupSqliteSourceTable(t *testing.T, conn *dbi.DbConn, table string) []map[string]any {
	t.Helper()
	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote(table))
	mustExec(t, conn, "CREATE TABLE "+quote(table)+" (id integer PRIMARY KEY, v_varchar text, v_num numeric, v_blob blob, v_text text)")
	mustExec(t, conn, "INSERT INTO "+quote(table)+" (v_varchar, v_num, v_blob, v_text) VALUES "+
		`('it''s a;b', 123.45, x'48656c6c6f', 'back\slash''quote'),`+
		`('中文🙂', -99.5, x'00ff10', '多行' || char(10) || 'text')`)
	return readAllRows(t, conn, table, "id")
}

// TestITSqliteToPgMigration sqlite源 → pg目标异构迁移
func TestITSqliteToPgMigration(t *testing.T) {
	sconn := sqliteConn(t)
	pconn := pgConn(t)
	defer pconn.Close()

	srcTable := "it_rvmig_sq_src"
	expectRows := setupSqliteSourceTable(t, sconn, srcTable)
	script := dumpTableScript(t, sconn, srcTable, pconn.GetDialect(), "postgres")

	// 结构断言：sqlite integer primary key 应转 pg serial 自增
	assert.Contains(t, script, "serial", "sqlite自增主键应转pg serial")

	execStmtsInTx(t, pconn, strings.NewReader(script))
	actualRows := readAllRows(t, pconn, srcTable, "id")
	require.Len(t, actualRows, len(expectRows))
	for i, expect := range expectRows {
		assertRow(t, expect, actualRows[i], i)
	}
}

// TestITSqliteToMysqlMigration sqlite源 → mysql目标异构迁移
func TestITSqliteToMysqlMigration(t *testing.T) {
	sconn := sqliteConn(t)
	mconn := mysqlConn(t)
	defer mconn.Close()

	srcTable := "it_rvmig_sq_src"
	expectRows := setupSqliteSourceTable(t, sconn, srcTable)
	script := dumpTableScript(t, sconn, srcTable, mconn.GetDialect(), "mysql")

	execStmtsInTx(t, mconn, strings.NewReader(script))
	actualRows := readAllRows(t, mconn, srcTable, "id")
	require.Len(t, actualRows, len(expectRows))
	for i, expect := range expectRows {
		assertRow(t, expect, actualRows[i], i)
	}
}
