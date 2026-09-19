package itest

// 导出格式层（CSV/JSON）真实数据库集成测试。
//
// 背景：Exporter 重构后 CSV/JSON 走「非脚本路径」（SupportsScript=false：无DDL段/无事务
// 钩子，Begin/ConsumeBatch*/End 纯数据序列化），该路径此前零运行时覆盖——CSV 消费者曾出现
// 缓冲未 Flush 的丢数据级缺陷恰好证明此区域无测试保护。本文件按基础设施平台数据准确性标准
// 钉死：全字段类型形态、NULL/特殊字符/emoji/二进制、跨批次边界、配置开关、多表产物结构。
//
// 断言原则：不猜测驱动返回值形态——期望值取自同一连接的直读结果（与 dump 链路同经
// Valuer 归一化），比对「导出产物 == 直读值」的等价性；特殊字符列另行逐字节钉死原文。
//
// 运行：cd server && go test -tags it -count=1 -run 'TestITExport|TestITQuery2Struct' ./internal/db/application/transfer/itest/

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/transfer"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dbi/value"
	"mayfly-go/internal/db/dbm/export"
)

// ─────────────────────────── 公共骨架 ───────────────────────────

// mustItExec 执行测试DDL/DML，失败即中止用例
func mustItExec(t *testing.T, conn *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := conn.Exec(sql); err != nil {
		t.Fatalf("exec failed [%s]: %s", sql, err.Error())
	}
}

// itFmtTable 导出格式测试统一表名与列清单（列顺序即产物字段顺序）
const itFmtTable = "it_export_fmt"

var itFmtColumns = []string{"id", "v_str", "v_rfc", "v_cn", "v_small", "v_big", "v_dec", "v_dt", "v_date", "v_txt", "v_bin"}

// itFmtRows 5行边界数据：常规/CSV特殊字符(引号+逗号+换行)/中文emoji/长文本+负小数/全NULL
// 行数5配合 BatchRows=2 → 3批（2+2+1），强制覆盖跨批次衔接（JSON逗号、CSV连续写）
func itFmtRowData() []map[string]any {
	return []map[string]any{
		{"id": 1, "v_str": "plain", "v_rfc": `he said "hi", ok`, "v_cn": "中文数据😀", "v_small": 42, "v_big": 9007199254740993, "v_dec": "12345678.123456", "v_dt": "2024-02-29 10:20:30.123456", "v_date": "2024-02-29", "v_txt": "short", "v_bin": "000102FF"},
		{"id": 2, "v_str": "", "v_rfc": "line1\nline2", "v_cn": "tab\there", "v_small": 0, "v_big": -9223372036854775808, "v_dec": "-0.000001", "v_dt": nil, "v_date": "1000-01-01", "v_txt": strings.Repeat("长文本", 700), "v_bin": nil},
		{"id": 3, "v_str": "nulls-below", "v_rfc": "semi;colon|pipe", "v_cn": "", "v_small": 1, "v_big": 9223372036854775807, "v_dec": "0.000000", "v_dt": "1970-01-01 00:00:01.000000", "v_date": nil, "v_txt": "", "v_bin": "DEADBEEF"},
		{"id": 4, "v_str": nil, "v_rfc": nil, "v_cn": nil, "v_small": nil, "v_big": nil, "v_dec": nil, "v_dt": nil, "v_date": nil, "v_txt": nil, "v_bin": nil},
		{"id": 5, "v_str": "尾行", "v_rfc": "", "v_cn": "😀", "v_small": 255, "v_big": 1, "v_dec": "99999999.999999", "v_dt": "9999-12-31 23:59:59.999999", "v_date": "9999-12-31", "v_txt": "x", "v_bin": "00"},
	}
}

// itFmtCreateMysql 建全类型表并写入边界数据（binary以hex字面量写入，含NUL字节）
func itFmtCreateMysql(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	q := conn.GetDialect().Quoter().QuoteIdent
	mustItExec(t, conn, "DROP TABLE IF EXISTS "+q(itFmtTable))
	mustItExec(t, conn, fmt.Sprintf("CREATE TABLE %s ("+
		"id INT PRIMARY KEY, v_str VARCHAR(100), v_rfc VARCHAR(500), v_cn VARCHAR(200), "+
		"v_small INT, v_big BIGINT, v_dec DECIMAL(20,6), v_dt DATETIME(6), v_date DATE, "+
		"v_txt TEXT, v_bin BLOB) DEFAULT CHARSET=utf8mb4", q(itFmtTable)))

	for _, row := range itFmtRowData() {
		sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q(itFmtTable),
			strings.Join(quoteAll(q, itFmtColumns), ","),
			valuesLiteral(row, itFmtColumns))
		mustItExec(t, conn, sql)
	}
}

func quoteAll(q func(string) string, names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = q(n)
	}
	return out
}

// valuesLiteral 行数据→INSERT值字面量（NULL→NULL，字符串转义，blob→0xhex）
func valuesLiteral(row map[string]any, columns []string) string {
	parts := make([]string, 0, len(columns))
	for _, c := range columns {
		v := row[c]
		switch val := v.(type) {
		case nil:
			parts = append(parts, "NULL")
		case int:
			parts = append(parts, fmt.Sprintf("%d", val))
		case int64:
			parts = append(parts, fmt.Sprintf("%d", val))
		case string:
			if c == "v_bin" {
				parts = append(parts, "0x"+val)
			} else {
				parts = append(parts, "'"+strings.ReplaceAll(val, "'", "''")+"'")
			}
		default:
			parts = append(parts, fmt.Sprintf("%v", val))
		}
	}
	return strings.Join(parts, ",")
}

