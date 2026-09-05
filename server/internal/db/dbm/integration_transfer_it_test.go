//go:build it

package dbm

// dbm 导出/导入/SQL执行链路集成测试：
//   - 导出：模拟DumpDb核心逻辑（元数据→类型转换→DDL+Insert SQL文本生成）
//   - 导入：模拟transfer2Db/ExecReader核心逻辑（方言切割器切割→事务内逐条执行）
//   - SQL执行：模拟ExecuteSql分发链路（切割→解析→按Stmt类型分发查询/执行）
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/dbm/

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// dumpTableScript 模拟DumpDb核心导出逻辑：注释头 + DDL（dropBeforeCreate）+ 数据Insert + DumpHelper包装
func dumpTableScript(t *testing.T, conn *dbi.DbConn, table string, targetDialect dbi.Dialect, targetDbType dbi.DbType) string {
	t.Helper()
	srcMeta := conn.GetMetadata()
	srcDialect := conn.GetDialect()

	tbs, err := srcMeta.GetTables(table)
	require.NoError(t, err)
	require.NotEmpty(t, tbs, "source table [%s] not found", table)
	srcCols, err := srcMeta.GetColumns(table)
	require.NoError(t, err)

	// 类型转换到目标方言（与DumpDb一致，同库导出也走该链路）
	cols := make([]dbi.Column, 0, len(srcCols))
	for _, col := range srcCols {
		if col.TableName != table {
			continue
		}
		require.NoError(t, dbi.ConvToTargetDbColumn(conn.Info.Type, targetDbType, targetDialect, &col),
			"convert column [%s] failed", col.ColumnName)
		cols = append(cols, col)
	}
	require.NotEmpty(t, cols)

	var sb strings.Builder
	sb.WriteString("-- ----------------------------\n-- Table structure: " + table + " \n-- ----------------------------\n")
	gen := targetDialect.GetSQLGenerator()
	for _, ddl := range gen.GenTableDDL(tbs[0], cols, true) {
		sb.WriteString(ddl + ";\n")
	}

	sb.WriteString("\n-- ----------------------------\n-- Data: " + table + " \n-- ----------------------------\n")
	dumpHelper := targetDialect.GetDumpHelper()
	require.NoError(t, dumpHelper.BeforeInsert(&sb, table))

	var rows [][]any
	_, err = conn.WalkTableRows(itCtx(), srcDialect.Quoter().Quote(table), func(row map[string]any, _ []*dbi.QueryColumn) error {
		vals := make([]any, len(cols))
		for i, c := range cols {
			vals[i] = row[c.ColumnName]
		}
		rows = append(rows, vals)
		return nil
	})
	require.NoError(t, err)
	if len(rows) > 0 {
		insertSqls := gen.GenInsert(table, cols, rows, dbi.DuplicateStrategyNone, nil)
		sb.WriteString(strings.Join(insertSqls, ";\n") + ";\n")
	}
	require.NoError(t, dumpHelper.AfterInsert(&sb, table, cols))

	return sb.String()
}

// importScript 模拟transfer2Db/ExecReader核心导入逻辑：方言切割器切割→事务内逐条执行
// 返回执行的语句数
func importScript(t *testing.T, conn *dbi.DbConn, script io.Reader) int {
	t.Helper()
	tx, err := conn.Begin()
	require.NoError(t, err)

	stmtCount := 0
	err = conn.GetDialect().GetSQLSplitter().SplitSQL(script, func(stmt string) error {
		stmtCount++
		if _, err := conn.TxExec(tx, stmt); err != nil {
			// 语句可能很大（大值批量INSERT），错误信息中截断避免刷屏
			stmtBrief := stmt
			if len(stmtBrief) > 200 {
				stmtBrief = stmtBrief[:200] + "..."
			}
			return fmt.Errorf("exec stmt [%s] failed: %w", stmtBrief, err)
		}
		return nil
	})
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("import script failed: %s", err.Error())
	}
	require.NoError(t, tx.Commit())
	return stmtCount
}

