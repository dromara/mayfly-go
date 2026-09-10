package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// splitAll 便捷方法：将sql按delimiter切割为语句列表
func splitAll(t *testing.T, sql string, delimiter rune, opts ...SplitOpts) []string {
	t.Helper()
	stmts := make([]string, 0)
	err := SplitStmtsWithOpts(strings.NewReader(sql), delimiter, callbackOpts(opts...), func(stmt string) error {
		stmts = append(stmts, stmt)
		return nil
	})
	require.NoError(t, err)
	return stmts
}

func callbackOpts(opts ...SplitOpts) SplitOpts {
	if len(opts) > 0 {
		return opts[0]
	}
	// 与历史 SplitStmts 行为一致：反斜杠为转义符（mysql语义）
	return SplitOpts{BackslashEscape: true}
}

func TestSplitStmts_Basic(t *testing.T) {
	assert.Equal(t, []string{"select 1", "select 2"}, splitAll(t, "select 1; select 2;", ';'))
	// 尾部无分隔符
	assert.Equal(t, []string{"select 1", "select 2"}, splitAll(t, "select 1;\nselect 2", ';'))
	// 空语句与纯空白被跳过
	assert.Equal(t, []string{"a"}, splitAll(t, ";;;  a;  ;\n;", ';'))
	// 空输入
	assert.Empty(t, splitAll(t, "", ';'))
	// 自定义分隔符
	assert.Equal(t, []string{"a", "b"}, splitAll(t, "a@b@", '@'))
	// 语句内换行保留
	assert.Equal(t, []string{"select 1\nfrom t"}, splitAll(t, "select 1\nfrom t;", ';'))
}

func TestSplitStmts_StringWithDelimiter(t *testing.T) {
	// 单引号字符串内的分号不切割
	assert.Equal(t, []string{"insert into t values ('a;b')"}, splitAll(t, `insert into t values ('a;b');`, ';'))
	// 双引号内的分号不切割
	assert.Equal(t, []string{`select "a;b"`}, splitAll(t, `select "a;b";`, ';'))
	// 双写单引号转义（SQL标准 ''）
	assert.Equal(t, []string{`insert into t values ('it''s;x')`}, splitAll(t, `insert into t values ('it''s;x');`, ';'))
	// 反斜杠转义引号（mysql语义，默认行为）
	assert.Equal(t, []string{`insert into t values ('it\'s;x')`}, splitAll(t, `insert into t values ('it\'s;x');`, ';'))
	// 字符串内的双引号与注释样文本
	assert.Equal(t, []string{`insert into t values ('-- not comment /* also not */')`},
		splitAll(t, `insert into t values ('-- not comment /* also not */');`, ';'))
}

func TestSplitStmts_Comments(t *testing.T) {
	// 行注释内的分号不切割，注释内容被移除
	assert.Equal(t, []string{"select 1", "select 2"},
		splitAll(t, "select 1; -- comment; here\nselect 2;", ';'))
	// 块注释内的分号不切割（注释内容被移除，但保留两侧原有空格）
	assert.Equal(t, []string{"select  1", "select 2"},
		splitAll(t, "select /* ; */ 1; select 2;", ';'))
	// 跨行块注释
	assert.Equal(t, []string{"select  1", "select 2"},
		splitAll(t, "select /* line1\nline2 ; */ 1;\nselect 2;", ';'))
	// 纯注释脚本不产生任何语句
	assert.Empty(t, splitAll(t, "-- only comment; here\n/* block; comment */", ';'))
	// 注释紧贴 token 时不得粘连（移除后补空白）：否则语义被改写为 aFROM t
	assert.Equal(t, []string{"SELECT a FROM t"}, splitAll(t, "SELECT a-- c\nFROM t;", ';'))
	assert.Equal(t, []string{"SELECT a FROM t"}, splitAll(t, "SELECT a/* c */FROM t;", ';'))
	// 注释与语句混合的多行脚本
	script := "-- header comment\ncreate table t(id int); -- trailing\n-- another\ninsert into t values(1);\n"
	assert.Equal(t, []string{"create table t(id int)", "insert into t values(1)"}, splitAll(t, script, ';'))
}

func TestSplitStmts_ChineseAndMultibyte(t *testing.T) {
	// 中文与emoji含分号形态的字符不误切
	assert.Equal(t, []string{"insert into t values ('中文；分号🙂;semi')"},
		splitAll(t, "insert into t values ('中文；分号🙂;semi');", ';'))
	// 中文分号（全角；）不作为分隔符
	assert.Equal(t, []string{"select '中文；'"}, splitAll(t, "select '中文；';", ';'))
}

// TestSplitStmts_StdSqlBackslash 标准SQL语义（sqlite/postgres/mssql等）：反斜杠为普通字符，
// 字符串以未转义引号结束，不影响语句切割
func TestSplitStmts_StdSqlBackslash(t *testing.T) {
	opts := SplitOpts{BackslashEscape: false}
	// sqlite中 '\' 是完整的单字符字符串，其后为另一条语句，必须切开（mysql语义下会误判为未闭合字符串）
	assert.Equal(t, []string{`insert into t values ('\')`, `insert into t values ('x')`},
		splitAll(t, `insert into t values ('\'); insert into t values ('x');`, ';', opts))
	// 'a\;b' 中分号在字符串内，不切割
	assert.Equal(t, []string{`insert into t values ('a\;b')`},
		splitAll(t, `insert into t values ('a\;b');`, ';', opts))
	// 标准SQL的 '' 双写转义
	assert.Equal(t, []string{`insert into t values ('it''s;x')`},
		splitAll(t, `insert into t values ('it''s;x');`, ';', opts))
}

// TestSplitStmts_MysqlBackslash mysql语义：反斜杠为转义符（默认行为，保持历史兼容）
func TestSplitStmts_MysqlBackslash(t *testing.T) {
	// \' 不结束字符串，分号在字符串内
	assert.Equal(t, []string{`insert into t values ('it\'s;x')`},
		splitAll(t, `insert into t values ('it\'s;x');`, ';'))
	// '\\' 结束转义后字符串正常闭合
	assert.Equal(t, []string{`insert into t values ('a\\;b')`},
		splitAll(t, `insert into t values ('a\\;b');`, ';'))
}

// TestSplitStmts_MysqlHashComment mysql的 # 行注释
func TestSplitStmts_MysqlHashComment(t *testing.T) {
	opts := SplitOpts{BackslashEscape: true, HashComment: true}
	assert.Equal(t, []string{"select 1", "select 2"},
		splitAll(t, "select 1; # comment; here\nselect 2;", ';', opts))
	// 字符串内的 # 不是注释
	assert.Equal(t, []string{"insert into t values ('a#b;c')"},
		splitAll(t, "insert into t values ('a#b;c');", ';', opts))
}

// TestSplitStmts_BadUtf8 非法UTF8字节容错：坏字节原样透传，不得中断解析导致后续语句丢失
func TestSplitStmts_BadUtf8(t *testing.T) {
	// 坏字节夹在语句中间：语句原样保留（含坏字节），后续语句正常切割
	script := "insert into t values ('a\xffb'); select 2;"
	assert.Equal(t, []string{"insert into t values ('a\xffb')", "select 2"}, splitAll(t, script, ';'))

	// 坏字节后跟多字节中文字符与更多语句
	script2 := "select 1; -- \xff 注释\nselect 中文; select 3;"
	assert.Equal(t, []string{"select 1", "select 中文", "select 3"}, splitAll(t, script2, ';'))
}