// itDumpFormat 驱动生产 DumpDbScript 按指定格式导出到内存缓冲
func itDumpFormat(t *testing.T, conn *dbi.DbConn, tables []string, format string, settings *export.Settings) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName:       conn.Info.Database,
		Tables:       tables,
		DumpData:     true,
		Writer:       buf,
		ExportFormat: format,
		Settings:     settings,
	}), "导出失败")
	return buf
}

// itBatchSettings 行数预算=2 强制多批次，暴露跨批次衔接缺陷
func itBatchSettings(format string) *export.Settings {
	s := export.DefaultSettings(format)
	s.BatchRows = 2
	return s
}

// itFmtExpectedStrings 直读源表（与dump同经Valuer归一）→ 每列字符串形态的期望矩阵。
// 直读值即导出的真值基准：链路差异（游标 vs 全量Query）不改变列值形态，若改变即为缺陷
func itFmtExpectedStrings(t *testing.T, conn *dbi.DbConn, columns []string) []map[string]string {
	t.Helper()
	q := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT %s FROM %s ORDER BY %s",
		strings.Join(quoteAll(q, columns), ","), q(itFmtTable), q("id")))
	require.NoError(t, err)
	out := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		m := make(map[string]string, len(columns))
		for _, c := range columns {
			m[c] = itTextOf(row[c])
		}
		out = append(out, m)
	}
	return out
}

// itTextOf 值→文本形态（与导出侧 formatValue 语义一致的独立复刻，两侧不共享代码以构成交叉验证）
func itTextOf(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// itJsonDecode 以 UseNumber 解码 JSON 数组产物：数字保留原始文本形态。
// 默认 Unmarshal 会把 JSON number 转 float64，超 2^53 的 bigint 在测试比对层即失真，
// 会把精确产物误判为缺陷（或反向掩盖产物失真），故一切含数值断言的 JSON 校验必须经此解码
func itJsonDecode(t *testing.T, output string) []map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(output))
	dec.UseNumber()
	var arr []map[string]any
	require.NoError(t, dec.Decode(&arr))
	return arr
}

// ─────────────────────────── CSV ───────────────────────────

// TestITExportCsvMysqlFullTypes MySQL全字段类型→CSV（BatchRows=2跨批次）：
// 产物必须能被RFC4180解析器无损读回；特殊字符（引号/逗号/换行/tab/分号）必须被正确引用，
// 解析回来与原文逐字节一致；NULL→空串、blob→hex文本、大整数→完整位数十进制文本
func TestITExportCsvMysqlFullTypes(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)

	buf := itDumpFormat(t, conn, []string{itFmtTable}, "csv", itBatchSettings("csv"))
	output := buf.String()

	reader := csv.NewReader(strings.NewReader(output))
	records, err := reader.ReadAll()
	require.NoError(t, err, "CSV产物必须可被RFC4180解析器完整读回")

	// 表头 + 5行数据（换行符字段被正确引用则不会被误拆行——行数即引用正确性的直接证据）
	require.Len(t, records, 6, "CSV必须为表头+5行；行数异常说明含换行字段未正确引用")
	assert.Equal(t, itFmtColumns, records[0], "表头必须为元数据列顺序")

	expected := itFmtExpectedStrings(t, conn, itFmtColumns)
	require.Len(t, expected, 5)
	for i, expRow := range expected {
		rec := records[i+1]
		require.Len(t, rec, len(itFmtColumns))
		for j, col := range itFmtColumns {
			assert.Equal(t, expRow[col], rec[j], "第%d行[%s]导出值与源库直读不一致", i+1, col)
		}
	}

	// 特殊字符逐字节钉死（独立于直读链路的绝对基准）
	assert.Equal(t, `he said "hi", ok`, records[1][2])
	assert.Equal(t, "line1\nline2", records[2][2], "含裸换行字段必须整字段引用读回")
	assert.Equal(t, "中文数据😀", records[1][3])
	assert.Equal(t, "tab\there", records[2][3], "tab字段原样读回")
	assert.Equal(t, "semi;colon|pipe", records[3][2], "分号在默认逗号分隔下为普通字符")
	assert.Equal(t, "9223372036854775807", records[3][5], "bigint极值必须完整位数输出（转float会丢精度）")
	assert.Equal(t, "-9223372036854775808", records[2][5], "bigint负极值")
	assert.Equal(t, "12345678.123456", records[1][6], "decimal 6位小数不得丢失/进位")
	assert.Equal(t, "-0.000001", records[2][6], "row2 负小数按位保真")
	// NULL形态：CSV无NULL语义，NULL与空字符串同形（格式固有约束，非缺陷；需要区分用JSON）
	assert.Equal(t, "", records[4][1], "NULL列导出为空串（CSV固有：与空串不可区分，见JSON对照）")
	// 二进制经 ValuerBytes 归一为hex文本，无损可还原
	assert.Equal(t, "000102ff", strings.ToLower(records[1][10]), "blob→hex文本（含NUL字节，无损可还原）")
	assert.Equal(t, "", records[4][10], "NULL blob为空串")
}

// TestITExportCsvSettingsMatrix IncludeHeader/FieldSeparator/LineTerminator 配置真实生效
func TestITExportCsvSettingsMatrix(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)

	// 无表头 + 分号分隔 + CRLF
	s := itBatchSettings("csv")
	s.IncludeHeader = false
	s.FieldSeparator = ";"
	s.LineTerminator = "\r\n"
	buf := itDumpFormat(t, conn, []string{itFmtTable}, "csv", s)
	output := buf.String()

	assert.True(t, strings.HasPrefix(output, "1;"), "IncludeHeader=false 首行必须直接是数据")
	assert.Contains(t, output, "\r\n", "LineTerminator=\\r\\n 必须生效")

	reader := csv.NewReader(strings.NewReader(output))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 5, "无表头时应恰为5行数据")
	assert.Equal(t, "semi;colon|pipe", records[2][2], "分号分隔下含分号字段必须被引用且正确读回")
}

