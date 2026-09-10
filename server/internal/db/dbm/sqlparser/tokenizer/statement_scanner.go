package tokenizer

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// txnBeginWords BEGIN 后紧跟这些词（或直接跟分隔符）时为开启事务语句，无对应 END，不入复合块
var txnBeginWords = map[string]bool{
	"transaction": true, "tran": true, "work": true, "distributed": true,
	"deferred": true, "immediate": true, "exclusive": true, "read": true, "commented": true,
}

// endCloserWords 可与 END 构成一个闭合词的关键字（END IF / END LOOP / END TRY 等），
// 随 END 一并消费，避免其被当作块起始词再次入栈
var endCloserWords = map[string]bool{
	"if": true, "loop": true, "while": true, "case": true, "repeat": true, "for": true,
	"try": true, "catch": true, "block": true, "function": true, "procedure": true, "proc": true,
	"package": true, "pkg": true, "trigger": true, "transaction": true, "tran": true,
	"handler": true, "with": true,
}

// proceduralOpenWords BlockProcedural 档位额外纳入的过程块起始词；
// 不含 repeat/for（MySQL REPEAT() 同名函数；FOR 必携 LOOP，由 LOOP 计入）
var proceduralOpenWords = map[string]bool{"if": true, "loop": true, "while": true}

// spDefinitionWords 存储程序定义特征词：loop/while/if 在 MySQL、Oracle 系中均为非保留字
// （可作表名/列名），故过程块起始词仅在已处于复合块内、或当前语句确为存储程序/触发器/事件定义时才入块
var spDefinitionWords = map[string]bool{"procedure": true, "function": true, "trigger": true, "event": true}

// Kind 字段使用的未闭合区域标签：符号形式语言中立，说明文本由调用方的 i18n 模板提供
const (
	KindString       = "''"    // 单引号字符串（标准 SQL 与 mysql 同形）
	KindIdentQuote   = "\"\""  // 双引号/反引号引用区域（标准 SQL 为标识符，mysql 系为字符串）
	KindBracketQuote = "[]"    // mssql/sqlite 的 [标识符]
	KindBlockComment = "/* */" // 块注释
	KindDollarQuote  = "$tag$" // PG dollar-quoted 字符串
	KindAltQuote     = "q'..'" // Oracle 系替代引用字面量
)

// maxPendingBlockBytes 未闭合复合块的文本累积上限。超过则认为块起始词属于误判
// （如表名为 begin/case 的普通查询），关闭块感知重新切割，避免超大脚本撑爆内存
const maxPendingBlockBytes = 4 << 20

// utf8BOM UTF-8 字节顺序标记（EF BB BF）。导出工具与 Excel 常给脚本文件加上此前缀，
// 它不属于 SQL 文本且 Go 的 unicode.IsSpace 不视其为空白，不剥离会附着在首条语句上导致服务端语法错误
const utf8BOM = "\uFEFF"

// UnterminatedError 未闭合的字面量/注释区域错误。
// 切割器所在层无请求上下文，故只携带定位数据，由调用方转换为带上下文的提示
type UnterminatedError struct {
	Line int    // 区域起始所在行（1 起）
	Kind string // 区域起始符号，如 ' /* */ $$ q'[
}

func (e *UnterminatedError) Error() string {
	return fmt.Sprintf("line %d: unterminated %s", e.Line, e.Kind)
}

// StatementScanner 方言感知的语句扫描器：按分隔符切割语句，保留注释与字面量原文，
// 并感知 BEGIN..END / CASE..END 等复合块（块内分隔符不切割）。
//
// 设计对齐国际同类实现：
//   - 词法区域（字符串/引用标识符/注释）由 DialectConfig 决定，与词法器共用同一套原语，
//     避免切割与解析两处规则漂移（sql-formatter 的 DialectOptions + quotePatterns 同构思路）；
//   - 注释原文保留（DataGrip/DBeaver 行为），否则 Oracle /*+ hint */ 与 MySQL /*!40101 ... */
//     可执行注释会被静默删除，且注释紧邻的 token 会粘连成非法 SQL；
//   - 未闭合字面量/注释返回明确错误而非静默吞并后续脚本（psql 会进入续行等待，
//     批处理场景无交互，故必须报错）。
//
// 流式用法：多次 Feed（按行/按块均可）+ 一次 Finish；内部只缓存当前语句，
// 未闭合复合块除外（受 maxPendingBlockBytes 保护）
type StatementScanner struct {
	cfg        DialectConfig
	delimiter  rune
	buf        string   // 当前语句累积文本（其起点即上一条语句的结束位置）
	pos        int      // buf 中已扫描到的偏移
	blockStack []string // 复合块栈（元素为入块词，"declare" 表示匿名块声明段）
	sawSpDef   bool     // 当前语句前缀是否出现存储程序定义特征词
	hasCode    bool     // 当前语句是否已出现非注释、非空白的代码字符
	lineOffset int      // buf 之前已消费掉的整行数，用于错误行号定位
	bomChecked bool     // 流起始的 UTF-8 BOM 是否已判定/剥离
}

