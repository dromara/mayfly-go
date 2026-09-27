//go:build it

package application

// 表格文件（CSV）导入链路（DbDataImportApp.Import）真实 sqlite 落库闭环测试。
//
// 用嵌入式 sqlite（临时文件，用后即删）验证「解析 → 列映射 → 目标列类型转换 → 方言批量 INSERT → 事务写入」
// 全链路的真实结果：类型原样落库、空值/空串语义、批量分片、单事务失败整体回滚、冲突策略前置校验、
// 按列位置映射对同名表头免疫、二进制列按字面字节写入。这些都是纯单测覆盖不到、只有真实库才成立的断言。
//
// 运行：cd server && go test -tags it -count=1 -run TestITDataImport ./internal/db/application/

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dbi/value"
	"mayfly-go/internal/db/dbm/importer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	importItTable = "it_data_import"
	importItNoKey = "it_data_import_nokey"
	importItBlob  = "it_data_import_blob"
)

// importItPrepare 建一张含多类型列的 sqlite 表：id 主键、name 非空、score 浮点、note 可空文本
func importItPrepare(t *testing.T) *dbi.DbConn {
	t.Helper()
	conn := appSQLiteConn(t)
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec("DROP TABLE IF EXISTS " + quote(importItTable))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY, name TEXT NOT NULL, score REAL, note TEXT)", quote(importItTable)))
	require.NoError(t, err, "建表失败")
	return conn
}

// importItRun 用给定选项驱动生产 Import 执行一段 CSV 内容。映射按列位置下标（"0".."3"）。
func importItRun(conn *dbi.DbConn, csv, table string, hasHeader, emptyAsNull bool, strategy, batch int) (*dto.DataImportRes, error) {
	app := &dbDataImportAppImpl{}
	mappings := []dto.ImportColumn{
		{Source: "0", Target: "id"},
		{Source: "1", Target: "name"},
		{Source: "2", Target: "score"},
		{Source: "3", Target: "note"},
	}
	return app.Import(context.Background(), &dto.DataImportReq{
		DbConn:            conn,
		TableName:         table,
		Reader:            strings.NewReader(csv),
		Filename:          "data.csv",
		Options:           &importer.Options{HasHeader: hasHeader},
		Columns:           mappings,
		EmptyAsNull:       emptyAsNull,
		DuplicateStrategy: strategy,
		BatchSize:         batch,
	})
}

// importItRows 读回用例表，按 id 升序
func importItRows(t *testing.T, conn *dbi.DbConn, table string) []map[string]any {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query("SELECT id, name, score, note FROM " + quote(table) + " ORDER BY id")
	require.NoError(t, err)
	return rows
}

func importItText(v any) string {
	switch b := v.(type) {
	case nil:
		return "<NULL>"
	case []byte:
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// TestITDataImportTypingAndNull 类型转换与空值→NULL 落库闭环：数值/文本/引号内逗号/空列语义都必须正确。
func TestITDataImportTypingAndNull(t *testing.T) {
	conn := importItPrepare(t)
	defer conn.Close()

	csv := "id,name,score,note\n" +
		"1,alice,1.5,hi\n" +
		"2,\"Smith, J\",2.25,\"带,逗号\"\n" +
		"3,bob,,\n"

	res, err := importItRun(conn, csv, importItTable, true, true, dbi.DuplicateStrategyNone, 100)
	require.NoError(t, err)
	assert.Equal(t, 3, res.TotalRows)
	assert.Equal(t, 3, res.Imported)

	rows := importItRows(t, conn, importItTable)
	require.Len(t, rows, 3)
	assert.Equal(t, "alice", importItText(rows[0]["name"]))
	assert.Equal(t, "Smith, J", importItText(rows[1]["name"]), "引号内含逗号字段必须整体读回")
	assert.Equal(t, "带,逗号", importItText(rows[1]["note"]), "中文+逗号内容不得被误切分")
	assert.Equal(t, "<NULL>", importItText(rows[2]["score"]), "空 score 应落 NULL")
	assert.Equal(t, "<NULL>", importItText(rows[2]["note"]), "空 note 应落 NULL")
	assert.Equal(t, "1.5", importItText(rows[0]["score"]), "REAL 值必须原样落库")
}

// TestITDataImportEmptyAsString 未勾选「空→NULL」时：可空文本列写空串，可空非文本列(数值)仍写 NULL
// （” 对数值/日期列非法，不能落空串）。
func TestITDataImportEmptyAsString(t *testing.T) {
	conn := importItPrepare(t)
	defer conn.Close()

	res, err := importItRun(conn, "id,name,score,note\n1,alice,,\n", importItTable, true, false, dbi.DuplicateStrategyNone, 100)
	require.NoError(t, err)
	require.Equal(t, 1, res.Imported)

	rows := importItRows(t, conn, importItTable)
	require.Len(t, rows, 1)
	assert.Equal(t, "", importItText(rows[0]["note"]), "可空文本列空值应写空字符串")
	assert.Equal(t, "<NULL>", importItText(rows[0]["score"]), "可空数值列空值应写 NULL 而非空串")
}

// TestITDataImportTemporalEmptyIsNull 日期时间列的空单元格必须写 NULL，即使未勾选「空→NULL」——
// ” 对 datetime 非法（MySQL 1292），非文本列的空值只能是 NULL。
func TestITDataImportTemporalEmptyIsNull(t *testing.T) {
	conn := appSQLiteConn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent
	tbl := "it_data_import_dt"
	_, err := conn.Exec("DROP TABLE IF EXISTS " + quote(tbl))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY, published_at DATETIME)", quote(tbl)))
	require.NoError(t, err)

	app := &dbDataImportAppImpl{}
	_, err = app.Import(context.Background(), &dto.DataImportReq{
		DbConn:      conn,
		TableName:   tbl,
		Reader:      strings.NewReader("id,published_at\n1,\n"),
		Filename:    "data.csv",
		Options:     &importer.Options{HasHeader: true},
		Columns:     []dto.ImportColumn{{Source: "0", Target: "id"}, {Source: "1", Target: "published_at"}},
		EmptyAsNull: false,
	})
	require.NoError(t, err)

	_, rows, err := conn.Query("SELECT typeof(published_at) AS ty FROM " + quote(tbl) + " WHERE id = 1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "null", importItText(rows[0]["ty"]), "日期时间列空单元格应落 NULL（typeof=null），而非 ''")
}

// TestITDataImportBatchSplit batchSize 小于行数时必须分多条批量语句且全部写入。
func TestITDataImportBatchSplit(t *testing.T) {
	conn := importItPrepare(t)
	defer conn.Close()

	var sb strings.Builder
	sb.WriteString("id,name,score,note\n")
	const n = 5
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "%d,name%d,1.1,x%d\n", i, i, i)
	}

	res, err := importItRun(conn, sb.String(), importItTable, true, true, dbi.DuplicateStrategyNone, 2)
	require.NoError(t, err)
	assert.Equal(t, n, res.Imported)
	assert.Equal(t, 3, res.BatchCount, "5 行按每批 2 行应生成 3 条批量语句")
	assert.Len(t, importItRows(t, conn, importItTable), n)
}

