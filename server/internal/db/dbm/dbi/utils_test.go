package dbi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuoteEscape(t *testing.T) {
	kases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"abc", "abc"},
		{"it's", "it''s"},
		{"a'b'c", "a''b''c"},
		{"''", "''''"},
		{`a"b`, `a"b`}, // 双引号不处理
		{"'; DROP TABLE users; --", "''; DROP TABLE users; --"},
	}

	for _, k := range kases {
		assert.Equal(t, k.expected, QuoteEscape(k.input))
	}
}

// TestQuoteEscapeNulByte 标准SQL语义下含NUL(0x00)文本必须快速失败而非生成损坏SQL；
// mysql语义下\0转义可无损承载NUL。
// 回归背景：实测sqlite tokenizer按C字符串语义遇NUL截断，dump产物中含NUL字面量
// 会静默产生损坏语句并在导入时损毁后续语句（大数据量IT抓出）
func TestQuoteEscapeNulByte(t *testing.T) {
	assert.PanicsWithValue(t,
		"text contains NUL(0x00) byte which cannot be carried by SQL script stream; refuse to generate corrupted SQL, value prefix: \"null\\x00byte\"",
		func() { QuoteEscape("null\x00byte") })
	// mysql语义：\0转义可无损承载NUL，且字面"\0"文本（反斜杠+0）不受影响
	assert.Equal(t, `nul\0end`, QuoteEscapeBackslash("nul\x00end"))
	assert.Equal(t, `\\0`, QuoteEscapeBackslash(`\0`))
	assert.Equal(t, `a\0b`, QuoteEscapeBackslash("a\x00b"))
	require.NotPanics(t, func() { QuoteEscape("normal text with emoji😀") })
}

// TestQuoteEscapeBackslash mysql字面量中反斜杠是转义符，必须与单引号同时转义，
// 且两者顺序无关（单引号双写引入的都是单引号字符，不会被反斜杠替换影响）
func TestQuoteEscapeBackslash(t *testing.T) {
	kases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"abc", "abc"},
		{`a\b`, `a\\b`},
		{`C:\path\to\file`, `C:\\path\\to\\file`},
		{`line1\nline2`, `line1\\nline2`},
		{"it's", "it''s"},
		{`it\'s`, `it\\''s`},
		{`\\`, `\\\\`},
		{`endslash\`, `endslash\\`},
		{"a\nb", "a\nb"}, // 真实换行符不属于反斜杠转义，保持原样
	}

	for _, k := range kases {
		assert.Equal(t, k.expected, QuoteEscapeBackslash(k.input))
	}
}

// TestQuoteEscapeBackslash_RoundTripMysqlLiteral 按mysql字面量解析语义（反斜杠转义优先、
// 其次单引号双写）解析转义后的内容，必须还原出原始值，确保不出现失真或提前闭合
func TestQuoteEscapeBackslash_RoundTripMysqlLiteral(t *testing.T) {
	samples := []string{
		"", "plain", "it's", `a"b`, `C:\path\'quoted`, "line1\nline2\r\nline3",
		"tab\there", "\\\\", "a\\b", `'; DROP TABLE users; --`, "中文\u0001字节\\尾",
		strings.Repeat("'a\\b", 100),
	}

	for _, origin := range samples {
		escaped := QuoteEscapeBackslash(origin)
		// 转义结果内不得存在裸反斜杠与裸单引号，否则字面量会失真或提前闭合
		for i := 0; i < len(escaped); i++ {
			switch escaped[i] {
			case '\\':
				assert.Equal(t, byte('\\'), escaped[i+1], "反斜杠必须成对出现")
				i++
			case '\'':
				assert.Equal(t, byte('\''), escaped[i+1], "单引号必须成对出现")
				i++
			}
		}
		assert.Equal(t, origin, parseMysqlLiteral(escaped), "mysql语义解析后必须等于原始值")
		// 标准SQL语义（无NO_BACKSLASH_ESCAPES场景，如pg/sqlite）：只双写单引号，反斜杠保持字面
		assert.Equal(t, origin, parseStdLiteral(QuoteEscape(origin)), "标准SQL语义解析后必须等于原始值")
	}
}

