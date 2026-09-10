package itest

// 特殊字符（回车CR/CRLF/NUL/多行JSON等）真实数据库导出→导入回环测试：
//   - mysql：varchar/text 列（json文本作为字符串值覆盖其转义），导出→导回mysql新表逐值比对 + 导出→导入sqlite跨方言比对
//   - pg：text/jsonb 列，导出→导回pg新表逐值比对（\0排除：pg text硬性禁止，属数据库固有限制）
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/dbm/

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

var escapeSpecialColumns = []dbi.Column{
	{ColumnName: "id", DataType: "int"},
	{ColumnName: "v_varchar", DataType: "varchar", CharMaxLength: 500},
	{ColumnName: "v_text", DataType: "text"},
}

// 特殊字符值谱系：CR / CRLF / 多行LF / 单引号 / 双引号 / 反斜杠 / tab / 中文 / emoji / 多行JSON
// 注：NUL（\0）行不在此处——sqlite无法通过SQL文本存储NUL字节（C API固有限制），
// 由TestITMysqlEscapeSpecialRoundtrip内单独验证mysql自身的NUL回环
var escapeSpecialRows = [][]any{
	{1, "cr\rreturn", "line1\rline2\r\rline4"},
	{2, "crlf\r\nend", "line1\r\nline2\nline3\ttab"},
	{3, `json"q'\path`, `{"msg": "it's \"quoted\"", "path": "C:\\tmp"}`},
	{4, "multi\njson\nlines", "多行json\n" + `{"a": [1, 2, "x'y"]}` + "\n" + `{"b": "🙂"}`},
	{5, "emoji🙂中文'", "back\\slash'tab\tend"},
}

// writeByGenInsert 走真实导出转义链路（GetSQLGenerator().GenInsert）写入数据
func writeByGenInsert(t *testing.T, conn *dbi.DbConn, table string, columns []dbi.Column, rows [][]any) {
	t.Helper()
	for _, sql := range conn.GetDialect().GetSQLGenerator().GenInsert(table, columns, rows, dbi.DuplicateStrategyNone, nil) {
		mustExec(t, conn, sql)
	}
}

// TestITMysqlEscapeSpecialRoundtrip mysql特殊字符导出→导回mysql + 导出→导入sqlite
func TestITMysqlEscapeSpecialRoundtrip(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	table := "it_escape_special"
	mustExec(t, conn, "DROP TABLE IF EXISTS `"+table+"`")
	mustExec(t, conn, "CREATE TABLE `"+table+"` (id INT PRIMARY KEY, v_varchar VARCHAR(500), v_text TEXT)")
	writeByGenInsert(t, conn, table, escapeSpecialColumns, escapeSpecialRows)

	// 导出→导回mysql新表（真实切割器切割 + 事务导入链路）
	script := dumpTableScript(t, conn, table, conn.GetDialect(), "mysql")
	importTable := table + "_import"
	mustExec(t, conn, "DROP TABLE IF EXISTS `"+importTable+"`")
	mustExec(t, conn, "CREATE TABLE `"+importTable+"` LIKE `"+table+"`")
	execStmtsInTx(t, conn, strings.NewReader(strings.ReplaceAll(script, table, importTable)))

	// 逐行逐列比对
	srcRows := readAllRows(t, conn, table, "id")
	dstRows := readAllRows(t, conn, importTable, "id")
	require.Len(t, dstRows, len(srcRows))
	for i := range srcRows {
		assertCell(t, "v_varchar", srcRows[i]["v_varchar"], dstRows[i]["v_varchar"])
		assertCell(t, "v_text", srcRows[i]["v_text"], dstRows[i]["v_text"])
	}

	// 跨方言：mysql→sqlite（json列被ConvToTargetDbColumn转为text）
	sqConn := sqliteConn(t)
	defer sqConn.Close()
	script2 := dumpTableScript(t, conn, table, sqConn.GetDialect(), "sqlite")
	mustExec(t, sqConn, "CREATE TABLE `"+table+"` (id INTEGER PRIMARY KEY, v_varchar TEXT, v_text TEXT)")
	execStmtsInTx(t, sqConn, strings.NewReader(script2))
	sqRows := readAllRows(t, sqConn, table, "id")
	require.Len(t, sqRows, len(srcRows))
	for i := range srcRows {
		assertCell(t, "v_varchar", srcRows[i]["v_varchar"], sqRows[i]["v_varchar"])
		assertCell(t, "v_text", srcRows[i]["v_text"], sqRows[i]["v_text"])
	}

	// NUL字节单独验证mysql自身的导出→导回回环（sqlite无法通过SQL文本存储NUL，见上方注释）
	nulRows := [][]any{{6, "nul\x00char", "nul\x00in\x00text"}}
	writeByGenInsert(t, conn, table, escapeSpecialColumns, nulRows)
	nulScript := dumpTableScript(t, conn, table, conn.GetDialect(), "mysql")
	mustExec(t, conn, "DROP TABLE IF EXISTS `"+importTable+"`")
	mustExec(t, conn, "CREATE TABLE `"+importTable+"` LIKE `"+table+"`")
	execStmtsInTx(t, conn, strings.NewReader(strings.ReplaceAll(nulScript, table, importTable)))
	nulSrc := readAllRows(t, conn, table, "id")
	nulDst := readAllRows(t, conn, importTable, "id")
	require.Len(t, nulDst, len(nulSrc))
	assertCell(t, "v_varchar", nulSrc[len(nulSrc)-1]["v_varchar"], nulDst[len(nulDst)-1]["v_varchar"])
	assertCell(t, "v_text", nulSrc[len(nulSrc)-1]["v_text"], nulDst[len(nulDst)-1]["v_text"])

	mustExec(t, conn, "DROP TABLE IF EXISTS `"+importTable+"`")
}