// NewStatementScanner 创建语句扫描器，delimiter 为语句分隔符（通常为 ';'）
func NewStatementScanner(cfg DialectConfig, delimiter rune) *StatementScanner {
	return &StatementScanner{cfg: cfg, delimiter: delimiter}
}

// Feed 喂入一段文本，返回其中已完整切割出的语句（已去除首尾空白）
func (s *StatementScanner) Feed(chunk string) ([]string, error) {
	if chunk == "" {
		return nil, nil
	}
	s.buf += chunk
	if !s.trimBom() {
		// 流起始文本尚不足以判定是否为 BOM（被输入块切断），等待更多输入
		return nil, nil
	}
	return s.scanRange(false, s.cfg.BlockMode != BlockNone)
}

// trimBom 剥离流起始的 UTF-8 BOM；返回 false 表示当前文本还无法判定，需等待更多输入后再扫
func (s *StatementScanner) trimBom() bool {
	if s.bomChecked {
		return true
	}
	if strings.HasPrefix(s.buf, utf8BOM) {
		s.buf = s.buf[len(utf8BOM):]
		s.bomChecked = true
		return true
	}
	// 当前文本是 BOM 的前缀（BOM 被输入块切断），不扫描也不丢数据
	if len(s.buf) < len(utf8BOM) && strings.HasPrefix(utf8BOM, s.buf) {
		return false
	}
	s.bomChecked = true
	return true
}

// Finish 输入结束：返回最后一条语句（无分隔符结尾的尾部文本）；
// 存在未闭合的字面量/注释时返回 *UnterminatedError
func (s *StatementScanner) Finish() ([]string, error) {
	// 输入已结束仍未能凑满 BOM 长度，剩下的就是普通文本
	s.bomChecked = true
	stmts, err := s.scanRange(true, s.cfg.BlockMode != BlockNone)
	if err != nil {
		return nil, err
	}
	// 仍存在未闭合的块起始词，说明 begin/case 等实为普通标识符或块体写坏了（如 CASE 表达式缺 END、
	// BEGIN 开事务后不再闭合），关闭块感知并从头重切当前语句，避免整段脚本被并为一条执行。
	// Go 侧不设「后续存在 END」的向前探测（前端有），未闭合的误判统一由本兜底路径纠正，两端结果一致
	if len(s.blockStack) > 0 {
		s.blockStack, s.pos, s.hasCode, s.sawSpDef = nil, 0, false, false
		rest, restartErr := s.scanRange(true, false)
		if restartErr != nil {
			return nil, restartErr
		}
		stmts = append(stmts, rest...)
	}
	if rest := strings.TrimSpace(s.buf); s.hasCode && rest != "" {
		// 已扫描但不含分隔符的尾部文本即为最后一条语句（buf 仅保留当前待切割语句）
		stmts = append(stmts, rest)
	}
	s.dropThrough(len(s.buf))
	return stmts, nil
}