// parseStmtType 解析SQL返回Stmt具体类型名（模拟ExecuteSql的解析分发判断）
func parseStmtType(t *testing.T, conn *dbi.DbConn, sql string) string {
	t.Helper()
	stmt, err := conn.GetDialect().GetSQLParser().Parse(sql)
	require.NoError(t, err, "parse sql [%s] failed", sql)
	require.NotNil(t, stmt)
	return fmt.Sprintf("%T", stmt)
}

// execDispatched 模拟ExecuteSql分发链路：切割→解析→按Stmt类型分发查询/执行
func execDispatched(t *testing.T, conn *dbi.DbConn, script string) {
	t.Helper()
	splitter := conn.GetDialect().GetSQLSplitter()
	parser := conn.GetDialect().GetSQLParser()
	require.NoError(t, splitter.SplitSQL(strings.NewReader(script), func(oneSql string) error {
		stmt, parseErr := parser.Parse(oneSql)
		if parseErr != nil {
			t.Fatalf("parse [%s] failed: %s", oneSql, parseErr.Error())
		}
		switch stmt.(type) {
		case *sqlstmt.SelectStmt, *sqlstmt.WithStmt, *sqlstmt.OtherStmt:
			_, _, err := conn.Query(oneSql)
			return err
		default:
			_, err := conn.Exec(oneSql)
			return err
		}
	}))
}

// ---------------------------------------------------------------------
// 导出 → 导入
// ---------------------------------------------------------------------

// mysql同库导出再导入（等价 transfer2Db 同构迁移链路：导出SQL文本→切割→事务执行）
// 注意：mysql默认DumpHelper会在数据前后输出 BEGIN;/COMMIT;，同样作为语句经TxExec执行，
// 本用例同时验证真实库对该行为的兼容性
func TestITMysqlDumpReimport(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	_, expectRows := setupMysqlSourceTable(t, conn, "it_dump_src")
	script := dumpTableScript(t, conn, "it_dump_src", conn.GetDialect(), "mysql")

	// 导出产物结构断言：注释/DDL/事务包装/插入语句
	assert.Contains(t, script, "-- Table structure: it_dump_src")
	assert.Contains(t, script, "DROP TABLE IF EXISTS")
	assert.Contains(t, script, "CREATE TABLE")
	assert.Contains(t, script, "BEGIN;")
	assert.Contains(t, script, "INSERT INTO")
	assert.Contains(t, script, "COMMIT;")
	assert.Contains(t, script, "it''s") // 默认值单引号转义

	// 导入回环：表被DROP重建后数据应与导出前完全一致
	importScript(t, conn, strings.NewReader(script))
	actualRows := readAllRows(t, conn, "it_dump_src", "id")
	require.Len(t, actualRows, len(expectRows))
	for i, expect := range expectRows {
		assertRow(t, expect, actualRows[i], i)
	}
}

// mysql导出为sqlite方言SQL文本 → sqlite切割导入（等价 transfer2File 文件迁移 + 用户下载后导入执行链路）
func TestITMysqlDumpToSqliteFileMigration(t *testing.T) {
	mconn := mysqlConn(t)
	defer mconn.Close()
	sconn := sqliteConn(t)
	defer sconn.Close()

	_, expectRows := setupMysqlSourceTable(t, mconn, "it_dump_file_src")
	script := dumpTableScript(t, mconn, "it_dump_file_src", sconn.GetDialect(), "sqlite")

	// sqlite方言产物断言
	assert.Contains(t, script, "DROP TABLE IF EXISTS")
	assert.Contains(t, script, "CREATE TABLE")
	assert.Contains(t, script, "INSERT INTO")
	// sqlite无 BEGIN;/COMMIT; 包装（其DumpHelper已覆写为空实现）
	assert.NotContains(t, script, "BEGIN;")

	importScript(t, sconn, strings.NewReader(script))
	actualRows := readAllRows(t, sconn, "it_dump_file_src", "id")
	require.Len(t, actualRows, len(expectRows))
	for i, expect := range expectRows {
		assertRow(t, expect, actualRows[i], i)
	}
}