// TestSanitizeCommentText 嵌入行注释的文本不得含换行类字符（否则注释提前结束，剩余内容成为可执行语句）
func TestSanitizeCommentText(t *testing.T) {
	kases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"tbl", "tbl"},
		{"a\nb", "a b"},
		{"a\r\nb", "a  b"},
		{"a\x00b", "a b"},
		{"it_cxhdr_ok\nDROP TABLE x", "it_cxhdr_ok DROP TABLE x"},
		{"a\u2028b", "a b"},
		{"含中文;分号--x", "含中文;分号--x"}, // 非换行字符保持原样（已处于注释内，无需转义）
	}

	for _, k := range kases {
		got := SanitizeCommentText(k.input)
		assert.Equal(t, k.expected, got)
		assert.NotContains(t, got, "\n")
		assert.NotContains(t, got, "\r")
	}
}

// parseMysqlLiteral 模拟mysql对字面量内容的解析：双写单引号与反斜杠转义单引号均为单引号，反斜杠加x为字符x
func parseMysqlLiteral(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s):
			i++
			switch s[i] {
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case '0':
				buf.WriteByte(0)
			default:
				buf.WriteByte(s[i])
			}
		case s[i] == '\'' && i+1 < len(s) && s[i+1] == '\'':
			i++
			buf.WriteByte('\'')
		default:
			buf.WriteByte(s[i])
		}
	}
	return buf.String()
}

// parseStdLiteral 模拟标准SQL对字面量内容的解析：仅双写单引号表示单引号
func parseStdLiteral(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' && i+1 < len(s) && s[i+1] == '\'' {
			i++
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
}

// TestIsQuotedSqlLiteral 字面量判定必须做完整的引号成对校验，不能用“首尾是引号”或子串正则代替
func TestIsQuotedSqlLiteral(t *testing.T) {
	kases := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"'", false},
		{"''", true}, // 空串字面量（必须与无默认值区分）
		{"'a'", true},
		{"'it''s'", true},
		{"'NULL'", true},
		{"'''quoted'''", true}, // 值本身首尾带引号的字面量
		{"'a", false},
		{"a'", false},
		{"a'b'c", false},         // 未包引号的原始值（MySQL 8.0形态），裸拼会语法错误
		{"'ab'cd", false},        // 引号提前闭合后仍有内容
		{"'abc' + 'def'", false}, // 拼接表达式不是单个字面量
		{`"abc"`, false},
	}
	for _, k := range kases {
		assert.Equal(t, k.want, IsQuotedSqlLiteral(k.input), "input=%q", k.input)
	}
}

// TestIsSqlFunctionExpr 函数形式判定不得误伤内容含括号的文本默认值
func TestIsSqlFunctionExpr(t *testing.T) {
	kases := []struct {
		input string
		want  bool
	}{
		{"now()", true},
		{"date_trunc('day', now)", true},
		{"TO_CHAR(SYSDATE,'YYYY')", true},
		{"pg_catalog.now()", true},
		{"dbo.f(1)", true},
		{"nextval('t_id_seq'::regclass)", true},
		{"(0)", false},               // 外层括号是表达式包装，不是函数调用
		{"unknown (pending)", false}, // 名称与括号间有空格的自由文本（MySQL 8.0去引号后的字面量内容）
		{"待确认(必填)", false},           // 非ASCII前缀不可能是函数名
		{"'a('", false},
		{"1+2", false},
		{"f(a", false}, // 未闭合
		{"", false},
	}
	for _, k := range kases {
		assert.Equal(t, k.want, IsSqlFunctionExpr(k.input), "input=%q", k.input)
	}
}

