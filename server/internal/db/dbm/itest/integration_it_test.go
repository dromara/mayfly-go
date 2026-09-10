package itest

// dbm 全流程集成测试：基于本机真实MySQL（localhost:3306，见server/config.yml）与SQLite，
// 验证 连接→元数据→DDL生成执行→转义数据读写→冲突策略→复制表→异构迁移 的真实业务链路。
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/dbm/
//
// 测试库 mayfly_dbm_it 自动创建，表以 it_ 前缀命名并在用例开始时清理。

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/mysql"  // 注册mysql方言
	_ "mayfly-go/internal/db/dbm/sqlite" // 注册sqlite方言
)

// ---------------------------------------------------------------------
// MySQL 全流程
// ---------------------------------------------------------------------

func TestITMysqlConnectAndMetadata(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	server, err := conn.GetMetadata().GetDbServer()
	require.NoError(t, err)
	t.Logf("mysql server version: %+v", server)

	tables, err := conn.GetMetadata().GetTables()
	require.NoError(t, err)
	t.Logf("tables in %s: %d", itMysqlDatabase, len(tables))
}

// DDL生成→真实执行→元数据回读→再用回读元数据生成DDL二次建表（元数据↔DDL双向闭环）
func TestITMysqlGenTableDDLRoundtrip(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	dialect := conn.GetDialect()
	gen := dialect.GetSQLGenerator()
	quote := dialect.Quoter().Quote
	table := "it_ddl_roundtrip"
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote(table))

	// 与此前修复点对应的断言目标：默认值单引号转义（DEFAULT 'it''s'）、注释转义
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true},
		{ColumnName: "name", DataType: "varchar", CharMaxLength: 128, Nullable: false, ColumnDefault: "it's", ColumnComment: "姓'名'"},
		{ColumnName: "code", DataType: "char", CharMaxLength: 32, Nullable: true},
		{ColumnName: "remark", DataType: "text", Nullable: true},
		{ColumnName: "amount", DataType: "decimal", NumPrecision: 10, NumScale: 2},
		{ColumnName: "cnt", DataType: "unsigned int"},
		{ColumnName: "created", DataType: "datetime", Nullable: true},
	}

	for _, ddl := range gen.GenTableDDL(dbi.Table{TableName: table}, columns, false) {
		mustExec(t, conn, ddl)
	}

	// 真实回读元数据，逐项断言DDL执行结果符合预期
	readCols, err := conn.GetMetadata().GetColumns(table)
	require.NoError(t, err)
	require.Len(t, readCols, 7)
	byName := make(map[string]dbi.Column)
	for _, col := range readCols {
		byName[col.ColumnName] = col
	}

	name := byName["name"]
	assert.Equal(t, "varchar", name.DataType)
	assert.Equal(t, 128, name.CharMaxLength)
	assert.False(t, name.Nullable)
	assert.Equal(t, "it's", name.ColumnDefault) // 默认值转义正确落地（无多余引号）
	assert.Equal(t, "姓'名'", name.ColumnComment) // 注释转义正确落地

	cnt := byName["cnt"]
	// unsigned信息修复回归：data_type不带unsigned后缀，需归一化为"unsigned int"供迁移链路匹配
	assert.Equal(t, "unsigned int", cnt.DataType)
	assert.Equal(t, "int unsigned", cnt.ColumnType)

	amount := byName["amount"]
	assert.Equal(t, 10, amount.NumPrecision)
	assert.Equal(t, 2, amount.NumScale)

	code := byName["code"]
	assert.Equal(t, 32, code.CharMaxLength)
	assert.True(t, code.Nullable)

	// 用真实回读的元数据再次生成DDL并建表，验证元数据→DDL闭环（GetColumnType重组、FixColumn后无脏数据）
	table2 := table + "_2"
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote(table2))
	for _, ddl := range gen.GenTableDDL(dbi.Table{TableName: table2}, readCols, false) {
		mustExec(t, conn, ddl)
	}
	readCols2, err := conn.GetMetadata().GetColumns(table2)
	require.NoError(t, err)
	require.Len(t, readCols2, 7)
	byName2 := make(map[string]dbi.Column)
	for _, col := range readCols2 {
		byName2[col.ColumnName] = col
	}
	assert.Equal(t, "it's", byName2["name"].ColumnDefault)
	assert.Equal(t, "unsigned int", byName2["cnt"].DataType)
	assert.Equal(t, 128, byName2["name"].CharMaxLength)
}

