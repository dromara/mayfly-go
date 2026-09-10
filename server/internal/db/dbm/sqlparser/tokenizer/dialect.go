package tokenizer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// BlockMode 复合语句块（BEGIN..END）感知档位，供语句切割器使用
type BlockMode int

const (
	// BlockNone 不感知复合块，块内分号照常切割（sqlite/clickhouse 等无过程语句的方言）
	BlockNone BlockMode = iota
	// BlockSql 仅感知 BEGIN..END 与 CASE..END（T-SQL 的 IF/WHILE 无 END 闭合；PG 过程体多在引号内）
	BlockSql
	// BlockProcedural 额外感知 IF..END IF / LOOP..END LOOP / WHILE..END WHILE（mysql 存储程序、Oracle 系 PL-SQL）
	BlockProcedural
)

// DialectConfig SQL 方言语义能力表（切割器与词法器共用的单一事实来源）。
//
// 此前切割器（sqlparser.Splitter / PgsqlSplitter）与词法器各自维护一套引号/注释规则，
// 三方语义独立演进而产生漂移（如词法器无条件把 \ 当转义符，PG 的 'a\' 会吞掉后续 token），
// 新增能力必须同时改多处。现统一由本表声明，方言只需注册一次。
type DialectConfig struct {
	// BackslashEscape 字符串内反斜杠是否为转义符（mysql/clickhouse true）。
	// 标准 SQL 与 PG（standard_conforming_strings=on）中 \ 为普通字符，必须 false，
	// 否则形如 '\' 的完整字符串会被判为未闭合而吞掉后续语句
	BackslashEscape bool

	// BacktickAsIdentifier 反引号是否为标识符引用符（mysql/clickhouse/sqlite）
	BacktickAsIdentifier bool

	// DoubleQuoteAsIdentifier 双引号是否为标识符引用符（标准 SQL 系）；false 时其为字符串字面量（mysql 系）
	DoubleQuoteAsIdentifier bool

	// HashLineComment # 是否为行注释起始符（mysql/clickhouse true；PG 中 # 是运算符、mssql 中是临时表前缀，必须 false）
	HashLineComment bool

	// DollarQuote 是否支持 PG 的 $tag$...$tag$ 字符串（函数体/DO 块内必含分号）
	DollarQuote bool

	// BracketQuote 是否支持 [标识符]（mssql 主引用符、sqlite 兼容形态），]] 为转义的 ]
	BracketQuote bool

	// AltQuoteLiteral 是否支持 Oracle 系 q'[..]' 替代引用字面量（内部引号与分号均为字面量内容）
	AltQuoteLiteral bool

	// EscapeStringPrefix 是否支持 E'...' 转义字符串（PG 系：普通串内 \ 为普通字符，仅 E 串内生效）
	EscapeStringPrefix bool

	// NestedBlockComment 块注释是否支持嵌套（PG 系 true）
	NestedBlockComment bool

	// LineCommentNeedsWhitespace 双横线行注释是否要求后随空白/行尾
	// （mysql 系 true：SELECT 1--2 是减法运算，若按注释处理会吞掉其后的整条语句）
	LineCommentNeedsWhitespace bool

	// ExecutableComment 是否支持 mysql 的 /*! ... */（或 /*!40101 ... */ 带版本门控）可执行注释：
	// 其内容会被服务端当作 SQL 执行，故纯由它构成的分段仍是语句，不可按注释丢弃
	ExecutableComment bool

	// BlockMode 复合语句块感知档位（仅语句切割器使用）
	BlockMode BlockMode

	// ExtraKeywords 方言额外关键字集合
	ExtraKeywords map[string]bool
}

