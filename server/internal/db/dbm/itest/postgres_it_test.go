package itest

// postgres 全流程集成测试：基于本地Docker PostgreSQL 16（localhost:5432，postgres/postgres），
// 验证 连接→元数据→DDL生成执行→转义数据读写→复制表→导出导入→异构迁移→SQL执行分发 的真实业务链路。
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/dbm/
//
// docker run -d --name mayfly-pg-it -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=mayfly_pg_it -p 5432:5432 postgres:16

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/postgres" // 注册postgres方言
)

// ---------------------------------------------------------------------
// 连接与元数据
// ---------------------------------------------------------------------

func TestITPgConnectAndMetadata(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()

	server, err := conn.GetMetadata().GetDbServer()
	require.NoError(t, err)
	t.Logf("postgres server version: %+v", server)
	assert.Contains(t, server.Version, "PostgreSQL")

	dbs, err := conn.GetMetadata().GetDbNames()
	require.NoError(t, err)
	assert.Contains(t, dbs, itPgDatabase)

	tables, err := conn.GetMetadata().GetTables()
	require.NoError(t, err)
	t.Logf("tables in %s: %d", itPgDatabase, len(tables))
}

// DDL生成→真实执行→元数据回读→再用回读元数据二次建表（元数据↔DDL双向闭环）
func TestITPgGenTableDDLRoundtrip(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()

	dialect := conn.GetDialect()
	gen := dialect.GetSQLGenerator()
	quote := dialect.Quoter().Quote
	table := "it_pg_ddl_roundtrip"
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote(table))

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int4", IsPrimaryKey: true, AutoIncrement: true},
		{ColumnName: "name", DataType: "varchar", CharMaxLength: 128, Nullable: false, ColumnDefault: "it's"},
		{ColumnName: "code", DataType: "char", CharMaxLength: 32, Nullable: true},
		{ColumnName: "remark", DataType: "text", Nullable: true},
		{ColumnName: "amount", DataType: "numeric", NumPrecision: 10, NumScale: 2},
		{ColumnName: "created", DataType: "timestamp", Nullable: true},
		{ColumnName: "ext", DataType: "jsonb", Nullable: true},
		{ColumnName: "bin", DataType: "bytea", Nullable: true},
	}

	for _, ddl := range gen.GenTableDDL(dbi.Table{TableName: table, TableComment: "表'注'释"}, columns, false) {
		mustExec(t, conn, ddl)
	}

	// 真实回读元数据，逐项断言DDL执行结果符合预期
	readCols, err := conn.GetMetadata().GetColumns(table)
	require.NoError(t, err)
	require.Len(t, readCols, 8)
	byName := make(map[string]dbi.Column)
	for _, col := range readCols {
		byName[col.ColumnName] = col
	}

	id := byName["id"]
	assert.True(t, id.AutoIncrement, "serial自增应被元数据识别")
	assert.True(t, id.IsPrimaryKey)

	name := byName["name"]
	assert.Equal(t, "varchar", name.DataType)
	assert.Equal(t, 128, name.CharMaxLength)
	assert.False(t, name.Nullable)
	// pg的column_default保留字面量书写形态（剥去::cast但保留引号与双写转义），
	// 与无默认值（空串）可区分，且内容含括号/引号的默认值不会被误判为表达式而丢失
	assert.Equal(t, "'it''s'", name.ColumnDefault, "元数据以字面量形态呈现默认值")
	// 语义验证：插入时省略name（其余非空列显式给值），库内必须真正落入转义后的原始默认值
	mustExec(t, conn, "INSERT INTO "+quote(table)+"(id, amount) VALUES (1, 0)")
	_, insRows, err := conn.Query("SELECT name FROM " + quote(table))
	require.NoError(t, err)
	require.Len(t, insRows, 1)
	assert.Equal(t, "it's", fmt.Sprint(insRows[0]["name"]), "默认值实际生效且未被转义失真")
	mustExec(t, conn, "DELETE FROM "+quote(table))

	amount := byName["amount"]
	assert.Equal(t, 10, amount.NumPrecision)
	assert.Equal(t, 2, amount.NumScale)

	assert.True(t, byName["code"].Nullable)
	assert.Equal(t, 32, byName["code"].CharMaxLength)

	// 表注释转义正确落地
	tbs, err := conn.GetMetadata().GetTables(table)
	require.NoError(t, err)
	require.NotEmpty(t, tbs)
	assert.Equal(t, "表'注'释", tbs[0].TableComment)

	// 用真实回读的元数据再次生成DDL并建表（dropBeforeCreate），验证元数据→DDL闭环
	for _, ddl := range gen.GenTableDDL(dbi.Table{TableName: table}, readCols, true) {
		mustExec(t, conn, ddl)
	}
	readCols2, err := conn.GetMetadata().GetColumns(table)
	require.NoError(t, err)
	require.Len(t, readCols2, 8)
	assert.True(t, byName2AutoIncrement(readCols2))
	// 二次建表后默认值必须仍在（不会因元数据形态被误判为表达式而静默丢失）
	for _, col := range readCols2 {
		if col.ColumnName == "name" {
			assert.Equal(t, "'it''s'", col.ColumnDefault, "元数据→DDL→元数据往返默认值不丢失")
		}
	}
}

