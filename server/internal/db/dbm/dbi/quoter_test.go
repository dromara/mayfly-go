package dbi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuoteTo(t *testing.T) {
	var (
		quoter = Quoter{'[', ']', AlwaysReserve}
		kases  = []struct {
			expected string
			value    string
		}{
			{"[table]", "table"},
			{"[table]", "[table]"},
			{`[table].*`, `[table].*`},
			{"[schema].[table]", "schema.table"},
			{`["schema].[table"]`, `"schema.table"`},
			{"[schema].[table] AS [table]", "schema.table AS table"},
			{" [table]", " table"},
			{"  [table]", "  table"},
			{"[table] ", "table "},
			{"[table]  ", "table  "},
			{" [table] ", " table "},
			{"  [table]  ", "  table  "},
		}
	)

	for _, v := range kases {
		t.Run(v.value, func(t *testing.T) {
			buf := &strings.Builder{}
			err := quoter.QuoteTo(buf, v.value)
			assert.NoError(t, err)
			assert.EqualValues(t, v.expected, buf.String())
		})
	}
}

func TestReversedQuoteTo(t *testing.T) {
	var (
		quoter = Quoter{'[', ']', func(s string) bool {
			return s == "table"
		}}
		kases = []struct {
			expected string
			value    string
		}{
			{"[table]", "table"},
			{"[table].*", `[table].*`},
			{`"table"`, `"table"`},
			{"schema.[table]", "schema.table"},
			{"[schema].[table]", `[schema].table`},
			{"schema.[table]", `schema.[table]`},
			{"[schema].[table]", `[schema].[table]`},
			{`"schema.table"`, `"schema.table"`},
			{"schema.[table] AS table1", "schema.table AS table1"},
		}
	)

	for _, v := range kases {
		t.Run(v.value, func(t *testing.T) {
			buf := &strings.Builder{}
			quoter.QuoteTo(buf, v.value)
			assert.EqualValues(t, v.expected, buf.String())
		})
	}
}

func TestJoin(t *testing.T) {
	cols := []string{"f1", "f2", "f3"}
	quoter := Quoter{'[', ']', AlwaysReserve}

	assert.EqualValues(t, "[a],[b]", quoter.Join([]string{"a", " b"}, ","))

	assert.EqualValues(t, "[a].*,[b].[c]", quoter.Join([]string{"a.*", " b.c"}, ","))

	assert.EqualValues(t, "[b] [a]", quoter.Join([]string{"b a"}, ","))

	assert.EqualValues(t, "[f1], [f2], [f3]", quoter.Join(cols, ", "))

	quoter.IsReserved = AlwaysNoReserve
	assert.EqualValues(t, "f1, f2, f3", quoter.Join(cols, ", "))
}

func TestQuotes(t *testing.T) {
	cols := []string{"f1", "f2", "t3.f3", "t4.*"}
	quoter := Quoter{'[', ']', AlwaysReserve}

	quotedCols := quoter.Quotes(cols)
	assert.EqualValues(t, []string{"[f1]", "[f2]", "[t3].[f3]", "[t4].*"}, quotedCols)
}

func TestTrim(t *testing.T) {
	kases := map[string]string{
		"[table_name]":          "table_name",
		"[schema].[table_name]": "schema.table_name",
	}

	for src, dst := range kases {
		assert.EqualValues(t, src, DefaultQuoter.Trim(src))
		assert.EqualValues(t, dst, Quoter{'[', ']', AlwaysReserve}.Trim(src))
	}
}

// TestTrim_PartialQuotePreserved 仅首或仅尾存在引用符时属于标识符本身，不得被剔除：
// 否则按名查询会匹配到另一张表（错表取列）或查不到
func TestTrim_PartialQuotePreserved(t *testing.T) {
	mysqlQuoter := Quoter{'`', '`', AlwaysReserve}
	for _, s := range []string{"a`", "`a", "a b", ""} {
		assert.Equal(t, s, mysqlQuoter.Trim(s))
	}
}

// TestQuoteIdent 元数据标识符引用：不切分（真实表名可含空格/点/分号），内部引用符双写
func TestQuoteIdent(t *testing.T) {
	mysqlQuoter := Quoter{'`', '`', AlwaysReserve}
	std := DefaultQuoter

	kases := []struct {
		name   string
		quoter Quoter
		value  string
		expect string
	}{
		{"mysql普通名", mysqlQuoter, "tbl", "`tbl`"},
		{"mysql含空格", mysqlQuoter, "my tbl", "`my tbl`"},
		{"mysql含点", mysqlQuoter, "a.b", "`a.b`"},
		{"mysql含分号与注释符", mysqlQuoter, "a;b--c", "`a;b--c`"},
		{"mysql含反引号需双写", mysqlQuoter, "a`b", "`a``b`"},
		{"mysql已完整引用那么原样", mysqlQuoter, "`a b`", "`a b`"},
		{"std含空格", std, "my tbl", `"my tbl"`},
		{"std含双引号需双写", std, `a"b`, `"a""b"`},
		{"mssql含右括号需双写", Quoter{'[', ']', AlwaysReserve}, "a]b", "[a]]b]"},
		{"空值", std, "", ""},
		{"未配置引用符", Quoter{0, 0, AlwaysReserve}, "a b", "a b"},
	}

	for _, k := range kases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.expect, k.quoter.QuoteIdent(k.value))
		})
	}
}

// TestQuoteIdent_NotSplitLikeQuote 对比Quote：含空格的真实表名被Quote切分后逐段引用会生成非法SQL，
// QuoteIdent那么不切分（两者适用场景不同，此处锁定差异）
func TestQuoteIdent_NotSplitLikeQuote(t *testing.T) {
	mysqlQuoter := Quoter{'`', '`', AlwaysReserve}
	assert.Equal(t, "`my` `表`", mysqlQuoter.Quote("my 表"))
	assert.Equal(t, "`my 表`", mysqlQuoter.QuoteIdent("my 表"))
}

// TestQuoteWordTo_EscapeInnerQuote Quote对片段中的词也必须双写词内引用符，避免提前闭合
func TestQuoteWordTo_EscapeInnerQuote(t *testing.T) {
	mysqlQuoter := Quoter{'`', '`', AlwaysReserve}
	assert.Equal(t, "`a``b`", mysqlQuoter.Quote("a`b"))
	assert.Equal(t, `"a""b"`, DefaultQuoter.Quote(`a"b`))
}
