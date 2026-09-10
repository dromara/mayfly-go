package sqlparser

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// splitCase 复杂字符串/标识符场景的切割用例：mysql 与标准SQL 语义差异通过独立字段声明，
// 两方言期望一致时只写 both；某方言在此输入下必须报未闭合错误时写该方言的 errKind
type splitCase struct {
	name  string
	sql   string
	both  []string // 共同期望
	mysql []string // 仅mysql语义期望（nil表示用both）
	std   []string // 仅标准SQL语义期望（nil表示用both）

	mysqlErrKind string // 非空：mysql语义下期望 *tokenizer.UnterminatedError 的 Kind
	stdErrKind   string // 非空：标准SQL语义下期望未闭合错误
}

func runSplitCases(t *testing.T, cases []splitCase) {
	t.Helper()
	for _, k := range cases {
		t.Run(k.name, func(t *testing.T) {
			mysqlWant, stdWant := k.both, k.both
			if k.mysql != nil {
				mysqlWant = k.mysql
			}
			if k.std != nil {
				stdWant = k.std
			}
			// 未声明期望的用例断言会静默通过，必须显式要求声明
			require.True(t, mysqlWant != nil || k.mysqlErrKind != "", "用例必须声明mysql期望或错误类型")
			require.True(t, stdWant != nil || k.stdErrKind != "", "用例必须声明std期望或错误类型")

			if k.mysqlErrKind != "" {
				_, err := splitErrAll(t, mysqlSplitter(), k.sql)
				assertUnterminated(t, err, 1, k.mysqlErrKind)
			} else {
				assert.Equal(t, mysqlWant, splitAll(t, mysqlSplitter(), k.sql), "mysql语义切割")
			}
			if k.stdErrKind != "" {
				_, err := splitErrAll(t, stdSplitter(), k.sql)
				assertUnterminated(t, err, 1, k.stdErrKind)
			} else {
				assert.Equal(t, stdWant, splitAll(t, stdSplitter(), k.sql), "标准SQL语义切割")
			}
		})
	}
}

// 复杂字符串/标识符场景的语句切割测试：dump 产物包含真实换行、引号内分号、注释样式文本、
// SQL标准单引号双写与 mysql 反斜杠转义、反引号/双引号标识符，切割结果必须与之严格一致，
// 否则迁移导入时会错切语句导致数据丢失或语法错误
func TestSplitterComplexValues(t *testing.T) {
	runSplitCases(t, []splitCase{
		{
			name: "多行INSERT值内含真实换行",
			sql:  "INSERT INTO t (a) VALUES \n('line1\nline2'),\n('b');",
			both: []string{"INSERT INTO t (a) VALUES \n('line1\nline2'),\n('b')"},
		},
		{
			name: "值含回车换行CRLF",
			sql:  "INSERT INTO t (a) VALUES ('l1\r\nl2\r\n');",
			both: []string{"INSERT INTO t (a) VALUES ('l1\r\nl2\r\n')"},
		},
		{
			name: "SQL标准双写单引号内含分号",
			sql:  `INSERT INTO t (a) VALUES ('it''s;ok');`,
			both: []string{`INSERT INTO t (a) VALUES ('it''s;ok')`},
		},
		{
			name: "值以双写引号结尾",
			sql:  `INSERT INTO t (a) VALUES ('abc'''); INSERT INTO t (a) VALUES (1);`,
			both: []string{`INSERT INTO t (a) VALUES ('abc''')`, `INSERT INTO t (a) VALUES (1)`},
		},
		{
			name: "值含行注释样式文本",
			sql:  `INSERT INTO t (a) VALUES ('a--b; c');`,
			both: []string{`INSERT INTO t (a) VALUES ('a--b; c')`},
		},
		{
			name: "值含块注释样式文本",
			sql:  `INSERT INTO t (a) VALUES ('/*x;y*/');`,
			both: []string{`INSERT INTO t (a) VALUES ('/*x;y*/')`},
		},
		{
			name: "值含JSON双引号与分号",
			sql:  `INSERT INTO t (a) VALUES ('{"k":"a;b","msg":"say \"hi\""}');`,
			both: []string{`INSERT INTO t (a) VALUES ('{"k":"a;b","msg":"say \"hi\""}')`},
		},
		{
			name: "值含HTML与script标签",
			sql:  `INSERT INTO t (a) VALUES ('<script>alert(''x'')</script>;');`,
			both: []string{`INSERT INTO t (a) VALUES ('<script>alert(''x'')</script>;')`},
		},
		{
			name: "值含中文全角与emoji",
			sql:  "INSERT INTO t (a) VALUES ('中文，ＢＢＣ😀;全角分号');",
			both: []string{"INSERT INTO t (a) VALUES ('中文，ＢＢＣ😀;全角分号')"},
		},
		{
			name: "双引号标识符含分号",
			sql:  `INSERT INTO "my;t" (a) VALUES (1);`,
			// 标准SQL中双引号是标识符引用符；mysql中是字符串字面量，两者均不在此切割
			both: []string{`INSERT INTO "my;t" (a) VALUES (1)`},
		},
		{
			name: "值内单反引号",
			sql:  "INSERT INTO t (a) VALUES ('a`b;c');",
			both: []string{"INSERT INTO t (a) VALUES ('a`b;c')"},
		},
		{
			name: "多语句混合复杂值",
			sql: `INSERT INTO t (a) VALUES ('x;y');
INSERT INTO t (a) VALUES ('it''s');
INSERT INTO t (a) VALUES ('multi
line');`,
			both: []string{`INSERT INTO t (a) VALUES ('x;y')`, `INSERT INTO t (a) VALUES ('it''s')`, "INSERT INTO t (a) VALUES ('multi\nline')"},
		},
		{
			name:  "反引号标识符含分号（mysql方言引号）",
			sql:   "INSERT INTO `my;t` (a) VALUES (1);",
			mysql: []string{"INSERT INTO `my;t` (a) VALUES (1)"},
			// 标准SQL无反引号语法，反引号视为普通字符，分号仍作为切割符
			std: []string{"INSERT INTO `my", "t` (a) VALUES (1)"},
		},
		{
			name:       "反引号标识符含单引号",
			sql:        "INSERT INTO `it's` (a) VALUES (1);",
			mysql:      []string{"INSERT INTO `it's` (a) VALUES (1)"},
			stdErrKind: tokenizer.KindString, // 标准SQL不识别反引号，单引号开串后至EOF未闭合
		},
		{
			name:  "反引号标识符内含分号的多语句",
			sql:   "SELECT * FROM `a;b`; SELECT * FROM `c;d`;",
			mysql: []string{"SELECT * FROM `a;b`", "SELECT * FROM `c;d`"},
			// 标准SQL无反引号语义，各分号均视为语句结束符
			std: []string{"SELECT * FROM `a", "b`", "SELECT * FROM `c", "d`"},
		},
	})
}