// TestITDataImportFailureRollbackAll 单事务内某批失败（主键重复）必须整体回滚，含已成功的前序批次。
func TestITDataImportFailureRollbackAll(t *testing.T) {
	conn := importItPrepare(t)
	defer conn.Close()

	// batchSize=1 拆两批：第二批 id 重复主键失败，第一批已写入的数据也必须随事务回滚清零
	csv := "id,name,score,note\n1,a,1.0,x\n1,b,2.0,y\n"
	_, err := importItRun(conn, csv, importItTable, true, true, dbi.DuplicateStrategyNone, 1)
	require.Error(t, err, "主键重复应报错")
	require.Len(t, importItRows(t, conn, importItTable), 0, "失败后整表必须回滚为空（含已执行的第一批）")
}

// TestITDataImportNotNullEmptyBecomesEmptyString 勾选「空→NULL」时，NOT NULL 列的空单元格退化为空串
// （写 NULL 必被数据库拒绝），可空列仍写 NULL——修复「勾了空→NULL 对非空列一定 1048」的矛盾。
func TestITDataImportNotNullEmptyBecomesEmptyString(t *testing.T) {
	conn := importItPrepare(t)
	defer conn.Close()

	// name 为 NOT NULL 且留空；note 可空且留空
	csv := "id,name,score,note\n1,,1.0,\n"
	res, err := importItRun(conn, csv, importItTable, true, true, dbi.DuplicateStrategyNone, 100)
	require.NoError(t, err, "NOT NULL 空单元格应退化为空串，导入不应失败")
	require.Equal(t, 1, res.Imported)

	rows := importItRows(t, conn, importItTable)
	require.Len(t, rows, 1)
	assert.Equal(t, "", importItText(rows[0]["name"]), "NOT NULL 列空值应写空串而非 NULL")
	assert.Equal(t, "<NULL>", importItText(rows[0]["note"]), "可空列空值仍写 NULL")
}

// TestITDataImportHeaderlessByIndex 无表头文件按列下标映射（source "0".."3"）。
func TestITDataImportHeaderlessByIndex(t *testing.T) {
	conn := importItPrepare(t)
	defer conn.Close()

	res, err := importItRun(conn, "1,alice,1.5,hi\n", importItTable, false, true, dbi.DuplicateStrategyNone, 100)
	require.NoError(t, err)
	require.Equal(t, 1, res.Imported)

	rows := importItRows(t, conn, importItTable)
	require.Len(t, rows, 1)
	assert.Equal(t, "alice", importItText(rows[0]["name"]))
	assert.Equal(t, "hi", importItText(rows[0]["note"]))
}