func byName2AutoIncrement(cols []dbi.Column) bool {
	for _, col := range cols {
		if col.ColumnName == "id" {
			return col.AutoIncrement
		}
	}
	return false
}

// ---------------------------------------------------------------------
// 转义数据真实读写回环
// ---------------------------------------------------------------------

func TestITPgEscapeDataRoundtrip(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()

	quote := conn.GetDialect().Quoter().Quote
	table := "it_pg_escape"
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote(table))
	mustExec(t, conn, fmt.Sprintf(`CREATE TABLE %s (
		id int PRIMARY KEY,
		v_varchar varchar(128),
		v_text text,
		v_dec numeric(10,2),
		v_ts timestamp,
		v_bytea bytea
	)`, quote(table)))

	columns, err := conn.GetMetadata().GetColumns(table)
	require.NoError(t, err)
	require.Len(t, columns, 6)

	// 期望数据：单引号/反斜杠/双引号/分号/换行tab/中文/emoji/NULL/空串/二进制(hex形态)
	expectRows := []map[string]any{
		{
			"id": 1, "v_varchar": "it's", "v_text": "line1\nline2\tend",
			"v_dec": "123.45", "v_ts": "2026-01-02 03:04:05",
			"v_bytea": hexEncode([]byte("bin data")),
		},
		{
			"id": 2, "v_varchar": `a\b;c`, "v_text": `say "hi"`,
			"v_dec": "-99.99", "v_ts": nil, "v_bytea": nil,
		},
		{
			"id": 3, "v_varchar": "中文'单引'🙂", "v_text": "",
			"v_dec": "0.01", "v_ts": "2026-12-31 23:59:59",
			"v_bytea": hexEncode([]byte("blob'quote\\slash")),
		},
	}

	// 通过GenInsert生成插入SQL并真实执行（走完整的值转义链路）
	gen := conn.GetDialect().GetSQLGenerator()
	values := make([][]any, 0, len(expectRows))
	for _, row := range expectRows {
		rowVals := make([]any, 0, len(columns))
		for _, col := range columns {
			rowVals = append(rowVals, row[col.ColumnName])
		}
		values = append(values, rowVals)
	}
	for _, sql := range gen.GenInsert(table, columns, values, dbi.DuplicateStrategyNone, nil) {
		mustExec(t, conn, sql)
	}

	actualRows := readAllRows(t, conn, table, "id")
	require.Len(t, actualRows, len(expectRows))
	for i, expect := range expectRows {
		assertRow(t, expect, actualRows[i], i)
	}

	// 冲突策略 ON CONFLICT DO UPDATE 真实执行
	conflictValues := [][]any{{1, "updated", "u", "1.00", nil, nil}}
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	for _, sql := range gen.GenInsert(table, columns, conflictValues, dbi.DuplicateStrategyUpdate, meta) {
		mustExec(t, conn, sql)
	}
	rows := readAllRows(t, conn, table, "id")
	require.Len(t, rows, 3)
	assert.Equal(t, "updated", normalizeDbValue(rows[0]["v_varchar"]))
}