// 各方言能力表：与前端 utils/sqlParser.ts 的 dialectSplitOptions 逐项对应，
// 新增方言时在此登记一次即可，切割器与词法器自动获得正确语义
var (
	// StdConfig 标准 SQL（未注册方言的回落语义）：反斜杠为普通字符，双引号为标识符
	StdConfig = DialectConfig{DoubleQuoteAsIdentifier: true}

	// MysqlConfig mysql/mariadb/tidb 语义
	MysqlConfig = DialectConfig{
		BackslashEscape:            true,
		BacktickAsIdentifier:       true,
		HashLineComment:            true,
		LineCommentNeedsWhitespace: true,
		ExecutableComment:          true,
		BlockMode:                  BlockProcedural,
	}

	// ClickhouseConfig clickhouse 语义（# 与 -- 均为行注释，-- 不要求后随空白）
	ClickhouseConfig = DialectConfig{
		BackslashEscape:      true,
		BacktickAsIdentifier: true,
		HashLineComment:      true,
	}

	// PgConfig postgres 系（含 gauss/人大金仓/vastbase/redshift 等兼容方言）
	PgConfig = DialectConfig{
		DoubleQuoteAsIdentifier: true,
		DollarQuote:             true,
		EscapeStringPrefix:      true,
		NestedBlockComment:      true,
		BlockMode:               BlockSql,
	}

	// SqliteConfig sqlite 语义（兼容 MySQL 的两种引用符，但反斜杠为普通字符）
	SqliteConfig = DialectConfig{
		BacktickAsIdentifier:    true,
		BracketQuote:            true,
		DoubleQuoteAsIdentifier: true,
	}

	// MssqlConfig T-SQL 语义（方括号标识符 + BEGIN..END/BEGIN TRY；IF、WHILE 无 END 闭合故不纳入过程关键字）
	MssqlConfig = DialectConfig{
		BracketQuote:            true,
		DoubleQuoteAsIdentifier: true,
		BlockMode:               BlockSql,
	}

	// OracleConfig oracle 语义（q-quote + PL-SQL 过程块）
	OracleConfig = DialectConfig{
		DoubleQuoteAsIdentifier: true,
		AltQuoteLiteral:         true,
		BlockMode:               BlockProcedural,
	}

	// DmConfig 达梦语义（兼容 Oracle 的 q-quote 与 PL-SQL 过程块，反斜杠为普通字符）
	DmConfig = DialectConfig{
		DoubleQuoteAsIdentifier: true,
		AltQuoteLiteral:         true,
		BlockMode:               BlockProcedural,
	}
)

// RegionKind 共享扫描器识别出的区域类型
type RegionKind int

const (
	RegionNone         RegionKind = iota
	RegionString                  // '...' 或 mysql 的 "..."
	RegionIdentifier              // "a"（标准 SQL）、`a`、[a]
	RegionDollar                  // $tag$...$tag$
	RegionAltQuote                // q'[..]'
	RegionLineComment             // -- .. 或 # ..
	RegionBlockComment            // /* .. */
)

// IsExecutableComment sql[i:] 是否为 mysql 的可执行注释起始（/*! 或 /*!NNNNN）。
// 区域边界仍按块注释扫描，只是不视为可丢弃的注释文本
func (c DialectConfig) IsExecutableComment(sql string, i int) bool {
	return c.ExecutableComment && strings.HasPrefix(sql[i:], "/*!")
}

// IsLineCommentStart sql[i:] 是否为行注释起始（含 mysql 的双横线空白规则）
func (c DialectConfig) IsLineCommentStart(sql string, i int) bool {
	r, _ := utf8.DecodeRuneInString(sql[i:])
	if r == '#' && c.HashLineComment {
		return true
	}
	if r != '-' || !strings.HasPrefix(sql[i:], "--") {
		return false
	}
	// mysql 语义：-- 后必须跟空白/行尾才算注释，否则是减法运算
	return !c.LineCommentNeedsWhitespace || isSpaceOrEOF(sql, i+2)
}

// IsBlockCommentStart sql[i:] 是否为块注释起始
func (c DialectConfig) IsBlockCommentStart(sql string, i int) bool {
	return strings.HasPrefix(sql[i:], "/*")
}

// SkipLineComment 跳过行注释，返回行尾换行符的下标（注释本体不含换行符）
func (c DialectConfig) SkipLineComment(sql string, i int) int {
	idx := strings.IndexByte(sql[i:], '\n')
	if idx < 0 {
		return len(sql)
	}
	return i + idx
}

