package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 复杂字符串/标识符场景的语句切割测试：dump 产物包含真实换行、引号内分号、注释样式文本、
// SQL标准单引号双写与 mysql 反斜杠转义、反引号/双引号标识符，切割结果必须与之严格一致，
// 否则迁移导入时会错切语句导致数据丢失或语法错误
func TestSplitterComplexValues(t *testing.T) {
	kases := []struct {
		name  string
		sql   string
		both  []string // mysql与标准SQL语义下的共同期望
		mysql []string // 仅mysql语义期望（nil表示用both）
		std   []string // 仅标准SQL语义期望（nil表示用both）
	}{
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
			// 标准SQL无反引号语法，反引号视为普通字符，分号仍作为切割符（现状文档化）
			std: []string{"INSERT INTO `my", "t` (a) VALUES (1)"},
		},
		{
			name:  "反引号标识符含单引号",
			sql:   "INSERT INTO `it's` (a) VALUES (1);",
			mysql: []string{"INSERT INTO `it's` (a) VALUES (1)"},
			// 标准SQL不识别反引号，单引号开串后直到EOF未闭合，尾部 ; 作为串内字符保留
			std: []string{"INSERT INTO `it's` (a) VALUES (1);"},
		},
		{
			name:  "反引号标识符内含分号的多语句",
			sql:   "SELECT * FROM `a;b`; SELECT * FROM `c;d`;",
			mysql: []string{"SELECT * FROM `a;b`", "SELECT * FROM `c;d`"},
			// 标准SQL无反引号语义，各分号均视为语句结束符（现状文档化）
			std: []string{"SELECT * FROM `a", "b`", "SELECT * FROM `c", "d`"},
		},
	}

	for _, k := range kases {
		t.Run(k.name, func(t *testing.T) {
			mysqlWant, stdWant := k.both, k.both
			if k.mysql != nil {
				mysqlWant = k.mysql
			}
			if k.std != nil {
				stdWant = k.std
			}
			require.NotNil(t, mysqlWant, "用例必须声明期望（both/mysql/std 至少其一），否则断言会静默通过")
			require.NotNil(t, stdWant, "用例必须声明期望（both/mysql/std 至少其一），否则断言会静默通过")
			assert.Equal(t, mysqlWant, splitAll(t, NewMysqlSplitter(), k.sql), "mysql语义切割")
			assert.Equal(t, stdWant, splitAll(t, NewStdSQLSplitter(), k.sql), "标准SQL语义切割")
		})
	}
}

// mysql 反斜杠转义语义下的复杂值：\' 不结束字符串，\\ 为字面反斜杠
func TestSplitterMysqlBackslashEscapeValues(t *testing.T) {
	kases := []struct {
		name  string
		sql   string
		mysql []string
		std   []string
	}{
		{
			name:  "值含转义单引号与分号",
			sql:   `INSERT INTO t (a) VALUES ('a\'b;c');`,
			mysql: []string{`INSERT INTO t (a) VALUES ('a\'b;c')`},
			// 标准SQL中反斜杠为普通字符，\' 即结束字符串，其后的 ; 触发切割；
			// 余下 c'); 中 ' 重新开串后直到EOF未闭合，故尾部 ; 保留在语句内
			std: []string{`INSERT INTO t (a) VALUES ('a\'b`, `c');`},
		},
		{
			name:  "值以双反斜杠结尾",
			sql:   `INSERT INTO t (a) VALUES ('a\\'); INSERT INTO t (a) VALUES (2);`,
			mysql: []string{`INSERT INTO t (a) VALUES ('a\\')`, `INSERT INTO t (a) VALUES (2)`},
			std:   []string{`INSERT INTO t (a) VALUES ('a\\')`, `INSERT INTO t (a) VALUES (2)`},
		},
		{
			name: "pg形态值以单反斜杠结尾",
			sql:  `INSERT INTO t (a) VALUES ('a\'); INSERT INTO t (a) VALUES (2);`,
			// mysql语义下 \' 不结束字符串，直到EOF再无开引号可闭合，整段视为未闭合单条语句（含尾部分号）
			mysql: []string{`INSERT INTO t (a) VALUES ('a\'); INSERT INTO t (a) VALUES (2);`},
			std:   []string{`INSERT INTO t (a) VALUES ('a\')`, `INSERT INTO t (a) VALUES (2)`},
		},
		{
			name:  "JSON含转义斜杠路径",
			sql:   `INSERT INTO t (a) VALUES ('{"p":"C:\\tmp\\a;b"}');`,
			mysql: []string{`INSERT INTO t (a) VALUES ('{"p":"C:\\tmp\\a;b"}')`},
			std:   []string{`INSERT INTO t (a) VALUES ('{"p":"C:\\tmp\\a;b"}')`},
		},
	}

	for _, k := range kases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.mysql, splitAll(t, NewMysqlSplitter(), k.sql), "mysql语义切割")
			assert.Equal(t, k.std, splitAll(t, NewStdSQLSplitter(), k.sql), "标准SQL语义切割")
		})
	}
}

