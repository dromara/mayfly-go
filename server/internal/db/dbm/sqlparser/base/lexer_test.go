package base

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

func newTestLexer() *Lexer {
	return NewLexer("", tokenizer.DialectConfig{})
}

// TestExtractColumnAndAlias 列名/别名提取：AS定位必须忽略括号内与字符串字面量内的AS
func TestExtractColumnAndAlias(t *testing.T) {
	l := newTestLexer()
	cases := []struct {
		text  string
		col   string
		alias string
	}{
		{"phone", "phone", ""},
		{"phone AS p", "phone", "p"},
		{"phone as p", "phone", "p"},
		{"t.phone AS p", "phone", "p"},             // 限定名取尾段
		{"`t`.`phone` AS p", "phone", "p"},         // 反引号去引用
		{`"t"."PHONE" AS p`, "PHONE", "p"},         // 双引号去引用
		{"CAST(x AS CHAR)", "CAST(x AS CHAR)", ""}, // 括号内AS不得误判
		{"MAX(phone) AS mp", "MAX(phone)", "mp"},
		{"COUNT(*)", "COUNT(*)", ""},
		{"phone AS '别名'", "phone", "'别名'"},                             // 带引号别名原样返回（上层处理）
		{`CONCAT('a AS b', x)`, "CONCAT('a AS b', x)", ""},             // 字符串字面量内AS不得误判
		{`CONCAT('it''s', phone) AS c`, "CONCAT('it''s', phone)", "c"}, // ''转义
		{"CASE WHEN a > 1 THEN b END AS c", "CASE WHEN a > 1 THEN b END", "c"},
		{"phone p", "phone", "p"}, // 无AS隐式别名
	}
	for _, c := range cases {
		col, alias := l.ExtractColumnAndAlias(c.text)
		if col != c.col || alias != c.alias {
			t.Errorf("ExtractColumnAndAlias(%q) = (%q, %q), 期望 (%q, %q)", c.text, col, alias, c.col, c.alias)
		}
	}
}

// TestExtractColumnName 列名提取：去限定、去引用、尾段
func TestExtractColumnName(t *testing.T) {
	l := newTestLexer()
	cases := []struct {
		text string
		col  string
	}{
		{"phone", "phone"},
		{"t.phone", "phone"},
		{"db.t.phone", "phone"},
		{"`phone`", "phone"},
		{`"PHONE"`, "PHONE"},
		{"  t.`phone`  ", "phone"},
		{"", ""},
	}
	for _, c := range cases {
		if col := l.ExtractColumnName(c.text); col != c.col {
			t.Errorf("ExtractColumnName(%q) = %q, 期望 %q", c.text, col, c.col)
		}
	}
}

// TestSplitIdentifiers 空白分割需覆盖换行（多行SQL的隐式别名场景）
func TestSplitIdentifiers(t *testing.T) {
	l := newTestLexer()
	cases := []struct {
		text  string
		parts []string
	}{
		{"phone p", []string{"phone", "p"}},
		{"phone\np", []string{"phone", "p"}},
		{"phone \t p", []string{"phone", "p"}},
		{"phone", []string{"phone"}},
		{"", nil},
	}
	for _, c := range cases {
		parts := l.SplitIdentifiers(c.text)
		if len(parts) != len(c.parts) {
			t.Errorf("SplitIdentifiers(%q) = %v, 期望 %v", c.text, parts, c.parts)
			continue
		}
		for i := range parts {
			if parts[i] != c.parts[i] {
				t.Errorf("SplitIdentifiers(%q) = %v, 期望 %v", c.text, parts, c.parts)
			}
		}
	}
}

// TestFindTopLevelAS 括号深度与字符串字面量感知的AS定位
func TestFindTopLevelAS(t *testing.T) {
	cases := []struct {
		text string
		idx  int
	}{
		{"phone AS p", 6},
		{"CAST(x AS CHAR)", -1},
		{"CONCAT('a AS b', x)", -1},
		{"f(x) AS y", 5},
		{"f((x)) AS y", 7},
		{"no alias here", -1},
		{"astrix", -1},  // AS子串不算
		{"x\nAS\ny", 2}, // 换行分隔的AS
	}
	for _, c := range cases {
		if idx := findTopLevelAS(c.text); idx != c.idx {
			t.Errorf("findTopLevelAS(%q) = %d, 期望 %d", c.text, idx, c.idx)
		}
	}
}