// SkipBlockComment 从 /* 起始处跳过块注释（按方言决定是否支持嵌套），返回 */ 之后的下标；未闭合时返回 len(sql) 与 false
func (c DialectConfig) SkipBlockComment(sql string, i int) (int, bool) {
	depth := 1
	for pos := i + 2; pos < len(sql); {
		if strings.HasPrefix(sql[pos:], "*/") {
			depth--
			pos += 2
			if depth == 0 {
				return pos, true
			}
			continue
		}
		if c.NestedBlockComment && strings.HasPrefix(sql[pos:], "/*") {
			depth++
			pos += 2
			continue
		}
		_, size := utf8.DecodeRuneInString(sql[pos:])
		pos += size
	}
	return len(sql), false
}

// ReadDollarTag 读取 i 处起始的 dollar-quote 标签（$$ 或 $tag$），语义对齐 PG 词法：
// tag 体仅允许字母/数字/下划线且不得跨行；incomplete 为 true 表示输入提前结束，需更多数据才能判定
func ReadDollarTag(sql string, i int) (tag string, incomplete bool) {
	if i < 0 || i >= len(sql) || sql[i] != '$' {
		return "", false
	}
	isTagByte := func(ch byte) bool {
		return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_'
	}
	for j := i + 1; j < len(sql); j++ {
		ch := sql[j]
		if ch == '$' {
			for _, t := range []byte(sql[i+1 : j]) {
				if !isTagByte(t) {
					return "", false
				}
			}
			return sql[i : j+1], false
		}
		if !isTagByte(ch) {
			return "", false
		}
	}
	return "", true
}

// IsEscapeStringQuote 判断 i 处单引号是否为 PG 的 E'...' 转义字符串：
// 仅单引号（PG 无 E"..." 语法）、E/e 须为独立词（前一个字符不能是标识符字符），
// 否则列名尾字母 e 会误开启转义语义
func (c DialectConfig) IsEscapeStringQuote(sql string, quoteIdx int) bool {
	if !c.EscapeStringPrefix || quoteIdx == 0 || sql[quoteIdx] != '\'' {
		return false
	}
	prefix := sql[quoteIdx-1]
	return (prefix == 'E' || prefix == 'e') && !IsWordByte(quoteIdx-2, sql)
}

// IsQuoteStart 下标处（已解码 rune）是否为方言引用符/字符串起始符（与 SkipQuoted 配套使用）
func (c DialectConfig) IsQuoteStart(r rune) bool {
	switch r {
	case '\'', '"':
		return true
	case '`':
		return c.BacktickAsIdentifier
	case '[':
		return c.BracketQuote
	}
	return false
}

// SkipQuoted 从 i 处的引号/方括号起始符跳过整个字符串或引用标识符区域。
// 返回区域结束后的下标、区域类型与是否闭合；开闭符可异字符（如 [x]），
// 双写转义（” / "" / “ / ]]）与反斜杠转义按方言语义各自独立判定
func (c DialectConfig) SkipQuoted(sql string, i int) (end int, kind RegionKind, closed bool) {
	open, size := utf8.DecodeRuneInString(sql[i:])
	if open == '$' {
		tag, _ := ReadDollarTag(sql, i)
		if tag == "" {
			return i + size, RegionNone, true
		}
		idx := strings.Index(sql[i+len(tag):], tag)
		if idx < 0 {
			return len(sql), RegionDollar, false
		}
		return i + len(tag) + idx + len(tag), RegionDollar, true
	}

	closeCh := open
	doubling := false
	backslash := c.BackslashEscape
	switch open {
	case '\'':
		kind = RegionString
		doubling = true
		// E'...' 串内反斜杠才是转义符，普通串按方言语义
		backslash = c.BackslashEscape || c.IsEscapeStringQuote(sql, i)
	case '"':
		doubling = true
		backslash = c.BackslashEscape
		if c.DoubleQuoteAsIdentifier {
			kind = RegionIdentifier
		} else {
			kind = RegionString
		}
	case '`':
		kind = RegionIdentifier
		doubling = true
		backslash = false // 反引号标识符内反斜杠为普通字符，转义靠双写
	case '[':
		kind = RegionIdentifier
		closeCh = ']'
		doubling = true // ]] 为转义的右括号
		backslash = false
	}

	for pos := i + size; pos < len(sql); {
		r, rsize := utf8.DecodeRuneInString(sql[pos:])
		if backslash && r == '\\' && pos+1 < len(sql) {
			pos += 2
			continue
		}
		if r == closeCh {
			if doubling && pos+rsize < len(sql) && nextRuneIs(sql[pos+rsize:], closeCh) {
				pos += rsize * 2
				continue
			}
			return pos + rsize, kind, true
		}
		pos += rsize
	}
	return len(sql), kind, false
}