// ─────────────────────────── JSON ───────────────────────────

// TestITExportJsonMysqlFullTypes MySQL全字段类型→JSON（BatchRows=2跨批次）：
// 产物必须是合法JSON数组（跨批次逗号是历史高危点）；NULL→null（与空串可区分，CSV做不到的）；
// 大整数经 Valuer 转字符串保精度；数值/文本逐值与源库直读一致
func TestITExportJsonMysqlFullTypes(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)

	buf := itDumpFormat(t, conn, []string{itFmtTable}, "json", itBatchSettings("json"))
	output := buf.String()
	require.True(t, json.Valid([]byte(output)), "JSON产物必须是合法JSON（跨批次逗号/闭合错误=产物不可消费）\n产物片段: %.200s", output)

	arr := itJsonDecode(t, output)
	require.Len(t, arr, 5)

	// NULL 语义保留（与CSV的关键格式差异）
	assert.Nil(t, arr[3]["v_str"], "NULL必须输出为JSON null")
	assert.NotNil(t, arr[0]["v_str"])
	// 空串与NULL在JSON中可区分
	assert.Equal(t, "", arr[1]["v_str"], "空字符串必须输出为\"\"而非null")

	// 特殊字符/中文/emoji 逐字节
	assert.Equal(t, `he said "hi", ok`, arr[0]["v_rfc"])
	assert.Equal(t, "line1\nline2", arr[1]["v_rfc"])
	assert.Equal(t, "中文数据😀", arr[0]["v_cn"])

	// 精度敏感值与源直读交叉比对（经同一Valuer归一化）
	expected := itFmtExpectedStrings(t, conn, itFmtColumns)
	for i, exp := range expected {
		for _, col := range []string{"v_big", "v_dec", "v_dt", "v_date", "v_bin"} {
			assert.Equal(t, exp[col], itTextOf(arr[i][col]), "第%d行[%s] JSON值与源库直读不一致", i+1, col)
		}
	}

	// 19位bigint必须以字符串输出（JSON number承载会float64丢精度，Valuer已保精度转字符串——钉死该行为）
	assert.Equal(t, "9223372036854775807", arr[2]["v_big"], "超出安全整数范围的bigint必须为JSON字符串")
	// int64 域内 bigint 以 JSON number 输出且文本精确（UseNumber 解码下与源值逐字符一致）
	assert.Equal(t, "9007199254740993", itTextOf(arr[0]["v_big"]), "2^53+1 的 bigint 产物必须逐位精确（科学计数法=测试或实现的精度事故）")
	// blob→hex文本（JSON场景无损还原约定）
	assert.Equal(t, "deadbeef", strings.ToLower(arr[2]["v_bin"].(string)))

	// 小整数以 JSON number（无引号）形态输出，数值消费方可直接用
	assert.Contains(t, output, `"v_small":42`, "小整数必须是JSON number而非字符串")
}

// TestITExportJsonPrettyAndMultiTable PrettyPrint美化合法；多表产物必须为按表名分组的合法对象
// （回归守卫：修复前多表输出相邻数组拼接=非法JSON）
func TestITExportJsonPrettyAndMultiTable(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)
	q := conn.GetDialect().Quoter().QuoteIdent
	mustItExec(t, conn, "DROP TABLE IF EXISTS "+q("it_fmt_second"))
	mustItExec(t, conn, "CREATE TABLE "+q("it_fmt_second")+" (id INT PRIMARY KEY, v VARCHAR(20))")
	mustItExec(t, conn, "INSERT INTO "+q("it_fmt_second")+` VALUES (1,'a'),(2,'b')`)
	// 不清理：it_fmt_second 为本用例独占命名，且各建表用例开头均 DROP IF EXISTS，无残留影响
	// （教训：t.Cleanup 在 defer conn.Close() 之后执行，cleanup 内不可再用已关闭连接）

	// PrettyPrint：合法且可读
	s := export.DefaultSettings("json")
	s.PrettyPrint = true
	buf := itDumpFormat(t, conn, []string{itFmtTable}, "json", s)
	require.True(t, json.Valid(buf.Bytes()), "PrettyPrint产物必须仍为合法JSON")
	assert.Contains(t, buf.String(), "\n  \"", "PrettyPrint应有缩进")

	// 多表：{"table1":[...],"table2":[...]} 合法对象
	multi := itDumpFormat(t, conn, []string{itFmtTable, "it_fmt_second"}, "json", export.DefaultSettings("json"))
	var obj map[string][]map[string]any
	require.NoError(t, json.Unmarshal(multi.Bytes(), &obj), "多表JSON产物必须是合法对象（相邻数组拼接是非法JSON）\n产物: %.300s", multi.String())
	require.Len(t, obj, 2)
	assert.Len(t, obj[itFmtTable], 5)
	assert.Len(t, obj["it_fmt_second"], 2)
	assert.Equal(t, "a", obj["it_fmt_second"][0]["v"])
}

