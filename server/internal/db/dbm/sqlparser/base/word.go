package base

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// 单词值匹配：只用于「结构上不存在歧义」的位置（子句头部、或已识别的子句尾部）。
//
// 为什么不能一律靠关键字表：START / WAIT / SKIP / NOWAIT / OF / TOP 在 Oracle 系里并非保留字
// （`CREATE TABLE t(start NUMBER)` 合法）。把它们登记成关键字，会让同名的裸标识符在
// 表引用、别名判定等「只认 TokenIdentifier」的位置解析失败，属于为修一处而坏一处。
// 而这些词所在的位置本身已无歧义：子句头部只允许出现子句关键字，FOR UPDATE 之后只允许锁修饰词。
//
// 反向约束：WHERE / FROM / LIMIT / ON 等真保留字的**边界**判定仍必须走 IsKeyword，
// 否则非保留字方言的裸列名会被误当边界截断（如 oracle 的 `WHERE left = 1`）。

// IsUsableName 判断当前位置的 token 能否作为名称（表名/限定段/别名/列名）。
//
// 为什么不能只认 TokenIdentifier：某个词是不是保留字**因方言而异且随版本变化**，
// 把它登记成关键字（好让子句边界识别它）就会同时让它在该方言的名称位上解析失败
// —— 表名解析不出、From 变空，脱敏血缘与回滚采集因此拿不到表（是静默失效，不是报错）。
// 这里改用结构判据：调用方只在「该位置只能是名称」的地方使用它，且真边界词一律先被拒，
// 于是 `FROM quick`、`INSERT INTO t (delayed)` 这类非保留字用法能建出结构，
// 而 WHERE/ORDER/LIMIT/UNION/JOIN/FROM/SELECT/WITH(表提示头) 等永远不会被当成名字吞掉。
func (l *Lexer) IsUsableName() bool {
	return l.isUsableNameAt(0)
}

// isUsableNameAt 判断相对当前位置偏移 offset 的 token 能否作为名称
func (l *Lexer) isUsableNameAt(offset int) bool {
	// 层次子句头部必须最先判定：START/CONNECT 在这些方言里是裸标识符（非保留字），
	// 一旦按类型先行放行，`FROM t START WITH ...` 的 start 就会被吃成表别名，整段层次子句随之失联
	if l.isHierarchicHeadAt(offset) {
		return false
	}
	tok := l.Peek(offset)
	if tok.Type == tokenizer.TokenIdentifier || tok.Type == tokenizer.TokenString {
		return true
	}
	if tok.Type != tokenizer.TokenKeyword {
		return false
	}
	// 关键字形态还需排除边界词：子句头、连接词、语句/子查询收尾
	if tok.IsKeyword("SELECT", "AS", "JOIN", "WITH", "BY", "SET", "UNION") {
		return false
	}
	for _, boundary := range [][]string{fromClauseEndKeywords, exprEndKeywords} {
		if tok.IsKeyword(boundary...) {
			return false
		}
	}
	return !l.isJoinStartAt(offset)
}

// IsWord 判断当前 token 值是否为给定单词之一（大小写不敏感；关键字与裸标识符都计入）
func (l *Lexer) IsWord(words ...string) bool {
	tok := l.Current()
	if tok.Type != tokenizer.TokenKeyword && tok.Type != tokenizer.TokenIdentifier {
		return false
	}
	return containsWord(words, tok.Value)
}

// isWordPair 判断当前 token 与后随 token 是否依次为给定两个单词
func (l *Lexer) isWordPair(head, tail string) bool {
	return l.isWordPairAt(0, head, tail)
}

// isWordPairAt 判断偏移 offset 起是否依次为给定两个单词
func (l *Lexer) isWordPairAt(offset int, head, tail string) bool {
	return l.peekIsWordAt(offset, head) && l.peekIsWordAt(offset+1, tail)
}

// PeekIsWord 判断相对当前位置偏移 offset 的 token 值是否为给定单词之一。
// 用于「子句头已确认、修饰词在后随位」的形态（如 INSERT 之后一位是否为 IGNORE）
func (l *Lexer) PeekIsWord(offset int, words ...string) bool {
	return l.peekIsWordAt(offset, words...)
}

// peekIsWordAt 是 PeekIsWord 的内部实现（词对匹配按偏移复用它）
func (l *Lexer) peekIsWordAt(offset int, words ...string) bool {
	tok := l.Peek(offset)
	if tok.Type != tokenizer.TokenKeyword && tok.Type != tokenizer.TokenIdentifier {
		return false
	}
	return containsWord(words, tok.Value)
}

// wordAt 判断相对当前位置偏移 offset 的 token 值是否为给定单词
func wordAt(l *Lexer, offset int, word string) bool {
	return l.PeekIsWord(offset, word)
}

// 子句边界的关键词集合（与 IsExprEnd/IsFromClauseEnd 同源，供名称位排除使用）：
// 边界词一律不可作表名/别名，否则 `FROM a INTERSECT SELECT ...` 的 INTERSECT 会被吃成别名，
// 整段集合操作随之丢失；`WHERE x = 1 EXCEPT SELECT ...` 的边界词被吞进条件，
// 会让 WHERE 文本连集合操作一起回放（回放出去就是另一条语义的 SQL）
var (
	// 集合操作词（UNION/INTERSECT/EXCEPT 各方言通用，MINUS 只在 Oracle 系登记为关键字）
	setOpKeywords = []string{"UNION", "INTERSECT", "EXCEPT", "MINUS"}

	exprEndKeywords = append([]string{
		"FROM", "WHERE", "GROUP", "HAVING", "ORDER", "LIMIT", "OFFSET",
		"INTO", "SET", "VALUES", "ON", "USING", "FOR", "RETURNING", "FETCH",
	}, setOpKeywords...)

	fromClauseEndKeywords = append([]string{
		"WHERE", "GROUP", "HAVING", "ORDER", "LIMIT", "OFFSET",
		"INTO", "FOR", "RETURNING", "FETCH", "USING", "FROM",
	}, setOpKeywords...)
)

// containsWord 判断 value 是否等于 words 中的任一项（大小写不敏感）
func containsWord(words []string, value string) bool {
	for _, w := range words {
		if strings.EqualFold(w, value) {
			return true
		}
	}
	return false
}

// isNumberToken 当前 token 是否为数字字面量
func isNumberToken(t tokenizer.Token) bool {
	return t.Type == tokenizer.TokenNumber
}
