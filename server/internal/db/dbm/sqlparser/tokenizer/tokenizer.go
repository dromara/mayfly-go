package tokenizer

import (
	"strings"
)

// Tokenizer 将 SQL 字符串拆分为 Token 序列
type Tokenizer struct {
	sql     string
	pos     int
	length  int
	config  DialectConfig
	Tokens  []Token
	current int
}

// New 创建一个新的 Tokenizer
func New(sql string, config DialectConfig) *Tokenizer {
	t := &Tokenizer{
		sql:    sql,
		pos:    0,
		length: len(sql),
		config: config,
		Tokens: make([]Token, 0),
	}
	t.tokenize()
	// 追加 EOF token
	t.Tokens = append(t.Tokens, Token{Type: TokenEOF, Value: "", Pos: t.length, End: t.length})
	return t
}

// tokenize 执行词法分析
func (t *Tokenizer) tokenize() {
	for t.pos < t.length {
		ch := t.sql[t.pos]

		// 跳过空白字符
		if isWhitespace(ch) {
			t.pos++
			continue
		}

		// 行注释（-- 与 #）：双横线是否要求后随空白由方言能力表决定（mysql 中 1--2 是减法而非注释）
		if t.config.IsLineCommentStart(t.sql, t.pos) {
			t.skipLineComment()
			continue
		}

		// 块注释 /* */：PG 系支持嵌套
		if t.config.IsBlockCommentStart(t.sql, t.pos) {
			t.skipBlockComment()
			continue
		}

		// 字符串字面量与引用标识符（'..' ".." `..` [..] $tag$.. q'[..]'）
		if t.readQuotedRegion(ch) {
			continue
		}

		// 数字
		if isDigit(ch) {
			t.readNumber()
			continue
		}

		// 标识符或关键字（字母、_、@ 开头）
		if isIdentifierStart(ch) {
			t.readIdentifierOrKeyword()
			continue
		}

		// 运算符和标点符号
		if isOperatorStart(ch) {
			t.readOperator()
			continue
		}

		if isPunctuation(ch) {
			t.Tokens = append(t.Tokens, Token{
				Type:  TokenPunctuation,
				Value: string(ch),
				Pos:   t.pos,
				End:   t.pos + 1,
			})
			t.pos++
			continue
		}

		// 未知字符，跳过
		t.pos++
	}
}

// skipLineComment 跳过一个行注释（到行尾或 EOF）
func (t *Tokenizer) skipLineComment() {
	t.pos = t.config.SkipLineComment(t.sql, t.pos)
}

// skipBlockComment 跳过一个块注释（未闭合时消费至结尾，保持词法不中断）
func (t *Tokenizer) skipBlockComment() {
	end, _ := t.config.SkipBlockComment(t.sql, t.pos)
	t.pos = end
}

// readQuotedRegion 尝试读取一个字符串字面量或引用标识符区域。
// 开闭符、双写转义、反斜杠转义、E'...' 前缀、$tag$、Oracle q'[..]' 等语义全部取自 DialectConfig，
// 与语句切割器共用同一套规则，避免两处实现漂移（如旧版无条件把 \ 当转义符，PG 的 'a\' 会吞掉后续 token）
func (t *Tokenizer) readQuotedRegion(ch byte) bool {
	switch ch {
	case '\'', '"':
		// 双引号是字符串还是标识符（mysql 系 / 标准 SQL 系）由 SkipQuoted 返回的区域类型区分
	case '`':
		if !t.config.BacktickAsIdentifier {
			return false
		}
	case '[':
		if !t.config.BracketQuote {
			return false
		}
	case 'q', 'Q':
		end, _, _, ok := t.config.SkipAltQuote(t.sql, t.pos)
		if !ok {
			return false // 非 q'[..]' 形态（如表名 q、列名 seq），交回普通词法处理
		}
		t.emit(TokenString, end)
		return true
	case '$':
		return t.readDollarQuote()
	default:
		return false
	}
	end, kind, _ := t.config.SkipQuoted(t.sql, t.pos)
	tokType := TokenString
	if kind == RegionIdentifier {
		tokType = TokenIdentifier
	}
	t.emit(tokType, end)
	return true
}

// emit 追加 [t.pos, end) 对应的 token 并推进位置
func (t *Tokenizer) emit(tokType TokenType, end int) {
	t.Tokens = append(t.Tokens, Token{
		Type:  tokType,
		Value: t.sql[t.pos:end],
		Pos:   t.pos,
		End:   end,
	})
	t.pos = end
}

// readDollarQuote 读取 PostgreSQL $tag$ ... $tag$ 风格的引号内容；未命中或无结束标记时回退为普通字符
func (t *Tokenizer) readDollarQuote() bool {
	if !t.config.DollarQuote {
		return false
	}
	tag, _ := ReadDollarTag(t.sql, t.pos)
	if tag == "" {
		return false
	}
	idx := strings.Index(t.sql[t.pos+len(tag):], tag)
	if idx < 0 {
		return false
	}
	t.emit(TokenString, t.pos+len(tag)+idx+len(tag))
	return true
}