// TestUnwrapOuterParens 剔外层括号必须做括号平衡扫描，否则 (1)+(2) 会被剪成非法片段 1)+(2
func TestUnwrapOuterParens(t *testing.T) {
	kases := []struct {
		input   string
		inner   string
		trimmed bool
	}{
		{"('abc')", "'abc'", true},
		{"(N'abc')", "N'abc'", true},
		{"(3)", "3", true},
		{"((1+2))", "(1+2)", true},
		{"(a(b))", "a(b)", true},
		{"( getdate() )", "getdate()", true},
		{"('a(b')", "'a(b'", true}, // 字面量内部的括号不参与平衡计数
		{"(1)+(2)", "", false},
		{"abc", "", false},
		{"a)", "", false},
		{"(", "", false},
		{"", "", false},
	}
	for _, k := range kases {
		inner, ok := UnwrapOuterParens(k.input)
		assert.Equal(t, k.trimmed, ok, "input=%q", k.input)
		assert.Equal(t, k.inner, inner, "input=%q", k.input)
	}
}

// TestGenColumnDefaultSql 列默认值往DDL的全集回归：覆盖各库元数据的真实呈现形态
// （MySQL 8.0去引号裸值、5.7/MariaDB/达梦/sqlite带引号字面量、pg剥cast后的字面量、
// SQL Server/MySQL 8.0.13+带外层括号的定义原文），全部经真实库实测确认
func TestGenColumnDefaultSql(t *testing.T) {
	escape := QuoteEscape
	kases := []struct {
		name     string
		dataType string
		raw      string
		expected string
	}{
		// ---- 无默认值/NULL ----
		{"空原文不输出", "varchar", "", ""},
		{"裸NULL（无默认值）不输出", "varchar", "NULL", ""},
		{"字符串NULL默认值保留", "varchar", "'NULL'", " DEFAULT 'NULL'"},

		// ---- 带引号字面量形态（5.7/MariaDB/sqlite/达梦/pg） ----
		{"字面量-普通", "varchar", "'abc'", " DEFAULT 'abc'"},
		{"字面量-空串与无默认值可区分", "varchar", "''", " DEFAULT ''"},
		{"字面量-含单引号", "varchar", "'it''s'", " DEFAULT 'it''s'"},
		{"字面量-含括号", "varchar", "'(0)'", " DEFAULT '(0)'"},
		{"字面量-中文含括号", "varchar", "'待确认(必填)'", " DEFAULT '待确认(必填)'"},
		{"字面量-json内容", "jsonb", `'{"k": 1}'`, ` DEFAULT '{"k": 1}'`},
		{"字面量-含分号不得逃逸", "varchar", "'a; DROP TABLE t; --'", " DEFAULT 'a; DROP TABLE t; --'"},

		// ---- MySQL 8.0去引号裸值形态 ----
		{"裸值-普通", "varchar", "abc", " DEFAULT 'abc'"},
		{"裸值-含单引号", "varchar", "it's", " DEFAULT 'it''s'"},
		{"裸值-含括号函数形不成函数", "varchar", "unknown (pending)", " DEFAULT 'unknown (pending)'"},
		{"裸值-中文含括号", "varchar", "待确认(必填)", " DEFAULT '待确认(必填)'"},
		{"裸值-仅首尾括号按字面量", "varchar", "(0)", " DEFAULT '(0)'"},
		{"裸值-纯空白默认值必须存活", "varchar", " ", " DEFAULT ' '"},
		{"裸值-首尾空白属于值本身不得剥除", "varchar", "  x  ", " DEFAULT '  x  '"},
		{"裸值-反斜杠在标准SQL方言不转义", "varchar", `a\b`, ` DEFAULT 'a\b'`},
		{"裸值-制表符与多空格原文保留", "varchar", " \t ", " DEFAULT ' \t '"},
		{"裸值-尾部单引号不丢字符", "varchar", "end'", " DEFAULT 'end'''"},

		// ---- 可证明的裸字面量/无括号关键字 ----
		{"数字裸拼", "bigint", "3", " DEFAULT 3"},
		{"负数字裸拼", "int", "-1", " DEFAULT -1"},
		{"浮点裸拼", "decimal", "1.5", " DEFAULT 1.5"},
		{"十六进制裸拼(mysql binary)", "binary", "0x1234", " DEFAULT 0x1234"},
		{"位字面量裸拼(mysql bit)", "bit", "b'01'", " DEFAULT b'01'"},
		{"CURRENT_TIMESTAMP裸拼", "timestamp", "CURRENT_TIMESTAMP", " DEFAULT CURRENT_TIMESTAMP"},
		{"SYSDATE归一为日期分量(oracle date列)", "date", "SYSDATE", " DEFAULT CURRENT_DATE"},
		{"日期字面量必须引用", "datetime", "2020-01-01 00:00:00", " DEFAULT '2020-01-01 00:00:00'"},
		{"日期时间ISO形引用", "timestamp", "2020-01-01T10:00:00.123", " DEFAULT '2020-01-01T10:00:00.123'"},

		// ---- 函数/表达式默认值：跳而不产生非法DDL ----
		{"其他函数默认值跳过", "timestamp", "abs(-1)", ""},
		{"限定名函数跳过", "timestamp", "pg_catalog.now()", ""},
		{"日期表达式不得写成字符串", "date", "sysdate+1", ""},
		{"序列默认值跳过", "int8", "nextval('t_id_seq'::regclass)", ""},
		// 已知限制：字符串列上的裸表达式（Oracle的USER、'a' || 'b'）与MySQL 8.0去引号后的
		// 字面量内容（DEFAULT 'USER'）形态完全相同且无法区分，宁可当成字面量也不能全丢
		{"字符串列上的裸表达式按字面量保留", "varchar2", "USER", " DEFAULT 'USER'"},
		{"字符串列上的拼接表达式按字面量保留", "varchar", "'a' || 'b'", " DEFAULT '''a'' || ''b'''"},

		// ---- 括号包裹的定义原文（SQL Server / MySQL 8.0.13+） ----
		{"括号字面量还原", "varchar", "('abc')", " DEFAULT 'abc'"},
		{"括号N前缀字面量还原", "nvarchar", "(N'abc')", " DEFAULT 'abc'"},
		{"括号包裹的getdate()剥层归一", "datetime", "(getdate())", " DEFAULT CURRENT_TIMESTAMP"},
		{"括号内层含引号重新转义", "varchar", "('it''s')", " DEFAULT 'it''s'"},
		{"括号数字在整型列上裸拼", "int", "(0)", " DEFAULT 0"},
		{"多层括号表达式跳过", "int", "((1+2))", ""},
		{"括号函数跳过", "datetime", "(date_trunc('day',now()))", ""},
		{"非整体括号剪碎后跳过", "int", "(1)+(2)", ""},
		{"整型列不可识别裸文本跳过", "int", "abc", ""},

		// ---- 与SQL关键字/数字同形的字符串默认值（MySQL 8.0去引号呈现）：必须引用，否则目标库重新解释 ----
		{"字符串列默认值恰为CURRENT_TIMESTAMP必须引用", "varchar", "CURRENT_TIMESTAMP", " DEFAULT 'CURRENT_TIMESTAMP'"},
		{"字符串列默认值恰为TRUE必须引用", "varchar", "TRUE", " DEFAULT 'TRUE'"},
		{"字符串列默认值恰为FALSE必须引用", "char", "FALSE", " DEFAULT 'FALSE'"},
		{"字符串列默认值恰为数字必须引用", "text", "0", " DEFAULT '0'"},
		{"enum列默认值恰为关键字必须引用", "enum", "NULL_", " DEFAULT 'NULL_'"},
		// MySQL 8.0去引号呈现使 varchar DEFAULT 'now()' 与裸函数调用同形：字符串列的表达式默认值
		// 必须书写为(concat(...))形态，故裸函数形只能是字面量内容，当作函数丢弃会使非空列无默认值
		{"字符串列默认值恰为函数调用形态必须引用", "varchar", "now()", " DEFAULT 'now()'"},
		{"大字列默认值恰为函数调用形态必须引用", "text", "uuid_generate_v4()", " DEFAULT 'uuid_generate_v4()'"},
		{"二进制列的0x形态仍裸拼", "varbinary", "0x1234", " DEFAULT 0x1234"},

		// ---- 「当前日期/时间」默认值按目标列类型归一（多源多目标语法差异） ----
		{"源带fsp归一为无参关键字", "datetime", "CURRENT_TIMESTAMP(3)", " DEFAULT CURRENT_TIMESTAMP"},
		{"MySQL8.0的curdate()归一", "date", "curdate()", " DEFAULT CURRENT_DATE"},
		{"MySQL8.0的curtime()归一", "time", "curtime()", " DEFAULT CURRENT_TIME"},
		{"now()归一为CURRENT_TIMESTAMP", "timestamp", "now()", " DEFAULT CURRENT_TIMESTAMP"},
		{"SYSDATE入日期时间列保留完整语义", "datetime", "SYSDATE", " DEFAULT CURRENT_TIMESTAMP"},
		{"时间戳默认值入纯日期列取日期分量", "date", "CURRENT_TIMESTAMP", " DEFAULT CURRENT_DATE"},
		{"时间戳默认值入纯时间列取时间分量", "time", "CURRENT_TIMESTAMP", " DEFAULT CURRENT_TIME"},
		{"小写裸关键字归一", "datetime", "current_timestamp", " DEFAULT CURRENT_TIMESTAMP"},
	}

	for _, k := range kases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.expected, GenColumnDefaultSql(k.raw, k.dataType, escape))
		})
	}
}