// scanRange 核心扫描循环。allowBlock 为 false 时不感知复合块（用于块起始词误判的兜底重切）。
// 遇到尚未闭合的区域：atEOF 时报错，否则停留在该区域起始处等待后续输入
func (s *StatementScanner) scanRange(atEOF, allowBlock bool) ([]string, error) {
	var stmts []string
	for s.pos < len(s.buf) {
		if allowBlock && len(s.blockStack) > 0 && len(s.buf) > maxPendingBlockBytes {
			// 累积超限：块起始词判定为误判，关闭块感知从头重切当前语句
			s.pos, s.blockStack = 0, nil
			rest, err := s.scanRange(atEOF, false)
			return append(stmts, rest...), err
		}
		pos := s.pos
		r, size := utf8.DecodeRuneInString(s.buf[pos:])

		// 输入块边界可能落在 '--' / '/*' 等双字符记号中间，无法判定时等待后续数据
		if !atEOF && s.isAmbiguousTail(pos, r) {
			s.pos = pos
			return stmts, nil
		}

		// 注释与字面量区域整体跳过（原文保留在 buf 中，不参与切割）
		if s.cfg.IsLineCommentStart(s.buf, pos) {
			s.pos = s.cfg.SkipLineComment(s.buf, pos)
			continue
		}
		if s.cfg.IsBlockCommentStart(s.buf, pos) {
			end, closed := s.cfg.SkipBlockComment(s.buf, pos)
			if !closed {
				if atEOF {
					return stmts, s.unterminated(pos, KindBlockComment)
				}
				s.pos = pos
				return stmts, nil // 区域跨块输入尚未闭合，等待后续数据后重扫
			}
			s.pos = end
			if s.cfg.IsExecutableComment(s.buf, pos) {
				s.hasCode = true // mysql 的 /*! ... */ 内容将被执行，其单独成段时仍为一条语句
			}
			continue
		}
		if s.isQuoteStart(r) {
			end, _, closed := s.cfg.SkipQuoted(s.buf, pos)
			if !closed {
				if atEOF {
					return stmts, s.unterminated(pos, quoteKindLabel(r))
				}
				s.pos = pos
				return stmts, nil
			}
			s.hasCode = true
			s.pos = end
			continue
		}
		if r == '$' && s.cfg.DollarQuote {
			tag, incomplete := ReadDollarTag(s.buf, pos)
			if incomplete {
				s.pos = pos
				return stmts, nil
			}
			if tag == "" {
				s.pos = pos + size
				continue
			}
			idx := strings.Index(s.buf[pos+len(tag):], tag)
			if idx < 0 {
				if atEOF {
					return stmts, s.unterminated(pos, KindDollarQuote)
				}
				s.pos = pos
				return stmts, nil
			}
			s.hasCode = true
			s.pos = pos + len(tag) + idx + len(tag)
			continue
		}

		if r == s.delimiter {
			if allowBlock && len(s.blockStack) > 0 {
				// 复合块内的分隔符属于块体（存储过程的语句结尾），不切割
				s.pos = pos + size
				continue
			}
			// 无代码字符（仅空白与注释）的分段不构成语句，如 dump 开头的注释块与多余分隔符
			if stmt := strings.TrimSpace(s.buf[:pos]); s.hasCode && stmt != "" {
				stmts = append(stmts, stmt)
			}
			s.dropThrough(pos + size)
			continue
		}

		if !isScannerSpace(r) {
			s.hasCode = true // 出现实质代码（含字面量），本段文本才有资格构成语句
		}

		if (allowBlock || s.cfg.AltQuoteLiteral) && IsWordStart(pos, s.buf) {
			word, wordEnd := readWordLower(s.buf, pos)
			wait, err := s.consumeWord(word, pos, wordEnd, allowBlock, atEOF)
			if err != nil {
				return stmts, err
			}
			if wait {
				return stmts, nil
			}
			continue
		}

		s.pos = pos + size
	}
	return stmts, nil
}

// isAmbiguousTail 位于输入末尾的双字符记号前缀（'-' 可能是 '--' 开端，'/' 可能是 '/*' 开端），
// 需更多输入才能判定；按行喂入时仅出现在末行，按任意块喂入时保护切割位置不偏移
func (s *StatementScanner) isAmbiguousTail(pos int, r rune) bool {
	switch r {
	case '-':
		if pos+1 >= len(s.buf) {
			return true
		}
		return s.cfg.LineCommentNeedsWhitespace && s.buf[pos+1] == '-' && pos+2 >= len(s.buf)
	case '/':
		return pos+1 >= len(s.buf)
	}
	return false
}

// quoteKindLabel 引用起始符对应的未闭合区域标签
func quoteKindLabel(r rune) string {
	switch r {
	case '\'':
		return KindString
	case '"', '`':
		return KindIdentQuote
	case '[':
		return KindBracketQuote
	}
	return string(r)
}

// isQuoteStart 下标处（已解码 rune）是否为方言引用符/字符串起始符（委派能力表，与 MaskSqlComments 同源）
func (s *StatementScanner) isQuoteStart(r rune) bool {
	return s.cfg.IsQuoteStart(r)
}