// 转义数据真实读写回环：单引号/反斜杠/双引号/换行tab/中文/emoji/NULL/空串/二进制
func TestITMysqlEscapeDataRoundtrip(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	columns, expectRows := setupMysqlSourceTable(t, conn, "it_escape")
	actualRows := readAllRows(t, conn, "it_escape", "id")
	require.Len(t, actualRows, len(expectRows))

	for i, expect := range expectRows {
		assertRow(t, expect, actualRows[i], i)
	}

	// 真实查询条件回环：单引号与反斜杠作为where条件也能正确命中
	_, rows, err := conn.Query(fmt.Sprintf("SELECT id FROM %s WHERE v_varchar = 'it''s'", conn.GetDialect().Quoter().Quote("it_escape")))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	_ = columns
}

// 重复数据Update策略（ON DUPLICATE KEY UPDATE）真实执行
func TestITMysqlDuplicateStrategyUpdate(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	gen := conn.GetDialect().GetSQLGenerator()
	table := "it_dup_update"
	mustExec(t, conn, "DROP TABLE IF EXISTS "+conn.GetDialect().Quoter().Quote(table))
	mustExec(t, conn, fmt.Sprintf("CREATE TABLE %s (id int NOT NULL PRIMARY KEY, val varchar(50))", conn.GetDialect().Quoter().Quote(table)))

	columns := []dbi.Column{{ColumnName: "id", DataType: "int"}, {ColumnName: "val", DataType: "varchar", CharMaxLength: 50}}
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}

	// 首次插入
	for _, sql := range gen.GenInsert(table, columns, [][]any{{1, "a"}}, dbi.DuplicateStrategyUpdate, meta) {
		mustExec(t, conn, sql)
	}
	// 冲突更新
	for _, sql := range gen.GenInsert(table, columns, [][]any{{1, "b"}, {2, "c"}}, dbi.DuplicateStrategyUpdate, meta) {
		mustExec(t, conn, sql)
	}

	rows := readAllRows(t, conn, table, "id")
	require.Len(t, rows, 2)
	assert.Equal(t, "b", normalizeDbValue(rows[0]["val"]))
	assert.Equal(t, "c", normalizeDbValue(rows[1]["val"]))
}

// 复制表真实执行（结构+数据，数据为异步复制需等待）
func TestITMysqlCopyTable(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	_, _ = setupMysqlSourceTable(t, conn, "it_copy_src")

	require.NoError(t, conn.GetDialect().CopyTable(&dbi.DbCopyTable{TableName: "it_copy_src", CopyData: true}))
	time.Sleep(2 * time.Second) // 数据为异步复制

	tables, err := conn.GetMetadata().GetTables("it_copy_src_copy_%")
	// GetTables过滤可能不支持通配，退化用全表名比对
	if err != nil || len(tables) == 0 {
		tables, err = conn.GetMetadata().GetTables()
		require.NoError(t, err)
	}

	var copyTableName string
	for _, tb := range tables {
		if strings.HasPrefix(tb.TableName, "it_copy_src_copy_") {
			copyTableName = tb.TableName
			break
		}
	}
	require.NotEmpty(t, copyTableName, "copy table not found")

	srcRows := readAllRows(t, conn, "it_copy_src", "id")
	copyRows := readAllRows(t, conn, copyTableName, "id")
	assert.Len(t, copyRows, len(srcRows))
	if len(copyRows) > 0 {
		assertCell(t, "v_varchar", srcRows[0]["v_varchar"], copyRows[0]["v_varchar"])
	}
}

// ---------------------------------------------------------------------
// mysql → sqlite 异构迁移全流程（等价DumpDb链路：元数据→类型转换→DDL→数据→校验）
// ---------------------------------------------------------------------