// TestGenColumnDefaultSql_MysqlEscape mysql目标必须额外双写反斜杠（默认未开启NO_BACKSLASH_ESCAPES）
func TestGenColumnDefaultSql_MysqlEscape(t *testing.T) {
	assert.Equal(t, ` DEFAULT 'a\\b'`, GenColumnDefaultSql(`a\b`, "varchar", QuoteEscapeBackslash))
	assert.Equal(t, ` DEFAULT 'a\\b'`, GenColumnDefaultSql(`'a\b'`, "varchar", QuoteEscapeBackslash))
	assert.Equal(t, ` DEFAULT 'it\\''s'`, GenColumnDefaultSql(`it\'s`, "varchar", QuoteEscapeBackslash))
	// 标准SQL方言（pg/sqlite）反斜杠保持字面量
	assert.Equal(t, ` DEFAULT 'a\b'`, GenColumnDefaultSql(`a\b`, "varchar", QuoteEscape))
}

// TestGenColumnDefaultSql_LiteralRoundTrip 生成的DEFAULT子句内容按目标方言解析后必须等于原始默认值：
// 分别验证“源为带引号字面量”与“源为MySQL 8.0去引号裸值”（已实测：8.0的COLUMN_DEFAULT就是
// 未加引号未转义的原始值）两种形态，以及目标为mysql与目标为标准SQL两条路径
func TestGenColumnDefaultSql_LiteralRoundTrip(t *testing.T) {
	originals := []string{
		"abc", "it's", "", " ", "  x  ", "NULL", "(0)", "unknown (pending)", "待确认(必填)",
		`a\b`, `{"k": "v's"}`, "line1\nline2", `'; DROP TABLE users; --`, `a"b"c`, "'x\\b'x",
		strings.Repeat("'a\\b", 50),
	}

	for _, origin := range originals {
		// 源为带引号字面量（pg剥cast后、sqlite、达梦、MySQL 5.7）→ 目标为标准SQL方言
		if stdRaw := "'" + QuoteEscape(origin) + "'"; IsQuotedSqlLiteral(stdRaw) {
			got := GenColumnDefaultSql(stdRaw, "varchar", QuoteEscape)
			assert.Equal(t, origin, parseStdLiteral(defaultLiteralBody(t, got)), "字面量形态→标准SQL方言往返不等, raw=%q", stdRaw)
		}

		if origin == "" {
			assert.Equal(t, "", GenColumnDefaultSql(origin, "varchar", QuoteEscape), "空原文即无默认值")
			continue
		}
		if origin == "NULL" {
			// 已知限制：MySQL 8.0把 DEFAULT 'NULL' 与无默认值同样呈现为裸NULL，原理上不可区分，只能丢弃
			assert.Equal(t, "", GenColumnDefaultSql(origin, "varchar", QuoteEscape), "NULL裸形态无法与无默认值区分")
			continue
		}

		// 源为MySQL 8.0裸值 → 目标为mysql（反斜杠与单引号均需双写）
		gotMysql := GenColumnDefaultSql(origin, "varchar", QuoteEscapeBackslash)
		assert.Equal(t, origin, parseMysqlLiteral(defaultLiteralBody(t, gotMysql)), "裸值形态→mysql方言往返不等, origin=%q", origin)
		// 源为MySQL 8.0裸值 → 目标为标准SQL方言（反斜杠不是转义符）
		gotStd := GenColumnDefaultSql(origin, "varchar", QuoteEscape)
		assert.Equal(t, origin, parseStdLiteral(defaultLiteralBody(t, gotStd)), "裸值形态→标准SQL方言往返不等, origin=%q", origin)
	}
}