func hexEncode(b []byte) string {
	return fmt.Sprintf("%x", b)
}

// ---------------------------------------------------------------------
// 复制表
// ---------------------------------------------------------------------

func TestITPgCopyTable(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()

	table := "it_pg_copy_src"
	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote(table))
	// copy表名带时间戳后缀，清理历史残留避免前缀匹配到旧表
	_, staleRes, err := conn.Query("SELECT tablename FROM pg_tables WHERE tablename LIKE $1", table+"_copy_%")
	require.NoError(t, err)
	for _, re := range staleRes {
		mustExec(t, conn, fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(fmt.Sprintf("%v", re["tablename"]))))
	}
	// serial自增表：CopyTable需要重建序列，覆盖该复杂链路
	mustExec(t, conn, fmt.Sprintf("CREATE TABLE %s (id serial PRIMARY KEY, val varchar(50))", quote(table)))
	mustExec(t, conn, fmt.Sprintf("INSERT INTO %s (val) VALUES ('a'), ('b'), ('中文🙂')", quote(table)))

	require.NoError(t, conn.GetDialect().CopyTable(&dbi.DbCopyTable{TableName: table, CopyData: true}))
	time.Sleep(2 * time.Second) // 数据为异步复制

	tables, err := conn.GetMetadata().GetTables()
	require.NoError(t, err)
	var copyName string
	for _, tb := range tables {
		if strings.HasPrefix(tb.TableName, table+"_copy_") {
			copyName = tb.TableName
			break
		}
	}
	require.NotEmpty(t, copyName, "copy table not found")

	copyRows := readAllRows(t, conn, copyName, "id")
	require.Len(t, copyRows, 3)
	assert.Equal(t, "中文🙂", normalizeDbValue(copyRows[2]["val"]))

	// 复制表的序列应已重建：继续插入自增不冲突
	mustExec(t, conn, fmt.Sprintf("INSERT INTO %s (val) VALUES ('after-copy')", quote(copyName)))
	rows := readAllRows(t, conn, copyName, "id")
	require.Len(t, rows, 4)
}

// ---------------------------------------------------------------------
// 导出 → 导入
// ---------------------------------------------------------------------

// pg同库导出再导入（等价 transfer2Db 同构迁移链路），含 AfterInsert 的 setval 序列校正与 COMMIT
func TestITPgDumpReimport(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()

	table := "it_pg_dump_src"
	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote(table))
	mustExec(t, conn, fmt.Sprintf(`CREATE TABLE %s (id serial PRIMARY KEY, v_varchar varchar(128) DEFAULT 'it''s', v_text text)`, quote(table)))
	mustExec(t, conn, fmt.Sprintf(`INSERT INTO %s (v_varchar, v_text) VALUES ('a;b', 'it''s'), ('中文🙂', 'x\y')`, quote(table)))

	script := dumpTableScript(t, conn, table, conn.GetDialect(), "postgres")

	// 导出产物结构断言：DDL + 注释 + setval序列校正；BEGIN/COMMIT不输出（导入方自管事务）
	assert.Contains(t, script, "DROP TABLE IF EXISTS")
	assert.Contains(t, script, "CREATE TABLE")
	assert.Contains(t, script, "INSERT INTO")
	assert.Contains(t, script, "setval")
	assert.NotContains(t, script, "COMMIT;")
	assert.Contains(t, script, "it''s")

	// 导入回环
	execStmtsInTx(t, conn, strings.NewReader(script))
	rows := readAllRows(t, conn, table, "id")
	require.Len(t, rows, 2)
	assert.Equal(t, "a;b", normalizeDbValue(rows[0]["v_varchar"]))
	assert.Equal(t, "it's", normalizeDbValue(rows[0]["v_text"]))
	assert.Equal(t, "中文🙂", normalizeDbValue(rows[1]["v_varchar"]))

	// setval校正后自增不冲突
	mustExec(t, conn, fmt.Sprintf("INSERT INTO %s (v_varchar) VALUES ('after')", quote(table)))
	rows = readAllRows(t, conn, table, "id")
	require.Len(t, rows, 3)
}