// TestITExportEmptyTable 空表：CSV仅表头，JSON为空数组（不得输出残缺结构）
func TestITExportEmptyTable(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)
	q := conn.GetDialect().Quoter().QuoteIdent
	mustItExec(t, conn, "DELETE FROM "+q(itFmtTable))

	csvBuf := itDumpFormat(t, conn, []string{itFmtTable}, "csv", export.DefaultSettings("csv"))
	records, err := csv.NewReader(strings.NewReader(csvBuf.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1, "空表CSV应仅有表头")
	assert.Equal(t, itFmtColumns, records[0])

	jsonBuf := itDumpFormat(t, conn, []string{itFmtTable}, "json", export.DefaultSettings("json"))
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &arr), "空表JSON必须为合法 []\n产物: %q", jsonBuf.String())
	assert.Empty(t, arr)
}

// TestITExportUnsupportedFormat 未注册格式必须显式报错（禁止静默回退SQL——用户要CSV拿到SQL是事故）
func TestITExportUnsupportedFormat(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	err := transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		Tables: []string{itFmtTable}, DumpData: true, Writer: &bytes.Buffer{}, ExportFormat: "parquet",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}

// ─────────────────────── 方言独立性（多源） ───────────────────────

// TestITExportCsvSqliteSource SQLite（弱类型，聚合与列值形态与mysql差异大）→ CSV：
// 同一格式实现必须对方言无关，各值与源直读一致
func TestITExportCsvSqliteSource(t *testing.T) {
	conn := itSqliteNode(t)
	defer conn.Close()
	q := conn.GetDialect().Quoter().QuoteIdent
	mustItExec(t, conn, "CREATE TABLE "+q(itFmtTable)+" (id INTEGER PRIMARY KEY, v_num REAL, v_txt TEXT, v_bin BLOB)")
	mustItExec(t, conn, "INSERT INTO "+q(itFmtTable)+` VALUES (1, 1.5, '中文''x''', X'010203'), (2, NULL, NULL, NULL), (3, -0.25, '', X'FF')`)

	buf := itDumpFormat(t, conn, []string{itFmtTable}, "csv", itBatchSettings("csv"))
	records, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 4, "表头+3行")
	assert.Equal(t, "1.5", records[1][1], "sqlite REAL经Valuer归一后CSV文本与直读一致")
	assert.Contains(t, records[1][2], "中文", "sqlite弱类型文本列内容完整")
	assert.Equal(t, "010203", strings.ToLower(records[1][3]), "sqlite BLOB→hex文本")
	assert.Equal(t, "", records[2][1], "NULL→空串")
}

// TestITExportJsonPgSource PostgreSQL（[]byte数值形态、timestamp/numeric特有返回形态）→ JSON：
// 数值列经Valuer归一为字符串输出，禁止出现base64/字节数组形态（数据可读性事故）
func TestITExportJsonPgSource(t *testing.T) {
	conn := itPgNode(t)
	defer conn.Close()
	q := conn.GetDialect().Quoter().QuoteIdent
	mustItExec(t, conn, "DROP TABLE IF EXISTS "+q(itFmtTable))
	mustItExec(t, conn, "CREATE TABLE "+q(itFmtTable)+" (id INT PRIMARY KEY, v_num NUMERIC(20,6), v_ts TIMESTAMP(6), v_bool BOOLEAN, v_bytea BYTEA)")
	mustItExec(t, conn, "INSERT INTO "+q(itFmtTable)+" VALUES (1, 123.456000, '2024-02-29 10:20:30.123456', true, '\\x0102'::bytea), (2, NULL, NULL, false, NULL)")

	buf := itDumpFormat(t, conn, []string{itFmtTable}, "json", export.DefaultSettings("json"))
	require.True(t, json.Valid(buf.Bytes()))
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &arr))
	require.Len(t, arr, 2)

	// NUMERIC(20,6) 按 scale 保尾零输出（123.456000），trim 尾零反而是精度失真；禁止的是[]byte→base64形态
	assert.Equal(t, "123.456000", itTextOf(arr[0]["v_num"]), "pg NUMERIC必须以十进制文本输出（禁止[]byte→base64形态）")
	assert.Contains(t, itTextOf(arr[0]["v_ts"]), "2024-02-29")
	assert.Nil(t, arr[1]["v_num"], "NULL→null")
	assert.Equal(t, "0102", strings.ToLower(itTextOf(arr[0]["v_bytea"])), "pg BYTEA→hex文本")
}

// ─────────────────────── scan 包（Query2Struct） ───────────────────────

// TestITQuery2StructRealRows scan.All 提取为 dbi/scan 后真实结果集端到端验证：
// 列名→db tag映射、NULL列（指针/可空类型）、跨多批次游标无关（全量scan）
func TestITQuery2StructRealRows(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)

	type row struct {
		Id     int64  `db:"id"`
		VStr   string `db:"v_str"`
		VSmall int32  `db:"v_small"`
	}
	var rows []row
	require.NoError(t, conn.Query2Struct(
		fmt.Sprintf("SELECT id, v_str, v_small FROM %s WHERE v_str IS NOT NULL ORDER BY id",
			conn.GetDialect().Quoter().QuoteIdent(itFmtTable)), &rows))
	require.Len(t, rows, 4, "NULL行被过滤后应余4行")
	assert.Equal(t, int64(1), rows[0].Id)
	assert.Equal(t, "plain", rows[0].VStr)
	assert.Equal(t, int32(42), rows[0].VSmall)
	assert.Equal(t, "尾行", rows[3].VStr, "中文字段映射正确")
}

// ─────────────────────── SQL 路径对照（格式路由不串扰） ───────────────────────