// 噪声与容错：注释剥离、空语句、未闭合字符串不得 panic 或吞掉后续可用语句
func TestSplitterNoiseAndTolerance(t *testing.T) {
	// dump 头部注释与表分隔注释全部剥离，仅保留真实语句
	dumpHeader := `
-- ----------------------------
-- Dump Platform: mayfly-go
-- Dump Time: 2026-09-06 10:00:00
-- ----------------------------
DROP TABLE IF EXISTS t;
-- ----------------------------
-- Table structure: t
-- ----------------------------
CREATE TABLE t (a varchar(20));
-- ----------------------------
-- Data: t
-- ----------------------------
INSERT INTO t (a) VALUES ('it''s;a');
`
	want := []string{
		"DROP TABLE IF EXISTS t",
		"CREATE TABLE t (a varchar(20))",
		"INSERT INTO t (a) VALUES ('it''s;a')",
	}
	assert.Equal(t, want, splitAll(t, NewMysqlSplitter(), dumpHeader))
	assert.Equal(t, want, splitAll(t, NewStdSQLSplitter(), dumpHeader))

	// 纯注释、空语句不产生回调
	assert.Empty(t, splitAll(t, NewStdSQLSplitter(), "-- only comment\n;\n/* block */\n"))

	// 未闭合字符串：不panic，剩余内容作为一条语句返回（交由执行方报错，切割不吞数据）
	assert.Equal(t, []string{"INSERT INTO t (a) VALUES ('abc"}, splitAll(t, NewStdSQLSplitter(), "INSERT INTO t (a) VALUES ('abc"))

	// 未闭合反引号（mysql）：同样不panic且不丢内容
	assert.Equal(t, []string{"SELECT * FROM `abc"}, splitAll(t, NewMysqlSplitter(), "SELECT * FROM `abc"))

	// 块注释未闭合：其后语句被吞入注释，属切割器既有语义（不做乐观恢复）
	assert.Empty(t, splitAll(t, NewStdSQLSplitter(), "/* unclosed\nINSERT INTO t VALUES (1);"))

	// 非法UTF8字节不得导致后续语句丢失
	stmts := splitAll(t, NewStdSQLSplitter(), "INSERT INTO t (a) VALUES ('x\xfft');\nSELECT 1;")
	assert.Len(t, stmts, 2, "非法字节不应吞掉后续语句")
	assert.Equal(t, "SELECT 1", stmts[1])
}

// 各方言装配的切割器必须与方言字符串语义匹配：
// sqlite/postgres 反斜杠为普通字符，mysql 为转义符且识别反引号
func TestSplitterPerDialectSemantics(t *testing.T) {
	// pg形态：值以反斜杠结尾
	pgScript := `INSERT INTO t (a) VALUES ('a\'); INSERT INTO t (a) VALUES (2);`
	pgSplitter := NewStdSQLSplitter()
	assert.Len(t, splitAll(t, pgSplitter, pgScript), 2, "标准SQL下两条语句")

	// mysql形态：值以反斜杠结尾需双写
	myScript := `INSERT INTO t (a) VALUES ('a\\'); INSERT INTO t (a) VALUES (2);`
	assert.Len(t, splitAll(t, NewMysqlSplitter(), myScript), 2, "mysql下两条语句")

	// 反引号仅mysql识别
	backtick := "SELECT 1 FROM `a;b`;"
	assert.Len(t, splitAll(t, NewMysqlSplitter(), backtick), 1)
	assert.Len(t, splitAll(t, NewStdSQLSplitter(), backtick), 2)
}