// defaultLiteralBody 取字符串引用形态的DEFAULT子句内容（剥去前缀与首尾引用符），非引用形态直接失败
func defaultLiteralBody(t *testing.T, defaultSql string) string {
	t.Helper()
	if !strings.HasPrefix(defaultSql, " DEFAULT '") || !strings.HasSuffix(defaultSql, "'") {
		t.Fatalf("默认值未按字符串字面量引用: %q", defaultSql)
	}
	return strings.TrimSuffix(strings.TrimPrefix(defaultSql, " DEFAULT '"), "'")
}

// TestParseTimeKeywordDefault 「当前日期/时间」类默认值原文识别：各源库形态各异，
// 必须同时认出标准关键字、小写形态、带空括号与带小数秒参数的写法，且不能误认任意表达式
func TestParseTimeKeywordDefault(t *testing.T) {
	kases := []struct {
		raw     string
		keyword string
		parts   int
		fsp     int
		ok      bool
	}{
		{"CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP", TimePartDate | TimePartTime, -1, true},
		{"current_timestamp", "CURRENT_TIMESTAMP", TimePartDate | TimePartTime, -1, true},
		{" CURRENT_TIMESTAMP ", "CURRENT_TIMESTAMP", TimePartDate | TimePartTime, -1, true},
		{"now()", "NOW", TimePartDate | TimePartTime, -1, true},
		{"NOW(6)", "NOW", TimePartDate | TimePartTime, 6, true},
		{"CURRENT_TIMESTAMP(3)", "CURRENT_TIMESTAMP", TimePartDate | TimePartTime, 3, true},
		{"SYSDATE", "SYSDATE", TimePartDate | TimePartTime, -1, true},
		{"SYSTIMESTAMP", "SYSTIMESTAMP", TimePartDate | TimePartTime, -1, true},
		{"getdate", "GETDATE", TimePartDate | TimePartTime, -1, true},
		{"CURRENT_DATE", "CURRENT_DATE", TimePartDate, -1, true},
		{"curdate()", "CURDATE", TimePartDate, -1, true},
		{"CURRENT_TIME(2)", "CURRENT_TIME", TimePartTime, 2, true},
		{"curtime", "CURTIME", TimePartTime, -1, true},
		// 非「当前日期/时间」语义：带实参、含运算、限定名、被括号包裹的形态均不识别
		{"TO_CHAR(SYSDATE,'YYYY')", "", 0, -1, false},
		{"pg_catalog.now()", "", 0, -1, false},
		{"sysdate+1", "", 0, -1, false},
		{"(getdate())", "", 0, -1, false},
		{"CURRENT_TIMESTAMP(12)", "CURRENT_TIMESTAMP", TimePartDate | TimePartTime, 12, true}, // 超上限的精度仍可识别，由调用方按目标库上限夹取
		{"CURRENT_TIMESTAMP(x)", "", 0, -1, false},
		{"'NOW'", "", 0, -1, false},
		{"abc", "", 0, -1, false},
		{"", "", 0, -1, false},
	}

	for _, k := range kases {
		keyword, parts, fsp, ok := ParseTimeKeywordDefault(k.raw)
		assert.Equal(t, k.ok, ok, "raw=%q", k.raw)
		assert.Equal(t, k.keyword, keyword, "raw=%q", k.raw)
		assert.Equal(t, k.parts, parts, "raw=%q", k.raw)
		assert.Equal(t, k.fsp, fsp, "raw=%q", k.raw)
	}
}