// TestITExportFormatRoutingIsolation 同表分别导出 sql/csv/json 三格式互不串扰：
// sql含INSERT与事务钩子（SupportsScript路径）、csv无SQL残留、json为纯数据数组——
// 证明 SupportsScript 能力探测未把脚本段泄漏进非脚本格式
func TestITExportFormatRoutingIsolation(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)

	sqlBuf := itDumpFormat(t, conn, []string{itFmtTable}, "sql", export.DefaultSettings("sql"))
	assert.Contains(t, sqlBuf.String(), "INSERT INTO", "SQL格式必须有INSERT语句")

	csvBuf := itDumpFormat(t, conn, []string{itFmtTable}, "csv", export.DefaultSettings("csv"))
	assert.NotContains(t, strings.ToUpper(csvBuf.String()), "INSERT INTO", "CSV不得混入SQL文本")

	jsonBuf := itDumpFormat(t, conn, []string{itFmtTable}, "json", export.DefaultSettings("json"))
	assert.NotContains(t, strings.ToUpper(jsonBuf.String()), "INSERT INTO", "JSON不得混入SQL文本")
	var check []map[string]any
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &check))
	require.Len(t, check, 5)
}

// TestITExportLargeTextCsvQuoting 超长文本（700×3字符中文长文本=2100字符）+ 特殊字符在CSV中
// 必须完整往返（writer缓冲跨批次：长文本行落在批次边界上）
func TestITExportLargeTextCsvQuoting(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)

	buf := itDumpFormat(t, conn, []string{itFmtTable}, "csv", itBatchSettings("csv"))
	records, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err)
	longText := records[2][9] // 第2行 v_txt（records[0]表头）
	assert.Equal(t, 2100, utf8.RuneCountInString(longText), "700个中文字符×3 完整往返（rune计数，len按字节）")
	assert.Equal(t, strings.Repeat("长文本", 700), longText)
}

// itFmtDumpToFileWithFormat 文件级导出（部分用例需落盘验证）
func itFmtDumpToFileWithFormat(t *testing.T, conn *dbi.DbConn, format string, path string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		Tables: []string{itFmtTable}, DumpData: true, Writer: f, ExportFormat: format,
	}))
	require.NoError(t, f.Close())
}

// TestITExportFilesOnDiskEndToEnd 文件落盘形态：三格式产物非空、扩展名语义内容自洽
func TestITExportFilesOnDiskEndToEnd(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)
	dir := t.TempDir()

	for _, format := range []string{"sql", "csv", "json"} {
		path := filepath.Join(dir, itFmtTable+"."+format)
		itFmtDumpToFileWithFormat(t, conn, format, path)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.NotEmpty(t, data, "格式[%s]文件产物不得为空", format)
		switch format {
		case "json":
			assert.True(t, json.Valid(data), "落盘JSON必须合法")
		case "csv":
			records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
			require.NoError(t, err)
			assert.Len(t, records, 6)
		case "sql":
			assert.Contains(t, string(data), "INSERT INTO")
		}
	}
}

// ─────────────────────── 导出产物→回灌→逐值闭环 ───────────────────────

// itFmtCreateTargetPg 建回灌目标表（强类型验证最严格：NUMERIC/TIMESTAMP/BYTEA 全要按形态喂对）
func itFmtCreateTargetPg(t *testing.T, conn *dbi.DbConn, table string) {
	t.Helper()
	q := conn.GetDialect().Quoter().QuoteIdent
	mustItExec(t, conn, "DROP TABLE IF EXISTS "+q(table))
	mustItExec(t, conn, "CREATE TABLE "+q(table)+" (id INT PRIMARY KEY, v_str TEXT, v_rfc TEXT, v_cn TEXT, "+
		"v_small BIGINT, v_big BIGINT, v_dec NUMERIC(20,6), v_dt TIMESTAMP(6), v_date DATE, v_txt TEXT, v_bin BYTEA)")
}

// itFmtTextMatrix 按连接读回导出闭环表的文本形态矩阵（与源侧 itFmtExpectedStrings 同构可比）
func itFmtTextMatrix(t *testing.T, conn *dbi.DbConn, table string) []map[string]string {
	t.Helper()
	q := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT %s FROM %s ORDER BY %s",
		strings.Join(quoteAll(q, itFmtColumns), ","), q(table), q("id")))
	require.NoError(t, err)
	out := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		m := make(map[string]string, len(itFmtColumns))
		for _, c := range itFmtColumns {
			m[c] = itTextOf(row[c])
		}
		out = append(out, m)
	}
	return out
}

// itParseIntOpt 文本→int64参数；空串（CSV的NULL形态）→nil
func itParseIntOpt(t *testing.T, s string) any {
	t.Helper()
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	require.NoError(t, err, "bigint 文本必须可无损解析回 int64: %q", s)
	return v
}

// itHexToBytes hex文本→字节；空→nil。证明 CSV/JSON 中 blob 的 hex 文本形态可被消费方无损还原
func itHexToBytes(t *testing.T, s string) any {
	t.Helper()
	if s == "" {
		return nil
	}
	b, err := hex.DecodeString(s)
	require.NoError(t, err, "blob hex 文本必须可解码: %q", s)
	return b
}

