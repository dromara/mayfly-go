package base

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Lexer 提供词法分析辅助方法，不包含方言相关逻辑
type Lexer struct {
	SQL    string
	Tokens []tokenizer.Token
	Pos    int
	Length int
}

// NewLexer 创建基础词法分析器
func NewLexer(sql string, cfg tokenizer.DialectConfig) *Lexer {
	tok := tokenizer.New(sql, cfg)
	return &Lexer{
		SQL:    sql,
		Tokens: tok.Tokens,
		Pos:    0,
		Length: len(tok.Tokens),
	}
}

// Current 返回当前 token
func (l *Lexer) Current() tokenizer.Token {
	if l.Pos >= l.Length {
		return l.Tokens[l.Length-1] // EOF
	}
	return l.Tokens[l.Pos]
}

// Peek 预览指定偏移位置的 token
func (l *Lexer) Peek(offset int) tokenizer.Token {
	idx := l.Pos + offset
	if idx >= l.Length {
		return l.Tokens[l.Length-1]
	}
	return l.Tokens[idx]
}

// Consume 消费当前 token 并前进
func (l *Lexer) Consume() tokenizer.Token {
	if l.Pos >= l.Length {
		return l.Tokens[l.Length-1]
	}
	tok := l.Tokens[l.Pos]
	l.Pos++
	return tok
}

// ExpectValue 如果当前 token 值匹配则消费，否则返回当前 token
func (l *Lexer) ExpectValue(val string) tokenizer.Token {
	if l.Current().Value == val {
		return l.Consume()
	}
	return l.Current()
}

// SkipSemicolons 跳过分号
func (l *Lexer) SkipSemicolons() {
	for l.Current().Value == ";" {
		l.Consume()
	}
}

// SkipToNextStatement 跳过到下一个语句
func (l *Lexer) SkipToNextStatement() {
	for !l.Current().IsEOF() && l.Current().Value != ";" {
		l.Consume()
	}
	if l.Current().Value == ";" {
		l.Consume()
	}
}

// SkipParentheses 跳过一对括号及其内容
func (l *Lexer) SkipParentheses() {
	if l.Current().Value != "(" {
		return
	}
	l.Consume() // (
	depth := 1
	for !l.Current().IsEOF() && depth > 0 {
		if l.Current().Value == "(" {
			depth++
		} else if l.Current().Value == ")" {
			depth--
		}
		l.Consume()
	}
}

// SkipExpr 跳过整个表达式（用于 WHERE, HAVING 等）
func (l *Lexer) SkipExpr() {
	for !l.Current().IsEOF() {
		tok := l.Current()
		if tok.Value == "(" {
			l.SkipParentheses()
			continue
		}
		if tok.Value == "," || l.IsExprEnd() {
			break
		}
		l.Consume()
	}
}

// SkipGroupByExpr 跳过 GROUP BY 表达式（允许逗号分隔）
func (l *Lexer) SkipGroupByExpr() {
	for !l.Current().IsEOF() {
		tok := l.Current()
		if tok.Value == "(" {
			l.SkipParentheses()
			continue
		}
		if l.IsExprEnd() || tok.Value == ";" {
			break
		}
		l.Consume()
	}
}

// SkipOrderByExpr 跳过 ORDER BY 表达式
func (l *Lexer) SkipOrderByExpr() {
	for !l.Current().IsEOF() {
		if l.Current().Value == "," {
			l.Consume()
			continue
		}
		if l.Current().IsKeyword("ASC", "DESC") {
			l.Consume()
			continue
		}
		if l.IsExprEnd() || l.Current().Value == ";" {
			break
		}
		if l.Current().Value == "(" {
			l.SkipParentheses()
			continue
		}
		l.Consume()
	}
}

// IsExprEnd 判断表达式是否结束
func (l *Lexer) IsExprEnd() bool {
	tok := l.Current()
	// 注意：需包含 FETCH（oracle/达梦的 FETCH FIRST n ROWS ONLY 分页子句不能被当作表达式的一部分吞掉）
	return tok.IsKeyword("FROM", "WHERE", "GROUP", "HAVING", "ORDER", "LIMIT", "OFFSET",
		"UNION", "INTO", "SET", "VALUES", "ON", "USING", "FOR", "RETURNING", "FETCH",
		"LEFT", "RIGHT", "INNER", "OUTER", "CROSS", "NATURAL", "FULL", "JOIN") ||
		tok.Value == ";" || tok.Value == ")"
}

// IsFromClauseEnd 判断 FROM 子句是否结束
// 注意：需包含 FETCH（oracle/达梦的 FETCH FIRST n ROWS ONLY 分页子句紧跟表引用之后，属于 FROM 子句的结束边界）
func (l *Lexer) IsFromClauseEnd() bool {
	tok := l.Current()
	return tok.IsKeyword("WHERE", "GROUP", "HAVING", "ORDER", "LIMIT", "OFFSET",
		"UNION", "INTO", "FOR", "RETURNING", "FETCH") || tok.Value == ";" || tok.Value == ")"
}

