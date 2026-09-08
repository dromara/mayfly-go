package base

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Lexer 提供词法分析辅助方法，不包含方言相关逻辑
type Lexer struct {
	SQL    string
	Tokens []tokenizer.Token
	Pos    int
	Length int
	// SelectParser 子查询解析回调：由各方言Parser构造时注入（parseSelect的包装），
	// 表达式解析遇到(SELECT...)时可建SelectStmt树；nil时降级为文本节点
	SelectParser SelectParser
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

// SkipSelectItem 跳过SELECT项表达式：以顶层逗号或SELECT子句边界结束。
// 与SkipExpr的区别：JOIN前缀关键字（LEFT/RIGHT/INNER/OUTER/CROSS/NATURAL/FULL）
// 不作为边界——它们不是SQL保留字，可出现在项尾别名位置（如 AS full），
// 误判会截断项文本并使别名错乱
func (l *Lexer) SkipSelectItem() {
	for !l.Current().IsEOF() {
		tok := l.Current()
		if tok.Value == "(" {
			l.SkipParentheses()
			continue
		}
		if tok.Value == "," || l.IsSelectClauseEnd() {
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

// CaptureSkip 执行skip函数并返回其跳过的token区间（调用前Current位置 → 调用后Current位置）
// 对应的原始SQL文本（去除首尾空白）。用于GROUP BY/HAVING等“仅跳过不建模”的表达式段落
// 回填原始文本，标记其存在性；未消费任何token或无有效区间时返回空串
func (l *Lexer) CaptureSkip(skip func()) string {
	start := 0
	if !l.Current().IsEOF() {
		start = l.Current().Pos
	}
	before := l.Pos
	skip()
	if l.Pos == before && l.Current().IsEOF() {
		return ""
	}
	end := len(l.SQL)
	if !l.Current().IsEOF() {
		end = l.Current().Pos
	}
	if end <= start {
		return ""
	}
	return strings.TrimSpace(l.SQL[start:end])
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

// IsExprEnd 判断表达式是否结束（WHERE/HAVING/GROUP BY/ORDER BY等普通表达式场景）。
// 注意：不含JOIN前缀关键字（LEFT/RIGHT/FULL等）——它们不是SQL保留字，
// 在非保留字方言（oracle/dm）可作裸列名，误判会截断表达式
// （如 WHERE left = 1 AND right = 2 在left处被截断）；
// ON条件等其后可能紧跟JOIN子句的场景用 IsCondEnd
func (l *Lexer) IsExprEnd() bool {
	tok := l.Current()
	// 注意：需包含 FETCH（oracle/达梦的 FETCH FIRST n ROWS ONLY 分页子句不能被当作表达式的一部分吞掉）
	return tok.IsKeyword("FROM", "WHERE", "GROUP", "HAVING", "ORDER", "LIMIT", "OFFSET",
		"UNION", "INTO", "SET", "VALUES", "ON", "USING", "FOR", "RETURNING", "FETCH") ||
		tok.Value == ";" || tok.Value == ")"
}

// IsCondEnd 判断条件表达式是否结束：在 IsExprEnd 基础上额外将 JOIN 前缀关键字
// 作为边界——专用于 JOIN 的 ON 条件（ON a.id = b.id 后可能紧跟 LEFT JOIN）
func (l *Lexer) IsCondEnd() bool {
	if l.IsExprEnd() {
		return true
	}
	return l.IsJoinStart() || l.Current().IsKeyword("JOIN")
}

// SkipCondExpr 跳过条件表达式（JOIN的ON条件专用，见IsCondEnd）
func (l *Lexer) SkipCondExpr() {
	for !l.Current().IsEOF() {
		tok := l.Current()
		if tok.Value == "(" {
			l.SkipParentheses()
			continue
		}
		if tok.Value == "," || l.IsCondEnd() {
			break
		}
		l.Consume()
	}
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
		// 含\r\n：多行SQL的列别名分割
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
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

// findTopLevelAS 查找括号深度为0且不在单引号字符串字面量内的最后一个AS关键字位置（返回-1表示无）。
// 直接LastIndex" AS "会把 CAST(x AS CHAR) 等函数内部AS误判为列别名
func findTopLevelAS(text string) int {
	depth := 0
	inStr := false
	last := -1
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if inStr {
			if ch == '\'' {
				if i+1 < len(text) && text[i+1] == '\'' {
					i++ // ''转义
				} else {
					inStr = false
				}
			}
			continue
		}
		switch ch {
		case '\'':
			inStr = true
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && isASKeywordAt(text, i) {
				last = i
			}
		}
	}
	return last
}

// isASKeywordAt 判断text[i:]是否为独立的AS关键字（前非标识符字符，后跟空白）
func isASKeywordAt(text string, i int) bool {
	if i+2 > len(text) || !strings.EqualFold(text[i:i+2], "AS") {
		return false
	}
	if i > 0 && isIdentByte(text[i-1]) {
		return false
	}
	if i+2 >= len(text) {
		return false
	}
	next := text[i+2]
	return next == ' ' || next == '\t' || next == '\n' || next == '\r'
}

// isIdentByte 是否为标识符字符字节（字母/数字/下划线，含多字节UTF8高位字节）
func isIdentByte(b byte) bool {
	return b == '_' || b >= 0x80 || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// ClassifySelectItem 依据列项文本与别名判定 SelectItem 的 Kind 与 TableAlias。
// 各方言解析器统一调用，保证跨方言的 Kind 语义一致（调用方可直接依赖 IsStar()/Kind，无需再做文本启发式判定）：
//   - "*" / "t.*" → Star（t.* 记录去引用后的表别名）
//   - 主体含括号、位串运算符或空白 → Expr（MAX(u.phone)、"PHONE"||'-x'、CASE WHEN... 等）
//   - 其余为纯列引用 → Column；限定列（u.phone / db.t.col / "u"."PHONE"）记录 TableAlias（倒数第二段，去引用）
func (l *Lexer) ClassifySelectItem(text, alias string) (sqlstmt.SelectItemKind, string) {
	body := extractColumnBody(text, alias)
	if body == "*" {
		return sqlstmt.SelectItemStar, ""
	}
	if strings.HasSuffix(body, ".*") {
		return sqlstmt.SelectItemStar, l.Unquote(strings.TrimSuffix(body, ".*"))
	}
	if strings.ContainsAny(body, "(| \t\n") {
		return sqlstmt.SelectItemExpr, ""
	}
	parts := strings.Split(body, ".")
	if len(parts) < 2 {
		return sqlstmt.SelectItemColumn, ""
	}
	return sqlstmt.SelectItemColumn, l.Unquote(strings.TrimSpace(parts[len(parts)-2]))
}

// extractColumnBody 剔除列项文本尾部的别名（AS或隐式别名），返回列主体
func extractColumnBody(text, alias string) string {
	body := strings.TrimSpace(text)
	if alias == "" || !strings.HasSuffix(body, alias) {
		return body
	}
	body = strings.TrimSpace(body[:len(body)-len(alias)])
	if len(body) >= 2 && strings.EqualFold(body[len(body)-2:], "AS") {
		body = strings.TrimSpace(body[:len(body)-2])
	}
	return body
}

// ExtractColumnAndAlias 从列文本中提取列名和别名
func (l *Lexer) ExtractColumnAndAlias(text string) (string, string) {
	if idx := findTopLevelAS(text); idx >= 0 {
		colPart := strings.TrimSpace(text[:idx])
		aliasPart := strings.TrimSpace(text[idx+2:])
		return l.ExtractColumnName(colPart), aliasPart
	}
	parts := l.SplitIdentifiers(text)
	if len(parts) >= 2 {
		lastPart := parts[len(parts)-1]
		beforeLast := strings.TrimSpace(text[:len(text)-len(lastPart)])
		// 限定列主体（u.phone / db.t.col）的"."是限定符而非歧义，隐式别名同样成立
		if !strings.Contains(beforeLast, "(") || strings.HasSuffix(beforeLast, ")") {
			return l.ExtractColumnName(beforeLast), strings.TrimSpace(lastPart)
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