// itOptText 文本参数；空串→nil（NUMERIC/TIMESTAMP/DATE 列不接受空串字面量）
func itOptText(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// TestITExportCsvReimportRoundtrip CSV导出→解析→回灌PG→逐值闭环（数据准确性金标准）：
// 全 11 列×5 行含特殊字符/极值/中文/二进制，回灌后文本矩阵必须与源库逐格相等。
// 已知格式约束：CSV 无 NULL 语义，NULL与空串同形（本用例按文本等价类闭环——
// 需要严格 NULL 语义的场景应使用 JSON，由 TestITExportJsonReimportRoundtrip 钉死）
func TestITExportCsvReimportRoundtrip(t *testing.T) {
	src := itMysqlNode(t)
	defer src.Close()
	itFmtCreateMysql(t, src)

	buf := itDumpFormat(t, src, []string{itFmtTable}, "csv", itBatchSettings("csv"))
	records, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 6)

	tgt := itPgNode(t)
	defer tgt.Close()
	itFmtCreateTargetPg(t, tgt, "it_fmt_rt_csv")

	for _, rec := range records[1:] {
		_, err := tgt.Exec("INSERT INTO it_fmt_rt_csv (id, v_str, v_rfc, v_cn, v_small, v_big, v_dec, v_dt, v_date, v_txt, v_bin) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
			itParseIntOpt(t, rec[0]), rec[1], rec[2], rec[3],
			itParseIntOpt(t, rec[4]), itParseIntOpt(t, rec[5]), itOptText(rec[6]),
			itOptText(rec[7]), itOptText(rec[8]), rec[9], itHexToBytes(t, rec[10]))
		require.NoError(t, err, "CSV 行回灌 PG 失败（形态不被强类型接受）: %v", rec)
	}

	expected := itFmtExpectedStrings(t, src, itFmtColumns)
	got := itFmtTextMatrix(t, tgt, "it_fmt_rt_csv")
	require.Len(t, got, len(expected))
	for i := range expected {
		for _, col := range itFmtColumns {
			assert.Equal(t, expected[i][col], got[i][col], "CSV闭环 第%d行[%s] 失真", i+1, col)
		}
	}
}

// TestITExportJsonReimportRoundtrip JSON导出→回灌SQLite→逐值闭环 + NULL严格语义验证：
// JSON 的 null 与 "" 可区分（CSV 不能），回灌后源 NULL 列必须仍为 SQL NULL——
// 这是「导出产物承载完整语义、可无损恢复」的最强证据
func TestITExportJsonReimportRoundtrip(t *testing.T) {
	src := itMysqlNode(t)
	defer src.Close()
	itFmtCreateMysql(t, src)

	buf := itDumpFormat(t, src, []string{itFmtTable}, "json", itBatchSettings("json"))
	arr := itJsonDecode(t, buf.String())
	require.Len(t, arr, 5)

	tgt := itSqliteNode(t)
	defer tgt.Close()
	qt := tgt.GetDialect().Quoter().QuoteIdent
	// v_dec 用 TEXT 承载：本用例验证「JSON 文本往返无损」；sqlite NUMERIC affinity 会把十进制
	// 转 REAL（-0.000001→-1e-06 展示失真），属弱类型目标的类型映射语义，由 types_roundtrip 用例
	// 以数值宽松相等专门覆盖，两种职责不混用
	mustItExec(t, tgt, "CREATE TABLE "+qt("it_fmt_rt_json")+" (id INTEGER PRIMARY KEY, v_str TEXT, v_rfc TEXT, v_cn TEXT, "+
		"v_small INTEGER, v_big INTEGER, v_dec TEXT, v_dt DATETIME(6), v_date DATE, v_txt TEXT, v_bin BLOB)")

	itJsonArg := func(t *testing.T, row map[string]any, col string, conv func(*testing.T, string) any) any {
		v := row[col]
		if v == nil {
			return nil
		}
		return conv(t, itTextOf(v))
	}
	for _, row := range arr {
		_, err := tgt.Exec("INSERT INTO it_fmt_rt_json (id, v_str, v_rfc, v_cn, v_small, v_big, v_dec, v_dt, v_date, v_txt, v_bin) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
			itJsonArg(t, row, "id", func(tt *testing.T, s string) any { return itParseIntOpt(tt, s) }),
			row["v_str"], row["v_rfc"], row["v_cn"],
			itJsonArg(t, row, "v_small", func(tt *testing.T, s string) any { return itParseIntOpt(tt, s) }),
			itJsonArg(t, row, "v_big", func(tt *testing.T, s string) any { return itParseIntOpt(tt, s) }),
			itJsonArg(t, row, "v_dec", func(_ *testing.T, s string) any { return itOptText(s) }),
			itJsonArg(t, row, "v_dt", func(_ *testing.T, s string) any { return itOptText(s) }),
			itJsonArg(t, row, "v_date", func(_ *testing.T, s string) any { return itOptText(s) }),
			row["v_txt"],
			itJsonArg(t, row, "v_bin", func(tt *testing.T, s string) any { return itHexToBytes(tt, s) }))
		require.NoError(t, err, "JSON 行回灌 SQLite 失败: %v", row)
	}

	expected := itFmtExpectedStrings(t, src, itFmtColumns)
	got := itFmtTextMatrix(t, tgt, "it_fmt_rt_json")
	require.Len(t, got, len(expected))
	for i := range expected {
		for _, col := range itFmtColumns {
			assert.Equal(t, expected[i][col], got[i][col], "JSON闭环 第%d行[%s] 失真", i+1, col)
		}
	}

	// NULL 严格性：row4（id=4）源为全 NULL，回灌后必须仍是 SQL NULL 而非空串（JSON null 语义闭环）
	_, rows, err := tgt.Query("SELECT v_str IS NULL AS is_null, v_dt IS NULL AS dt_null FROM " + qt("it_fmt_rt_json") + " WHERE id = 4")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	isNull, ok := value.ValToInt64(rows[0]["is_null"])
	require.True(t, ok, "IS NULL 判定返回值形态异常: %T", rows[0]["is_null"])
	assert.Equal(t, int64(1), isNull, "JSON null 回灌后必须为 SQL NULL（区别于空串）")
}