// readNumber 读取一个数字（整数或浮点数）
func (t *Tokenizer) readNumber() {
	start := t.pos
	for t.pos < t.length && (isDigit(t.sql[t.pos]) || t.sql[t.pos] == '.') {
		t.pos++
	}
	// 支持科学计数法 e.g. 1e10, 1.5E-3
	if t.pos < t.length && (t.sql[t.pos] == 'e' || t.sql[t.pos] == 'E') {
		t.pos++
		if t.pos < t.length && (t.sql[t.pos] == '+' || t.sql[t.pos] == '-') {
			t.pos++
		}
		for t.pos < t.length && isDigit(t.sql[t.pos]) {
			t.pos++
		}
	}
	t.Tokens = append(t.Tokens, Token{
		Type:  TokenNumber,
		Value: t.sql[start:t.pos],
		Pos:   start,
		End:   t.pos,
	})
}

// readIdentifierOrKeyword 读取标识符或关键字
func (t *Tokenizer) readIdentifierOrKeyword() {
	start := t.pos
	for t.pos < t.length && isIdentifierPart(t.sql[t.pos]) {
		t.pos++
	}
	value := t.sql[start:t.pos]
	upper := strings.ToUpper(value)

	tokType := TokenIdentifier
	if Keywords[upper] {
		tokType = TokenKeyword
	} else if t.config.ExtraKeywords[upper] {
		tokType = TokenKeyword
	}

	t.Tokens = append(t.Tokens, Token{
		Type:  tokType,
		Value: value,
		Pos:   start,
		End:   t.pos,
	})
}

// readOperator 读取运算符
func (t *Tokenizer) readOperator() {
	start := t.pos
	// 尝试读取多字符运算符
	if t.pos+1 < t.length {
		two := t.sql[t.pos : t.pos+2]
		if two == "<=" || two == ">=" || two == "<>" || two == "!=" ||
			two == "||" || two == "::" || two == "->" || two == "->>" ||
			two == "=>" || two == ".." {
			// PostgreSQL :: 类型转换, -> JSON, ->> JSON text, => key-value, .. 范围
			t.pos += 2
			// 检查 ->>（三字符）
			if two == "->" && t.pos < t.length && t.sql[t.pos] == '>' {
				t.pos++
			}
			t.Tokens = append(t.Tokens, Token{
				Type:  TokenOperator,
				Value: t.sql[start:t.pos],
				Pos:   start,
				End:   t.pos,
			})
			return
		}
	}
	t.pos++
	t.Tokens = append(t.Tokens, Token{
		Type:  TokenOperator,
		Value: t.sql[start:t.pos],
		Pos:   start,
		End:   t.pos,
	})
}

// Peek 预览当前 token（不移动位置）
func (t *Tokenizer) Peek() Token {
	if t.current >= len(t.Tokens) {
		return t.Tokens[len(t.Tokens)-1] // EOF
	}
	return t.Tokens[t.current]
}

// Next 返回当前 token 并移动到下一个
func (t *Tokenizer) Next() Token {
	if t.current >= len(t.Tokens) {
		return t.Tokens[len(t.Tokens)-1] // EOF
	}
	tok := t.Tokens[t.current]
	t.current++
	return tok
}

// Consume 消耗当前位置的 token（等同于 Next，为了可读性）
func (t *Tokenizer) Consume() Token {
	return t.Next()
}

// Pos 返回当前 token 索引
func (t *Tokenizer) Pos() int {
	return t.current
}

// SetPos 设置当前 token 索引
func (t *Tokenizer) SetPos(p int) {
	t.current = p
}

// Length 返回 token 总数
func (t *Tokenizer) Length() int {
	return len(t.Tokens)
}

// TokenAt 获取指定索引的 token
func (t *Tokenizer) TokenAt(idx int) Token {
	if idx < 0 || idx >= len(t.Tokens) {
		return t.Tokens[len(t.Tokens)-1]
	}
	return t.Tokens[idx]
}

// helper functions
func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isIdentifierStart(ch byte) bool {
	return isLetter(ch) || ch == '_' || ch == '@'
}

func isIdentifierPart(ch byte) bool {
	return isLetter(ch) || isDigit(ch) || ch == '_' || ch == '@' || ch == '$'
}

func isOperatorStart(ch byte) bool {
	return ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '=' ||
		ch == '<' || ch == '>' || ch == '!' || ch == '|' || ch == ':' || ch == '~'
}

func isPunctuation(ch byte) bool {
	return ch == '(' || ch == ')' || ch == ',' || ch == ';' || ch == '.'
}