// consumeWord 处理一个整词：复合块入栈/出栈、Oracle q-quote 区域跳过。
// 返回 wait=true 表示因输入不完整需等待后续数据（位置停留在该词起始处，重扫该词）
func (s *StatementScanner) consumeWord(word string, pos, wordEnd int, allowBlock, atEOF bool) (wait bool, err error) {
	// 词尾贴着输入末尾：词本身可能继续（如 BEG → BEGIN），等待后续数据后重扫该词
	if !atEOF && wordEnd >= len(s.buf) {
		s.pos = pos
		return true, nil
	}
	// q'[..]'：定界符内的分号与单引号均为字面量内容，故其识别不受块感知开关限制
	if s.cfg.AltQuoteLiteral && altQuoteWords[word] {
		end, _, closed, ok := s.cfg.SkipAltQuote(s.buf, pos)
		if ok {
			if !closed {
				if atEOF {
					return false, s.unterminated(pos, KindAltQuote)
				}
				s.pos = pos
				return true, nil
			}
			s.pos = end
			return false, nil
		}
	}
	if spDefinitionWords[word] {
		s.sawSpDef = true
	}
	if !allowBlock {
		s.pos = wordEnd
		return false, nil
	}

	switch word {
	case "begin":
		after := skipScannerSpace(s.buf, wordEnd)
		if after >= len(s.buf) && !atEOF {
			s.pos = pos
			return true, nil // 尚看不到 BEGIN 后的下一个词，等待输入再判定
		}
		nextWord, _ := readWordLower(s.buf, after)
		// BEGIN;/BEGIN TRANSACTION 等开启事务语句无对应 END，不能入块（否则后续语句被整段吞并）
		if after < len(s.buf) && !isDelimiterAt(s.buf, after, s.delimiter) && !txnBeginWords[nextWord] && !s.topIs("declare") {
			s.pushBlock("begin")
		}
	case "declare":
		// PL-SQL 匿名块：DECLARE 变量段内的分号属声明结尾，仅语句起始且无块上下文时入块
		// （T-SQL 的 DECLARE 是独立语句，BlockSql 档位不处理）
		if len(s.blockStack) == 0 && strings.TrimSpace(s.buf[:pos]) == "" {
			s.pushBlock("declare")
		}
	case "end":
		if len(s.blockStack) > 0 {
			after := skipScannerSpace(s.buf, wordEnd)
			if after >= len(s.buf) && !atEOF {
				s.pos = pos
				return true, nil // 需确认 END 后的配对词（END IF 等），等待输入
			}
			s.popBlock()
			if IsWordStart(after, s.buf) {
				if closer, closerEnd := readWordLower(s.buf, after); endCloserWords[closer] {
					wordEnd = closerEnd
				}
			}
		}
	case "case":
		// 表达式 CASE..END 与语句 CASE..END CASE 均以一个 END 闭合，计数自然平衡
		s.pushBlock("case")
	default:
		if s.cfg.BlockMode != BlockProcedural || !proceduralOpenWords[word] ||
			(len(s.blockStack) == 0 && !s.sawSpDef) {
			break
		}
		if word == "if" {
			// IF 同时是函数名与 DDL 关键字（IF NOT EXISTS），以到下一个分隔符前是否存在 THEN 为准
			if !atEOF && !strings.ContainsRune(s.buf[wordEnd:], s.delimiter) {
				s.pos = pos
				return true, nil // 分隔符与 THEN 的先后尚未可知，等待后续输入
			}
			if !hasThenBefore(s.buf, wordEnd, s.delimiter) {
				break
			}
		}
		s.pushBlock(word)
	}
	s.pos = wordEnd
	return false, nil
}

func (s *StatementScanner) pushBlock(word string) {
	s.blockStack = append(s.blockStack, word)
}

func (s *StatementScanner) popBlock() {
	s.blockStack = s.blockStack[:len(s.blockStack)-1]
}

func (s *StatementScanner) topIs(word string) bool {
	return len(s.blockStack) > 0 && s.blockStack[len(s.blockStack)-1] == word
}

// dropThrough 丢弃 buf[:end]（该段已作为完整语句输出），其后的文本前移并重置扫描位置
func (s *StatementScanner) dropThrough(end int) {
	s.lineOffset += strings.Count(s.buf[:end], "\n")
	s.buf = s.buf[end:]
	s.pos = 0
	s.blockStack = nil
	s.sawSpDef = false
	s.hasCode = false
}

// unterminated 构造带行号定位的未闭合区域错误
func (s *StatementScanner) unterminated(pos int, kind string) error {
	return &UnterminatedError{Line: s.lineOffset + strings.Count(s.buf[:pos], "\n") + 1, Kind: kind}
}

// isDelimiterAt 下标 i 处的字符是否为语句分隔符
func isDelimiterAt(sql string, i int, delimiter rune) bool {
	r, _ := utf8.DecodeRuneInString(sql[i:])
	return r == delimiter
}

// hasThenBefore word 之后、下一个分隔符之前是否存在 THEN（MySQL 的 IF 既是函数名也是 DDL 关键字）
func hasThenBefore(sql string, from int, delimiter rune) bool {
	idx := strings.IndexRune(sql[from:], delimiter)
	to := len(sql)
	if idx >= 0 {
		to = from + idx
	}
	for i := from; i < to; {
		if !IsWordStart(i, sql) {
			i++
			continue
		}
		word, end := readWordLower(sql, i)
		if word == "then" {
			return true
		}
		i = end
	}
	return false
}

// SplitStatements 使用指定方言能力表将完整 SQL 文本切割为语句列表：
// 保留注释原文，感知复合块，未闭合的字面量/注释返回 *UnterminatedError
func SplitStatements(sql string, cfg DialectConfig, delimiter rune) ([]string, error) {
	scanner := NewStatementScanner(cfg, delimiter)
	stmts, err := scanner.Feed(sql)
	if err != nil {
		return nil, err
	}
	rest, err := scanner.Finish()
	if err != nil {
		return nil, err
	}
	return append(stmts, rest...), nil
}