func TestITMysqlToSqliteMigration(t *testing.T) {
	mconn := mysqlConn(t)
	defer mconn.Close()
	sconn := sqliteConn(t)
	defer sconn.Close()

	srcTable := "it_mig_src"
	srcColumns, expectRows := setupMysqlSourceTable(t, mconn, srcTable)
	require.Len(t, expectRows, 3)

	// 1. 类型转换：mysql → sqlite（走ConvToTargetDbColumn完整链路）
	targetDialect := sconn.GetDialect()
	convColumns := make([]dbi.Column, 0, len(srcColumns))
	for _, col := range srcColumns {
		if err := dbi.ConvToTargetDbColumn("mysql", "sqlite", targetDialect, &col); err != nil {
			t.Fatalf("convert column [%s] failed: %s", col.ColumnName, err.Error())
		}
		convColumns = append(convColumns, col)
	}

	// unsigned列转换语义断言（此前unsigned归一化修复的端到端验证）
	for _, col := range convColumns {
		switch col.ColumnName {
		case "v_uint":
			assert.Equal(t, "integer", col.DataType, "int unsigned应转换为sqlite INTEGER（且保留完整数值范围语义）")
		case "v_dt":
			assert.Equal(t, "datetime", col.DataType)
		case "v_blob":
			assert.Equal(t, "blob", col.DataType)
		}
	}

	// 2. 目标库建表（真实执行sqlite DDL）
	destTable := "it_mig_dest"
	mustExec(t, sconn, "DROP TABLE IF EXISTS "+targetDialect.Quoter().Quote(destTable))
	for _, ddl := range targetDialect.GetSQLGenerator().GenTableDDL(dbi.Table{TableName: destTable}, convColumns, false) {
		mustExec(t, sconn, ddl)
	}

	// 3. 从mysql游标读取数据（含大值uint64/4294967295/NULL等），转换后写入sqlite
	insertColumns := convColumns
	var insertValues [][]any
	_, err := mconn.WalkTableRows(itCtx(), mconn.GetDialect().Quoter().Quote(srcTable), func(row map[string]any, _ []*dbi.QueryColumn) error {
		rowVals := make([]any, 0, len(insertColumns))
		for _, col := range insertColumns {
			rowVals = append(rowVals, row[col.ColumnName])
		}
		insertValues = append(insertValues, rowVals)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, insertValues, 3)

	insertSqls := targetDialect.GetSQLGenerator().GenInsert(destTable, insertColumns, insertValues, dbi.DuplicateStrategyNone, nil)
	for _, sql := range insertSqls {
		mustExec(t, sconn, sql)
	}

	// 4. sqlite回读，逐行逐列对比（跨库值形态差异由assertCell宽松归一处理）
	actualRows := readAllRows(t, sconn, destTable, "id")
	require.Len(t, actualRows, len(expectRows))
	for i, expect := range expectRows {
		assertRow(t, expect, actualRows[i], i)
	}
}

// ---------------------------------------------------------------------
// sqlite 全流程
// ---------------------------------------------------------------------

func TestITSqliteFullFlow(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()

	dialect := conn.GetDialect()
	gen := dialect.GetSQLGenerator()
	quote := dialect.Quoter().Quote

	// 建表（真实执行）
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "INTEGER", IsPrimaryKey: true},
		{ColumnName: "name", DataType: "TEXT", Nullable: false},
		{ColumnName: "val", DataType: "TEXT", Nullable: true},
	}
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote("it_sqlite_flow"))
	for _, ddl := range gen.GenTableDDL(dbi.Table{TableName: "it_sqlite_flow"}, columns, false) {
		mustExec(t, conn, ddl)
	}

	// 转义数据插入（sqlite无反斜杠转义，反斜杠是普通字符）
	values := [][]any{
		{1, "it's", "a\\b"},
		{2, `say "hi"`, "line1\nline2"},
		{3, "中文'单引'🙂", nil},
	}
	for _, sql := range gen.GenInsert("it_sqlite_flow", columns, values, dbi.DuplicateStrategyNone, nil) {
		mustExec(t, conn, sql)
	}

	rows := readAllRows(t, conn, "it_sqlite_flow", "id")
	require.Len(t, rows, 3)
	assert.Equal(t, "it's", normalizeDbValue(rows[0]["name"]))
	assert.Equal(t, `a\b`, normalizeDbValue(rows[0]["val"]))
	assert.Equal(t, `say "hi"`, normalizeDbValue(rows[1]["name"]))
	assert.Equal(t, "line1\nline2", normalizeDbValue(rows[1]["val"]))
	assert.Equal(t, "中文'单引'🙂", normalizeDbValue(rows[2]["name"]))
	assert.Nil(t, normalizeDbValue(rows[2]["val"]))

	// 复制表（真实执行GetTableDDL→改名→建表→复制数据）
	require.NoError(t, dialect.CopyTable(&dbi.DbCopyTable{TableName: "it_sqlite_flow", CopyData: true}))
	time.Sleep(1 * time.Second)

	tables, err := conn.GetMetadata().GetTables()
	require.NoError(t, err)
	var copyName string
	for _, tb := range tables {
		if strings.HasPrefix(tb.TableName, "it_sqlite_flow_copy_") {
			copyName = tb.TableName
			break
		}
	}
	require.NotEmpty(t, copyName, "sqlite copy table not found")
	copyRows := readAllRows(t, conn, copyName, "id")
	assert.Len(t, copyRows, 3)
	assert.Equal(t, "it's", normalizeDbValue(copyRows[0]["name"]))
}