// altOpenCloseMap Oracle q-quote 括号型定界符配对，其余字符作定界符时首尾同字符
var altOpenCloseMap = map[rune]rune{'[': ']', '{': '}', '(': ')', '<': '>'}

// altQuoteWords Oracle 系替代引用字面量的合法前缀词（q'[..]' / nq'[..]' / uq'[..]'）
var altQuoteWords = map[string]bool{"q": true, "nq": true, "uq": true}

// SkipAltQuote 从 q/nq/uq 前缀词起始处跳过 Oracle 系 q'[..]' 替代引用字面量，
// 返回区域结束后的下标与是否闭合；非 q-quote 形态返回 ok=false
func (c DialectConfig) SkipAltQuote(sql string, i int) (end int, kind RegionKind, closed, ok bool) {
	if !c.AltQuoteLiteral {
		return 0, RegionNone, false, false
	}
	// 前缀词必须是词首，否则列名尾字母 q（如 seq'..')）会误开启 q-quote 区域
	if i > 0 && IsWordByte(i-1, sql) {
		return 0, RegionNone, false, false
	}
	word, wordEnd := readWordLower(sql, i)
	if !altQuoteWords[word] {
		return 0, RegionNone, false, false
	}
	if wordEnd >= len(sql) || sql[wordEnd] != '\'' {
		return 0, RegionNone, false, false
	}
	openIdx := wordEnd + 1
	if openIdx >= len(sql) {
		return 0, RegionNone, false, false
	}
	open, openSize := utf8.DecodeRuneInString(sql[openIdx:])
	if open == '\n' {
		return 0, RegionNone, false, false
	}
	closeCh, ok := altOpenCloseMap[open]
	if !ok {
		closeCh = open
	}
	altClose := string(closeCh) + "'"
	idx := strings.Index(sql[openIdx+openSize:], altClose)
	if idx < 0 {
		return len(sql), RegionAltQuote, false, true
	}
	return openIdx + openSize + idx + len(altClose), RegionAltQuote, true, true
}

func nextRuneIs(sql string, want rune) bool {
	r, _ := utf8.DecodeRuneInString(sql)
	return r == want
}

// readWordLower 读取 i 处起始的整词（小写）及其结束下标（不含）
func readWordLower(sql string, i int) (word string, end int) {
	for j := i; j < len(sql); {
		if !IsWordByte(j, sql) {
			return strings.ToLower(sql[i:j]), j
		}
		_, size := utf8.DecodeRuneInString(sql[j:])
		j += size
	}
	return strings.ToLower(sql[i:]), len(sql)
}

// isScannerSpace 是否为空白字符（含换行、制表符等）
func isScannerSpace(r rune) bool {
	return unicode.IsSpace(r)
}

// skipScannerSpace 跳过空白，返回首个非空白字符下标（无则返回 len(sql)）
func skipScannerSpace(sql string, i int) int {
	for i < len(sql) {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if !unicode.IsSpace(r) {
			return i
		}
		i += size
	}
	return i
}

func isSpaceOrEOF(sql string, i int) bool {
	if i >= len(sql) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(sql[i:])
	return unicode.IsSpace(r)
}

// IsWordByte 下标 i 处是否为标识符字符（含 $ 以兼容 PG 标识符；非 ASCII 字节视为词字符以兼容中文标识符）
func IsWordByte(i int, sql string) bool {
	if i < 0 || i >= len(sql) {
		return false
	}
	ch := sql[i]
	if ch >= utf8.RuneSelf {
		return true
	}
	return ch == '_' || ch == '$' || unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch))
}

// IsWordStart 下标 i 处是否为词首字符（$ 属 dollar-quote 判定范畴，故不作词首）
func IsWordStart(i int, sql string) bool {
	if i < 0 || i >= len(sql) {
		return false
	}
	ch := sql[i]
	if ch >= utf8.RuneSelf {
		return true
	}
	return ch == '_' || unicode.IsLetter(rune(ch))
}
