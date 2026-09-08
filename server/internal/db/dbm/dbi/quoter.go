package dbi

import (
	"mayfly-go/pkg/utils/collx"
	"strings"
)

// Quoter represents a quoter to the SQL identifier. i.e. table name or column name
type Quoter struct {
	Prefix     byte
	Suffix     byte
	IsReserved func(string) bool
}

var (
	// AlwaysNoReserve always think it's not a reverse word
	AlwaysNoReserve = func(string) bool { return false }

	// AlwaysReserve always reverse the word
	AlwaysReserve = func(string) bool { return true }

	// DefaultQuoter represents the default quoter
	DefaultQuote byte = '"'

	// DefaultQuoter the default quoter
	DefaultQuoter = Quoter{DefaultQuote, DefaultQuote, AlwaysReserve}
)

// IsEmpty return true if no prefix and suffix
func (q Quoter) IsEmpty() bool {
	return q.Prefix == 0 && q.Suffix == 0
}

// Quote quote a string
func (q Quoter) Quote(s string) string {
	var buf strings.Builder
	_ = q.QuoteTo(&buf, s)
	return buf.String()
}

// Strings quotes a slice of string
func (q Quoter) Quotes(s []string) []string {
	return collx.ArrayMap[string, string](s, func(val string) string {
		return q.Quote(val)
	})
}

// QuoteTo quotes the identifier. if the quotes are [ and ]
//
//	name -> [name]
//	[name] -> [name]
//	schema.name -> [schema].[name]
//	[schema].name -> [schema].[name]
//	schema.[name] -> [schema].[name]
//	name AS a  ->  [name] AS a
//	schema.name AS a  ->  [schema].[name] AS a
func (q Quoter) QuoteTo(buf *strings.Builder, value string) error {
	var i int
	for i < len(value) {
		start := findStart(value, i)
		if start > i {
			if _, err := buf.WriteString(value[i:start]); err != nil {
				return err
			}
		}
		if start == len(value) {
			return nil
		}

		nextEnd := findWord(value, start)
		if err := q.quoteWordTo(buf, value[start:nextEnd]); err != nil {
			return err
		}
		i = nextEnd
	}
	return nil
}

// QuoteIdent 引用来自元数据的单个标识符（真实表名/列名）。
//
// 与 Quote 的差异是必须的：Quote 为兼容 schema.name、name AS a 等SQL片段会按 . 与空格切分后
// 逐段引用，而元数据返回的是一个完整标识符，其内部合法包含空格、点、分号等字符（如MySQL的
// `my tbl`、`a.b`，pg/sqlite的"my tbl"），按片段切分必然生成非法SQL。故此处不做任何切分，
// 并将标识符内部的引用符按SQL语义双写（反引号、双引号、右中括号各自成对），保证含引用符的名称不被破坏。
//
// 空值、未配置引用符、或已是完整引用形态（如 `a`、"a"、[a]）时原样返回，兼容调用方传入已引用名称
func (q Quoter) QuoteIdent(s string) string {
	if s == "" || q.IsEmpty() {
		return s
	}
	if len(s) >= 2 && s[0] == q.Prefix && s[len(s)-1] == q.Suffix {
		return s
	}

	var buf strings.Builder
	buf.Grow(len(s) + 2)
	buf.WriteByte(q.Prefix)
	for i := 0; i < len(s); i++ {
		if s[i] == q.Suffix {
			buf.WriteByte(q.Suffix) // 引用符双写转义
		}
		buf.WriteByte(s[i])
	}
	buf.WriteByte(q.Suffix)
	return buf.String()
}

// Trim removes quotes from s
func (q Quoter) Trim(s string) string {
	// 仅当整体为完整引用形态时才去引用：否则真实标识符（如 mysql 的 `x 或 x`）首尾恰好存在的引用符
	// 会被当作语法引用剔除，使后续按名查询匹配到另一张表（错表取列）或查不到
	if len(s) < 2 || s[0] != q.Prefix || s[len(s)-1] != q.Suffix {
		return s
	}

	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case i == 0 && s[i] == q.Prefix:
		case i == len(s)-1 && s[i] == q.Suffix:
		case s[i] == q.Suffix && s[i+1] == '.':
		case s[i] == q.Prefix && s[i-1] == '.':
		default:
			buf.WriteByte(s[i])
		}
	}
	return buf.String()
}

// Join joins a slice with quoters
func (q Quoter) Join(a []string, separator string) string {
	var b strings.Builder
	_ = q.JoinWrite(&b, a, separator)
	return b.String()
}

// JoinWrite writes quoted content to a builder
func (q Quoter) JoinWrite(b *strings.Builder, a []string, sep string) error {
	if len(a) == 0 {
		return nil
	}

	n := len(sep) * (len(a) - 1)
	for i := 0; i < len(a); i++ {
		n += len(a[i])
	}

	b.Grow(n)
	for i, s := range a {
		if i > 0 {
			if _, err := b.WriteString(sep); err != nil {
				return err
			}
		}
		if err := q.QuoteTo(b, strings.TrimSpace(s)); err != nil {
			return err
		}
	}
	return nil
}

func (q Quoter) quoteWordTo(buf *strings.Builder, word string) error {
	// 空字符串直接写入，避免 word[0] 越界 panic
	if word == "" {
		return nil
	}
	if (word[0] == q.Prefix && word[len(word)-1] == q.Suffix) ||
		q.IsEmpty() || !q.IsReserved(word) || word == "*" {
		if _, err := buf.WriteString(word); err != nil {
			return err
		}
		return nil
	}

	if err := buf.WriteByte(q.Prefix); err != nil {
		return err
	}
	// 词内出现的引用符必须双写，否则如 a`b 会被提前闭合导致后续内容脱离引用（语法错误甚至注入）
	for i := 0; i < len(word); i++ {
		if word[i] == q.Suffix {
			if err := buf.WriteByte(q.Suffix); err != nil {
				return err
			}
		}
		if err := buf.WriteByte(word[i]); err != nil {
			return err
		}
	}
	return buf.WriteByte(q.Suffix)
}

func findWord(v string, start int) int {
	for j := start; j < len(v); j++ {
		switch v[j] {
		case '.', ' ':
			return j
		}
	}
	return len(v)
}

func findStart(value string, start int) int {
	if value[start] == '.' {
		return start + 1
	}
	if value[start] != ' ' {
		return start
	}

	k := -1
	for j := start; j < len(value); j++ {
		if value[j] != ' ' {
			k = j
			break
		}
	}
	if k == -1 {
		return len(value)
	}

	if k+1 < len(value) &&
		(value[k] == 'A' || value[k] == 'a') &&
		(value[k+1] == 'S' || value[k+1] == 's') {
		k += 2
	}

	for j := k; j < len(value); j++ {
		if value[j] != ' ' {
			return j
		}
	}
	return len(value)
}