// mysql 反斜杠转义语义下的复杂值：\' 不结束字符串，\\ 为字面反斜杠
func TestSplitterMysqlBackslashEscapeValues(t *testing.T) {
	runSplitCases(t, []splitCase{
		{
			name:  "值含转义单引号与分号",
			sql:   `INSERT INTO t (a) VALUES ('a\'b;c');`,
			mysql: []string{`INSERT INTO t (a) VALUES ('a\'b;c')`},
			// 标准SQL中反斜杠为普通字符，\' 即结束字符串，其后的 ; 触发切割；
			// 余下 c'); 中 ' 重新开串后直到EOF未闭合，必须报错而非把残缺语句交给执行方
			stdErrKind: tokenizer.KindString,
		},
		{
			name:  "值以双反斜杠结尾",
			sql:   `INSERT INTO t (a) VALUES ('a\\'); INSERT INTO t (a) VALUES (2);`,
			mysql: []string{`INSERT INTO t (a) VALUES ('a\\')`, `INSERT INTO t (a) VALUES (2)`},
			std:   []string{`INSERT INTO t (a) VALUES ('a\\')`, `INSERT INTO t (a) VALUES (2)`},
		},
		{
			name:         "pg形态值以单反斜杠结尾",
			sql:          `INSERT INTO t (a) VALUES ('a\'); INSERT INTO t (a) VALUES (2);`,
			mysqlErrKind: tokenizer.KindString, // mysql语义下 \' 不结束字符串，最终未闭合
			std:          []string{`INSERT INTO t (a) VALUES ('a\')`, `INSERT INTO t (a) VALUES (2)`},
		},
		{
			name:  "JSON含转义斜杠路径",
			sql:   `INSERT INTO t (a) VALUES ('{"p":"C:\\tmp\\a;b"}');`,
			mysql: []string{`INSERT INTO t (a) VALUES ('{"p":"C:\\tmp\\a;b"}')`},
			std:   []string{`INSERT INTO t (a) VALUES ('{"p":"C:\\tmp\\a;b"}')`},
		},
	})
}

// dump 脚本形态：注释归属其后的语句并原样保留，纯注释段与多余分隔符不产生语句
func TestSplitterDumpScript(t *testing.T) {
	dumpHeader := `
-- ----------------------------
-- Dump Platform: mayfly-go
-- ----------------------------
DROP TABLE IF EXISTS t;
-- Table structure: t
CREATE TABLE t (a varchar(20));
INSERT INTO t (a) VALUES ('it''s;a');
`
	// 语句文本未被改写：注释、换行、值内双写引号均原样保留
	want := []string{
		"-- ----------------------------\n-- Dump Platform: mayfly-go\n-- ----------------------------\nDROP TABLE IF EXISTS t",
		"-- Table structure: t\nCREATE TABLE t (a varchar(20))",
		"INSERT INTO t (a) VALUES ('it''s;a')",
	}
	assert.Equal(t, want, splitAll(t, mysqlSplitter(), dumpHeader))
	assert.Equal(t, want, splitAll(t, stdSplitter(), dumpHeader))

	// 纯注释、空语句不产生回调
	assert.Empty(t, splitAll(t, stdSplitter(), "-- only comment\n;\n/* block */\n"))

	// 非法UTF8字节不得导致后续语句丢失
	stmts := splitAll(t, stdSplitter(), "INSERT INTO t (a) VALUES ('x\xfft');\nSELECT 1;")
	assert.Len(t, stmts, 2, "非法字节不应吞掉后续语句")
	assert.Equal(t, "SELECT 1", stmts[1])
}

// 各方言装配的切割器必须与方言字符串语义匹配：
// sqlite/postgres 反斜杠为普通字符，mysql 为转义符且识别反引号
func TestSplitterPerDialectSemantics(t *testing.T) {
	// pg形态：值以反斜杠结尾
	pgScript := `INSERT INTO t (a) VALUES ('a\'); INSERT INTO t (a) VALUES (2);`
	assert.Len(t, splitAll(t, stdSplitter(), pgScript), 2, "标准SQL下两条语句")

	// mysql形态：值以反斜杠结尾需双写
	myScript := `INSERT INTO t (a) VALUES ('a\\'); INSERT INTO t (a) VALUES (2);`
	assert.Len(t, splitAll(t, mysqlSplitter(), myScript), 2, "mysql下两条语句")

	// 反引号仅mysql识别
	backtick := "SELECT 1 FROM `a;b`;"
	assert.Len(t, splitAll(t, mysqlSplitter(), backtick), 1)
	assert.Len(t, splitAll(t, stdSplitter(), backtick), 2)
}