// TestITDataImportOutOfIndex 映射下标越界必须报错，而非静默把整列写成 NULL。
func TestITDataImportOutOfIndex(t *testing.T) {
	conn := importItPrepare(t)
	defer conn.Close()

	app := &dbDataImportAppImpl{}
	_, err := app.Import(context.Background(), &dto.DataImportReq{
		DbConn:      conn,
		TableName:   importItTable,
		Reader:      strings.NewReader("id,name,score,note\n1,a,1.0,x\n"),
		Filename:    "data.csv",
		Options:     &importer.Options{HasHeader: true},
		Columns:     []dto.ImportColumn{{Source: "9", Target: "id"}},
		EmptyAsNull: true,
	})
	require.Error(t, err, "越界列下标必须报错而非静默 NULL")
}

// TestITDataImportIgnoreNoPK 忽略策略不强求主键：无主键无唯一索引的表也应能执行 insert-or-ignore 导入。
func TestITDataImportIgnoreNoPK(t *testing.T) {
	conn := appSQLiteConn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec("DROP TABLE IF EXISTS " + quote(importItNoKey))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (id INTEGER, name TEXT, score REAL, note TEXT)", quote(importItNoKey)))
	require.NoError(t, err)

	res, err := importItRun(conn, "id,name,score,note\n1,a,1.0,x\n", importItNoKey, true, true, dbi.DuplicateStrategyIgnore, 100)
	require.NoError(t, err, "无主键表执行忽略策略导入不应因缺主键被拒")
	assert.Equal(t, 1, res.Imported)
}

// TestITDataImportUpdateNeedsConflictKey 更新(upsert)策略必须有主键或唯一索引，否则明确报错而非静默直插。
func TestITDataImportUpdateNeedsConflictKey(t *testing.T) {
	conn := appSQLiteConn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec("DROP TABLE IF EXISTS " + quote(importItNoKey))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (id INTEGER, name TEXT, score REAL, note TEXT)", quote(importItNoKey)))
	require.NoError(t, err)

	_, err = importItRun(conn, "id,name,score,note\n1,a,1.0,x\n", importItNoKey, true, true, dbi.DuplicateStrategyUpdate, 100)
	require.Error(t, err, "无冲突键的表执行更新策略应报错，避免静默退化为重复插入")
}

// TestITDataImportBinaryLiteralBytes 二进制列按字面字节写入：形如 MD5 的「偶数长度全 hex」文本
// 必须原样存为其字节内容，而不能被 SQLValueBytes 当作十六进制解码（否则静默数据损坏）。
func TestITDataImportBinaryLiteralBytes(t *testing.T) {
	conn := appSQLiteConn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec("DROP TABLE IF EXISTS " + quote(importItBlob))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY, md5 BLOB)", quote(importItBlob)))
	require.NoError(t, err)

	md5Hex := "d41d8cd98f00b204e9800998ecf8427e" // 32 位全 hex，最易被误解码
	app := &dbDataImportAppImpl{}
	_, err = app.Import(context.Background(), &dto.DataImportReq{
		DbConn:      conn,
		TableName:   importItBlob,
		Reader:      strings.NewReader("id,md5\n1," + md5Hex + "\n"),
		Filename:    "data.csv",
		Options:     &importer.Options{HasHeader: true},
		Columns:     []dto.ImportColumn{{Source: "0", Target: "id"}, {Source: "1", Target: "md5"}},
		EmptyAsNull: true,
	})
	require.NoError(t, err)

	// 用 SQL 侧 CAST/length 断言，规避 sqlite 驱动把 BLOB 读回为 sql.RawBytes 的类型差异
	_, rows, err := conn.Query("SELECT CAST(md5 AS TEXT) AS t, length(md5) AS n FROM " + quote(importItBlob) + " WHERE id = 1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, md5Hex, importItText(rows[0]["t"]), "二进制列应存字面文本字节，而非被 hex 解码")
	cnt, ok := value.ValToInt64(rows[0]["n"])
	require.True(t, ok, "length 取值形态异常: %T", rows[0]["n"])
	assert.Equal(t, int64(len(md5Hex)), cnt, "字节长度应等于原文本长度（若被 hex 解码则只剩 16）")
}

// TestITDataImportUpdateRequiresConflictKeyMapped 更新策略要求冲突键列被映射：只映射非主键列时应报错，
// 否则 on-conflict 依据的主键不在插入列里 → 数据库生成新值永不冲突 → 静默插重复。
func TestITDataImportUpdateRequiresConflictKeyMapped(t *testing.T) {
	conn := importItPrepare(t)
	defer conn.Close()
	app := &dbDataImportAppImpl{}
	_, err := app.Import(context.Background(), &dto.DataImportReq{
		DbConn:            conn,
		TableName:         importItTable,
		Reader:            strings.NewReader("name\nalice\n"),
		Filename:          "data.csv",
		Options:           &importer.Options{HasHeader: true},
		Columns:           []dto.ImportColumn{{Source: "0", Target: "name"}},
		EmptyAsNull:       true,
		DuplicateStrategy: dbi.DuplicateStrategyUpdate,
	})
	require.Error(t, err, "更新策略未映射主键应报错而非静默插重复")
	assert.Contains(t, err.Error(), "conflict key")
}