// ─────────────────── MSSQL 源导出 + identity 迁移闭环 ───────────────────

// TestITExportMssqlCsvJson MSSQL（特有返回形态：uniqueidentifier/money/datetime2）→ CSV/JSON。
// mssql 节点在容器不可用时 Skip，本用例为该方言补格式层运行时覆盖
func TestITExportMssqlCsvJson(t *testing.T) {
	src := itMssqlNode(t) // 不可用自动 Skip
	defer src.Close()
	q := src.GetDialect().Quoter().QuoteIdent
	table := "it_export_fmt_mssql"
	mustItExec(t, src, "IF OBJECT_ID('"+table+"', 'U') IS NOT NULL DROP TABLE "+q(table))
	// 类型选 mssql 独有形态：uniqueidentifier(CAST→string 列)、money(CAST→numeric 列)、datetime2(驱动返 string 直透)
	mustItExec(t, src, "CREATE TABLE "+q(table)+" (id INT PRIMARY KEY, v_txt NVARCHAR(200), v_dec NUMERIC(20,6), v_dt2 DATETIME2(3), v_uid UNIQUEIDENTIFIER)")
	mustItExec(t, src, "INSERT INTO "+q(table)+" VALUES (1, N'mssql 中文😀', 12345678.123456, '2024-02-29 10:20:30.123', '6F9619FF-8B86-D011-B42D-00C04FC964FF')")
	mustItExec(t, src, "INSERT INTO "+q(table)+" VALUES (2, N'nulls', NULL, NULL, NULL)")

	// CAST 归一读回期望值（与导出侧独立路径，构成交叉验证）
	_, expRows, err := src.Query("SELECT id, CAST(v_txt AS NVARCHAR(200)) AS v_txt, CAST(v_dec AS VARCHAR(40)) AS v_dec, CAST(v_dt2 AS VARCHAR(40)) AS v_dt2, CAST(v_uid AS VARCHAR(40)) AS v_uid FROM " + q(table) + " ORDER BY id")
	require.NoError(t, err)
	require.Len(t, expRows, 2)

	// CSV：BatchRows=1 强制逐行分批
	sCsv := export.DefaultSettings("csv")
	sCsv.BatchRows = 1
	bufC := itDumpFormatM(t, src, []string{table}, "csv", sCsv)
	records, err := csv.NewReader(strings.NewReader(bufC.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 3, "表头+2行")
	assert.Equal(t, itTextOf(expRows[0]["v_txt"]), records[1][1])
	assert.Equal(t, itTextOf(expRows[0]["v_dec"]), records[1][2], "money/numeric 导出文本与直读一致")
	assert.Equal(t, "6F9619FF-8B86-D011-B42D-00C04FC964FF", strings.ToUpper(records[1][4]), "uniqueidentifier 完整往返")
	assert.Equal(t, "", records[2][2], "mssql NULL→CSV空串")

	// JSON：合法 + 值与直读一致 + null 语义
	sJson := export.DefaultSettings("json")
	sJson.BatchRows = 1
	bufJ := itDumpFormatM(t, src, []string{table}, "json", sJson)
	require.True(t, json.Valid(bufJ.Bytes()), "mssql 源 JSON 产物必须合法: %.200s", bufJ.String())
	arr := itJsonDecode(t, bufJ.String())
	require.Len(t, arr, 2)
	assert.Equal(t, itTextOf(expRows[0]["v_dec"]), itTextOf(arr[0]["v_dec"]))
	assert.Nil(t, arr[1]["v_dec"], "mssql NULL→JSON null")
	assert.Contains(t, itTextOf(arr[0]["v_uid"]), "6F9619FF")
	assert.Equal(t, "mssql 中文😀", itTextOf(arr[0]["v_txt"]), "NVARCHAR 中文+emoji 无损")
}

// TestITExportMysqlIdentityCsvJsonRoundtrip identity 表 CSV/JSON 导出→回灌→identity 值保真闭环：
// 自增主键的导出产物必须携带原值且可显式恢复（迁移保号场景的格式层证据）
func TestITExportMysqlIdentityCsvJsonRoundtrip(t *testing.T) {
	src := itMysqlNode(t)
	defer src.Close()
	q := src.GetDialect().Quoter().QuoteIdent
	table := "it_export_identity"
	mustItExec(t, src, "DROP TABLE IF EXISTS "+q(table))
	mustItExec(t, src, "CREATE TABLE "+q(table)+" (id INT AUTO_INCREMENT PRIMARY KEY, v VARCHAR(50) NOT NULL)")
	mustItExec(t, src, "INSERT INTO "+q(table)+" (v) VALUES ('a'),('b'),('c')")

	// JSON：主键显式携带（导出层不吞 identity 列）
	buf := itDumpFormatM(t, src, []string{table}, "json", export.DefaultSettings("json"))
	arr := itJsonDecode(t, buf.String())
	require.Len(t, arr, 3)
	for i, row := range arr {
		assert.Equal(t, fmt.Sprintf("%d", i+1), itTextOf(row["id"]), "identity 主键值必须出现在导出产物（丢列=迁移无法保号）")
	}

	// CSV 回灌 sqlite：主键显式写入保号
	records, err := csv.NewReader(strings.NewReader(itDumpFormatM(t, src, []string{table}, "csv", export.DefaultSettings("csv")).String())).ReadAll()
	require.NoError(t, err)
	tgt := itSqliteNode(t)
	defer tgt.Close()
	qt := tgt.GetDialect().Quoter().QuoteIdent
	mustItExec(t, tgt, "CREATE TABLE "+qt(table)+" (id INTEGER PRIMARY KEY, v TEXT NOT NULL)")
	for _, rec := range records[1:] {
		_, err := tgt.Exec("INSERT INTO "+qt(table)+" (id, v) VALUES (?, ?)", itParseIntOpt(t, rec[0]), rec[1])
		require.NoError(t, err)
	}
	_, tgtRows, err := tgt.Query("SELECT id, v FROM " + qt(table) + " ORDER BY id")
	require.NoError(t, err)
	require.Len(t, tgtRows, 3)
	for i, r := range tgtRows {
		assert.Equal(t, int64(i+1), mustItInt64(t, r["id"]), "identity 保号回灌闭环")
	}
}

// mustItInt64 断言取值为整型形态并返回
func mustItInt64(t *testing.T, v any) int64 {
	t.Helper()
	n, ok := value.ValToInt64(v)
	require.True(t, ok, "取值形态异常: %T", v)
	return n
}

// itDumpFormatM 多列表导出（列集合非统一 itFmtColumns 的用例使用）
func itDumpFormatM(t *testing.T, conn *dbi.DbConn, tables []string, format string, settings *export.Settings) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName: conn.Info.Database, Tables: tables, DumpData: true,
		Writer: buf, ExportFormat: format, Settings: settings,
	}))
	return buf
}