// TestCanonicalTimeKeywordDefault 按目标列类型归一：只有日期时间类列才允许把默认值改写成SQL关键字，
// 字符串列的默认值内容可能恰是该文本；分量按目标列可承载部分取舍
func TestCanonicalTimeKeywordDefault(t *testing.T) {
	kases := []struct {
		raw      string
		dataType string
		expected string
		ok       bool
	}{
		{"now()", "datetime", "CURRENT_TIMESTAMP", true},
		{"CURRENT_TIMESTAMP(6)", "timestamp", "CURRENT_TIMESTAMP", true},
		{"SYSDATE", "datetime", "CURRENT_TIMESTAMP", true},
		{"SYSDATE", "date", "CURRENT_DATE", true},
		{"CURRENT_TIMESTAMP", "date", "CURRENT_DATE", true},
		{"CURRENT_TIMESTAMP", "time", "CURRENT_TIME", true},
		{"curtime()", "time", "CURRENT_TIME", true},
		{"getdate", "smalldatetime", "CURRENT_TIMESTAMP", true},
		{"LOCALTIMESTAMP", "timestamptz", "CURRENT_TIMESTAMP", true},
		// 非日期时间列不改写：内容可能就是这段文本
		{"now()", "varchar", "", false},
		{"CURRENT_TIMESTAMP", "text", "", false},
		{"SYSDATE", "int", "", false},
		{"TO_CHAR(SYSDATE,'YYYY')", "datetime", "", false},
		{"sysdate+1", "date", "", false},
	}

	for _, k := range kases {
		got, ok := CanonicalTimeKeywordDefault(k.raw, k.dataType)
		assert.Equal(t, k.ok, ok, "raw=%q type=%q", k.raw, k.dataType)
		assert.Equal(t, k.expected, got, "raw=%q type=%q", k.raw, k.dataType)
	}
}

// TestTimePartsOfBaseType 目标列基础类型的分量判定：日期/时间/日期时间合一
func TestTimePartsOfBaseType(t *testing.T) {
	assert.Equal(t, TimePartDate, TimePartsOfBaseType("date"))
	assert.Equal(t, TimePartDate, TimePartsOfBaseType("DATE"))
	assert.Equal(t, TimePartTime, TimePartsOfBaseType("time"))
	assert.Equal(t, TimePartTime, TimePartsOfBaseType("timetz"))
	assert.Equal(t, TimePartDate|TimePartTime, TimePartsOfBaseType("datetime"))
	assert.Equal(t, TimePartDate|TimePartTime, TimePartsOfBaseType("timestamp"))
	assert.Equal(t, TimePartDate|TimePartTime, TimePartsOfBaseType("timestamptz"))
	// 无法识别的日期时间形态按合一处理，宁可多保留信息
	assert.Equal(t, TimePartDate|TimePartTime, TimePartsOfBaseType("smalldatetime"))
}