// sqlite脚本含反斜杠字面量值：标准SQL切割语义下的真实执行（切割器方言修复的端到端验证）
func TestITSqliteScriptBackslashStmts(t *testing.T) {
	sconn := sqliteConn(t)
	defer sconn.Close()

	quote := sconn.GetDialect().Quoter().Quote
	mustExec(t, sconn, fmt.Sprintf("DROP TABLE IF EXISTS %s", quote("it_bs")))
	script := fmt.Sprintf(`create table %s (id integer primary key, val text);
insert into %s values (1, '\');
insert into %s values (2, 'x\;y');`,
		quote("it_bs"), quote("it_bs"), quote("it_bs"))

	stmtCount := importScript(t, sconn, strings.NewReader(script))
	require.Equal(t, 3, stmtCount, "sqlite标准语义下应切出3条语句")

	rows := readAllRows(t, sconn, "it_bs", "id")
	require.Len(t, rows, 2)
	assert.Equal(t, `\`, normalizeDbValue(rows[0]["val"]))
	assert.Equal(t, `x\;y`, normalizeDbValue(rows[1]["val"]))
}

// ---------------------------------------------------------------------
// SQL 执行分发链路（切割 → 解析 → 按类型分发）
// ---------------------------------------------------------------------

func TestITMysqlSqlExecDispatch(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	// 解析类型断言（模拟ExecuteSql的Stmt类型分发判断）
	assert.Equal(t, "*sqlstmt.DdlStmt", parseStmtType(t, conn, "create table `it_exec` (id int primary key, val varchar(100))"))
	assert.Equal(t, "*sqlstmt.InsertStmt", parseStmtType(t, conn, "insert into `it_exec` values (1, 'a')"))
	assert.Equal(t, "*sqlstmt.UpdateStmt", parseStmtType(t, conn, "update `it_exec` set val = 'b' where id = 1"))
	assert.Equal(t, "*sqlstmt.DeleteStmt", parseStmtType(t, conn, "delete from `it_exec` where id = 1"))
	assert.Equal(t, "*sqlstmt.SelectStmt", parseStmtType(t, conn, "select * from `it_exec`"))

	// 全链路：切割（值内分号/引号不干扰）→ 解析 → 分发执行
	mustExec(t, conn, "DROP TABLE IF EXISTS `it_exec`")
	script := "create table `it_exec` (id int primary key, val varchar(100));\n" +
		"-- 插入含分号与引号的值\n" +
		"insert into `it_exec` values (1, 'a;b''c'), (2, '中文🙂#哈');\n" +
		"# mysql井号注释\n" +
		"update `it_exec` set val = 'u;p''q' where id = 2;\n" +
		"delete from `it_exec` where id = 1;\n" +
		"select * from `it_exec`;\n"
	execDispatched(t, conn, script)

	rows := readAllRows(t, conn, "it_exec", "id")
	require.Len(t, rows, 1)
	assert.Equal(t, "u;p'q", normalizeDbValue(rows[0]["val"]))
}

func TestITSqliteSqlExecDispatch(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()

	// sqlite沿用默认pgsql解析器（与标准SQL最接近）
	assert.Equal(t, "*sqlstmt.DdlStmt", parseStmtType(t, conn, `create table "it_exec_s" (id integer primary key, val text)`))
	assert.Equal(t, "*sqlstmt.InsertStmt", parseStmtType(t, conn, `insert into "it_exec_s" values (1, 'a')`))
	assert.Equal(t, "*sqlstmt.UpdateStmt", parseStmtType(t, conn, `update "it_exec_s" set val = 'b' where id = 1`))
	assert.Equal(t, "*sqlstmt.SelectStmt", parseStmtType(t, conn, `select * from "it_exec_s"`))

	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, fmt.Sprintf("DROP TABLE IF EXISTS %s", quote("it_exec_s")))
	script := fmt.Sprintf(`create table %s (id integer primary key, val text);
-- 插入含分号的值
insert into %s values (1, 'a;b'), (2, '中文🙂');
update %s set val = 'u;p' where id = 2;
select count(*) from %s;`,
		quote("it_exec_s"), quote("it_exec_s"), quote("it_exec_s"), quote("it_exec_s"))
	execDispatched(t, conn, script)

	rows := readAllRows(t, conn, "it_exec_s", "id")
	require.Len(t, rows, 2)
	assert.Equal(t, "a;b", normalizeDbValue(rows[0]["val"]))
	assert.Equal(t, "u;p", normalizeDbValue(rows[1]["val"]))
}