// ─────────────────── 能力探测与过滤边界（第二轮补强） ───────────────────

// TestITExportCsvDumpDdlIgnored DumpDDL=true + 非脚本格式（CSV/JSON）：
// DDL/索引段被 SupportsScript 能力探测静默跳过（格式承载不了 DDL 是事实），
// 但数据段必须完整不受影响——钉死能力探测的边界行为，防止将来把 DDL 写进 CSV 造成脏产物
func TestITExportCsvDumpDdlIgnored(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)

	for _, format := range []string{"csv", "json"} {
		buf := &bytes.Buffer{}
		require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
			Tables: []string{itFmtTable}, DumpDDL: true, DumpData: true,
			Writer: buf, ExportFormat: format,
		}))
		out := buf.String()
		up := strings.ToUpper(out)
		assert.NotContains(t, up, "CREATE TABLE", "格式[%s]不得混入 DDL", format)
		assert.NotContains(t, up, "DROP TABLE", "格式[%s]不得混入 DROP", format)
		assert.NotContains(t, up, "INDEX", "格式[%s]不得混入索引语句", format)
	}

	// 对照：数据完整未受 DumpDDL 影响
	csvBuf := &bytes.Buffer{}
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		Tables: []string{itFmtTable}, DumpDDL: true, DumpData: true, Writer: csvBuf, ExportFormat: "csv",
	}))
	records, err := csv.NewReader(strings.NewReader(csvBuf.String())).ReadAll()
	require.NoError(t, err)
	assert.Len(t, records, 6, "DumpDDL=true 不影响 CSV 数据行数")
}

// TestITExportTableFilterCsvJson 表级 where 过滤（大表分片迁移的底层能力）在非脚本格式路径同样生效
func TestITExportTableFilterCsvJson(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)

	// CSV：id<=3
	sCsv := export.DefaultSettings("csv")
	bufC := &bytes.Buffer{}
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		Tables: []string{itFmtTable}, DumpData: true, Writer: bufC, ExportFormat: "csv", Settings: sCsv,
		TableFilter: map[string]string{itFmtTable: "id <= 3"},
	}))
	records, err := csv.NewReader(strings.NewReader(bufC.String())).ReadAll()
	require.NoError(t, err)
	assert.Len(t, records, 4, "CSV TableFilter 生效：表头+3行")

	// JSON：v_small > 1
	bufJ := &bytes.Buffer{}
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		Tables: []string{itFmtTable}, DumpData: true, Writer: bufJ, ExportFormat: "json",
		TableFilter: map[string]string{itFmtTable: "v_small > 1"},
	}))
	arr := itJsonDecode(t, bufJ.String())
	for _, row := range arr {
		if row["v_small"] != nil {
			n, ok := value.ValToInt64(row["v_small"])
			require.True(t, ok)
			assert.Greater(t, n, int64(1), "JSON TableFilter 生效：v_small>1")
		}
	}
}

// TestITExportCsvMultiTablePerTableHeader 多表 CSV：每表独立表头分段（文本格式固有形态），
// 消费方按表头行切段；钉死不误并表/不误吞表头
func TestITExportCsvMultiTablePerTableHeader(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()
	itFmtCreateMysql(t, conn)
	q := conn.GetDialect().Quoter().QuoteIdent
	mustItExec(t, conn, "DROP TABLE IF EXISTS "+q("it_fmt_csv_second"))
	mustItExec(t, conn, "CREATE TABLE "+q("it_fmt_csv_second")+" (id INT PRIMARY KEY, w VARCHAR(10))")
	mustItExec(t, conn, `INSERT INTO `+q("it_fmt_csv_second")+` VALUES (7,'x'),(8,'y')`)

	buf := itDumpFormat(t, conn, []string{itFmtTable, "it_fmt_csv_second"}, "csv", export.DefaultSettings("csv"))
	records, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err)
	// 表头1 + 数据5 + 表头2 + 数据2 = 10
	require.Len(t, records, 10, "多表 CSV 必须每表独立表头分段")
	assert.Equal(t, itFmtColumns, records[0], "第一表表头")
	assert.Equal(t, []string{"id", "w"}, records[6], "第二表表头紧随第一表数据")
	assert.Equal(t, "7", records[7][0])
	assert.Equal(t, "y", records[8][1])
}
