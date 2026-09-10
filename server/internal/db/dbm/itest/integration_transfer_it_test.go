package itest

// dbm SQL生成器与切割器的真实库集成测试：
//   - execStmtsInTx/execDispatched 在真实数据库上执行手写或生成器产物，验证切割/转义/类型映射
//   - 导入/导出**链路**（DumpDbScript编排、importDumpStream的事务控制语句过滤与批级提交）的
//     权威集成测试在 transfer 包（驱动真实函数），本包不重复验证也不复刻其逻辑
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

// dumpTableScript 组装“表结构DDL + 数据INSERT + DumpHelper包装”脚本：逐环节调用生产生成器
// （ConvToTargetDbColumn/GenTableDDL/GenInsert/BeforeInsert/AfterInsert），用于验证**列类型映射与转义**。
//
// 它不等同于生产DumpDbScript（未复刻其分批预算与多表编排），导出链路的权威IT在transfer包
// 驱动真实DumpDbScript（db_dump_batch_it_test.go / db_transfer_whole_db_it_test.go）
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

// execStmtsInTx 在事务内逐条执行脚本语句（经真实方言切割器切割）。
//
// 仅作“生成器产物能被真实库执行”的验证载体：不含生产导入链路的注释头/事务控制语句过滤与
// 批级提交（dumpTableScript产物里的 BEGIN;/COMMIT;在此会被当作真实语句执行，mysql上会隐式提交），
// 因此本文件与使用它的用例均**不能**用于断言导入链路行为；导入链路的权威IT在transfer包
// 驱动真实importDumpStream（db_transfer_import_it_test.go / db_transfer_whole_db_it_test.go）
// 返回执行的语句数
func execStmtsInTx(t *testing.T, conn *dbi.DbConn, script io.Reader) int {
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
// 生成器产物在真实库上的执行（切割/转义/事务包装兼容性的语句层验证）
// ---------------------------------------------------------------------

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

	stmtCount := execStmtsInTx(t, sconn, strings.NewReader(script))
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