// mysql导出为postgres方言SQL文本 → pg切割导入（异构文件迁移链路）
func TestITMysqlToPgMigration(t *testing.T) {
	mconn := mysqlConn(t)
	defer mconn.Close()
	pconn := pgConn(t)
	defer pconn.Close()

	srcTable := "it_mig_pg_src"
	_, expectRows := setupMysqlSourceTable(t, mconn, srcTable)
	script := dumpTableScript(t, mconn, srcTable, pconn.GetDialect(), "postgres")

	// unsigned应升位转换避免溢出（v_uint=4294967295超int4上限）
	assert.Contains(t, script, "int8", "int unsigned应升位为int8")
	assert.Contains(t, script, "numeric", "bigint unsigned应转为numeric")

	execStmtsInTx(t, pconn, strings.NewReader(script))
	pgQuote := pconn.GetDialect().Quoter().Quote
	actualRows := readAllRows(t, pconn, srcTable, "id")
	require.Len(t, actualRows, len(expectRows))
	for i, expect := range expectRows {
		assertRow(t, expect, actualRows[i], i)
	}
	_ = pgQuote
}

// ---------------------------------------------------------------------
// SQL 执行分发链路（切割 → 解析 → 按类型分发）
// ---------------------------------------------------------------------

func TestITPgSqlExecDispatch(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()

	// 解析类型断言
	assert.Equal(t, "*sqlstmt.DdlStmt", parseStmtType(t, conn, `create table "it_pg_exec" (id int primary key, val text)`))
	assert.Equal(t, "*sqlstmt.InsertStmt", parseStmtType(t, conn, `insert into "it_pg_exec" values (1, 'a')`))
	assert.Equal(t, "*sqlstmt.UpdateStmt", parseStmtType(t, conn, `update "it_pg_exec" set val = 'b' where id = 1`))
	assert.Equal(t, "*sqlstmt.DeleteStmt", parseStmtType(t, conn, `delete from "it_pg_exec" where id = 1`))
	assert.Equal(t, "*sqlstmt.SelectStmt", parseStmtType(t, conn, `select * from "it_pg_exec"`))

	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote("it_pg_exec"))

	// 全链路脚本：普通语句 + DO $$ 块（体内分号不切割）+ 函数定义
	script := `create table "it_pg_exec" (id serial primary key, val text);
-- 插入含分号与引号的值
insert into "it_pg_exec" (val) values ('a;b''c'), ('中文🙂'), (E'esc\'aped;x');
DO $$ BEGIN
    update "it_pg_exec" set val = 'do;block' where val like 'a;b%';
END $$;
select * from "it_pg_exec";`
	execDispatched(t, conn, script)

	rows := readAllRows(t, conn, "it_pg_exec", "id")
	require.Len(t, rows, 3)
	// DO块真实执行：a;b'c 被更新为 do;block
	assert.Equal(t, "do;block", normalizeDbValue(rows[0]["val"]))
	assert.Equal(t, "中文🙂", normalizeDbValue(rows[1]["val"]))
	assert.Equal(t, `esc'aped;x`, normalizeDbValue(rows[2]["val"]), "E-string转义值应真实写入")

	// 函数定义脚本（体内多个分号+注释）：独立切割器的核心价值验证
	fnScript := `CREATE OR REPLACE FUNCTION it_pg_fn_add(a integer) RETURNS integer AS $fn$
BEGIN
    RETURN a + 1; -- body; comment
END;
$fn$ LANGUAGE plpgsql;`
	mustExec(t, conn, "DROP FUNCTION IF EXISTS it_pg_fn_add(integer)")
	stmtCount := execStmtsInTx(t, conn, strings.NewReader(fnScript+";\nselect it_pg_fn_add(41) as res;"))
	require.Equal(t, 2, stmtCount, "函数定义与select应切为2条语句")
	_, res, err := conn.Query("select it_pg_fn_add(41) as res")
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "42", fmt.Sprintf("%v", normalizeDbValue(res[0]["res"])))
}