// IsSelectClauseEnd 判断是否到达 SELECT 子句末尾
func (l *Lexer) IsSelectClauseEnd() bool {
	tok := l.Current()
	return tok.IsKeyword("FROM", "WHERE", "GROUP", "HAVING", "ORDER", "LIMIT", "OFFSET", "UNION", "INTO", "FOR") ||
		tok.Value == ";" || tok.Value == ")"
}

// IsJoinStart 判断是否为 JOIN 起始
func (l *Lexer) IsJoinStart() bool {
	tok := l.Current()
	return tok.IsKeyword("LEFT", "RIGHT", "INNER", "OUTER", "NATURAL", "CROSS", "FULL", "STRAIGHT_JOIN")
}

// TextFrom 返回从 start 到当前位置的原始 SQL 文本
// 注意：start为0（语句起始）时会包含首个 token 之前的前导注释与空白，保证语句原文完整
func (l *Lexer) TextFrom(start int) string {
	if start >= l.Length {
		return ""
	}
	end := l.Pos
	if end >= l.Length {
		end = l.Length - 1
	}
	if end < start {
		end = start
	}
	startTok := l.Tokens[start]
	endTok := l.Tokens[end]
	if endTok.Type == tokenizer.TokenEOF && end > 0 {
		endTok = l.Tokens[end-1]
	}
	startPos := startTok.Pos
	if start == 0 {
		startPos = 0
	}
	if endTok.End <= startPos {
		return ""
	}
	return l.SQL[startPos:endTok.End]
}

// TextFromExclusive 返回从 start 到当前位置之前（不包含当前 token）的原始 SQL 文本
func (l *Lexer) TextFromExclusive(start int) string {
	if start >= l.Length {
		return ""
	}
	end := l.Pos - 1
	if end < start {
		end = start
	}
	startTok := l.Tokens[start]
	endTok := l.Tokens[end]
	if endTok.Type == tokenizer.TokenEOF && end > 0 {
		endTok = l.Tokens[end-1]
	}
	if endTok.End <= startTok.Pos {
		return ""
	}
	return l.SQL[startTok.Pos:endTok.End]
}

// ParseInt 解析整数，忽略错误
func (l *Lexer) ParseInt(s string) int {
	n := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}

// Unquote 去除标识符引号
func (l *Lexer) Unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '`' && s[len(s)-1] == '`') ||
			(s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// SplitDotParts 按点分割标识符，考虑引号
func (l *Lexer) SplitDotParts(text string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	var quoteChar byte = 0
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if inQuote {
			current.WriteByte(ch)
			if ch == quoteChar {
				if i+1 < len(text) && text[i+1] == quoteChar {
					i++
				} else {
					inQuote = false
					quoteChar = 0
				}
			}
		} else if ch == '.' {
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
		} else if ch == '`' || ch == '"' {
			inQuote = true
			quoteChar = ch
			current.WriteByte(ch)
		} else {
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}
	return parts
}

// SplitIdentifiers 按空白分割标识符
func (l *Lexer) SplitIdentifiers(text string) []string {
	var parts []string
	var current strings.Builder
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// ExtractColumnName 从文本中提取列名（不带表前缀）
func (l *Lexer) ExtractColumnName(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	parts := l.SplitDotParts(text)
	if len(parts) >= 2 {
		return l.Unquote(parts[len(parts)-1])
	}
	return l.Unquote(text)
}

// ExtractColumnAndAlias 从列文本中提取列名和别名
func (l *Lexer) ExtractColumnAndAlias(text string) (string, string) {
	upper := strings.ToUpper(text)
	if idx := strings.LastIndex(upper, " AS "); idx >= 0 {
		colPart := strings.TrimSpace(text[:idx])
		aliasPart := strings.TrimSpace(text[idx+4:])
		return l.ExtractColumnName(colPart), aliasPart
	}
	parts := l.SplitIdentifiers(text)
	if len(parts) >= 2 {
		lastPart := parts[len(parts)-1]
		beforeLast := strings.TrimSpace(text[:len(text)-len(lastPart)])
		if !strings.Contains(beforeLast, ".") {
			if !strings.Contains(beforeLast, "(") || strings.HasSuffix(beforeLast, ")") {
				return l.ExtractColumnName(beforeLast), strings.TrimSpace(lastPart)
			}
		}
	}
	return l.ExtractColumnName(text), ""
}

// TrimTrailingComma 去除尾部逗号
func TrimTrailingComma(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, ",") {
		return strings.TrimSpace(s[:len(s)-1])
	}
	return s
}