// TestITPgEscapeSpecialRoundtrip pg特殊字符（含$字面量、多行jsonb）导出→导回pg
// 注：\0不在测试范围——pg text硬性禁止NUL字节，属数据库固有限制
func TestITPgEscapeSpecialRoundtrip(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()

	table := "it_pg_escape_special"
	mustExec(t, conn, "DROP TABLE IF EXISTS "+conn.GetDialect().Quoter().Quote(table))
	mustExec(t, conn, "CREATE TABLE "+conn.GetDialect().Quoter().Quote(table)+
		" (id INT PRIMARY KEY, v_text TEXT, v_jsonb JSONB)")

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int"},
		{ColumnName: "v_text", DataType: "text"},
		{ColumnName: "v_jsonb", DataType: "jsonb"},
	}
	rows := [][]any{
		{1, "cr\rreturn\r\nlf\nline2\ttab", `{"msg": "it's \"quoted\"", "path": "C:\\tmp"}`},
		{2, `dollar$$inside'quote"quote\back`, "{\n  \"a\": [1, 2, \"x'y\"],\n  \"b\": \"🙂\"\n}"},
		// JSON字符串值内的控制字符必须为转义序列形式（JSON规范禁止裸控制字符），
		// 与生产链路一致（mysql/pg的json列回读文本均已规范化为转义形式）
		{3, "e'escape\\end🙂", `{"nested": {"arr": ["a\r\nb", "c'd"], "note": "line1\nline2"}}`},
	}
	writeByGenInsert(t, conn, table, columns, rows)

	// 导出→导回pg新表（真实PgsqlSplitter切割 + 事务导入链路）
	script := dumpTableScript(t, conn, table, conn.GetDialect(), "postgres")
	importTable := table + "_import"
	mustExec(t, conn, "DROP TABLE IF EXISTS "+conn.GetDialect().Quoter().Quote(importTable))
	mustExec(t, conn, "CREATE TABLE "+conn.GetDialect().Quoter().Quote(importTable)+
		" (id INT PRIMARY KEY, v_text TEXT, v_jsonb JSONB)")
	execStmtsInTx(t, conn, strings.NewReader(strings.ReplaceAll(script, table, importTable)))

	srcRows := readAllRows(t, conn, table, "id")
	dstRows := readAllRows(t, conn, importTable, "id")
	require.Len(t, dstRows, len(srcRows))
	for i := range srcRows {
		assertCell(t, "v_text", srcRows[i]["v_text"], dstRows[i]["v_text"])
		assertCell(t, "v_jsonb", srcRows[i]["v_jsonb"], dstRows[i]["v_jsonb"])
	}

	// jsonb内容的语义级抽查（规范化文本含原特殊字符）
	assert.Contains(t, fmt.Sprint(dstRows[0]["v_jsonb"]), `it's \"quoted\"`)
	assert.Contains(t, fmt.Sprint(dstRows[0]["v_jsonb"]), `C:\\tmp`)

	mustExec(t, conn, "DROP TABLE IF EXISTS "+conn.GetDialect().Quoter().Quote(importTable))
}
