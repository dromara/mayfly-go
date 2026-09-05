package sqlparser

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode/utf8"

	"mayfly-go/internal/pkg/utils"
)

// PgsqlSplitter PostgreSQL 专属 SQL 切割器
// 相比标准SQL切割器需额外处理三类PG专属语法，否则会导致语句错切：
//   - dollar-quoted 字符串：$$...$$ / $tag$...$tag$（函数定义、DO块体内必含分号）
//   - E'...' 转义字符串：反斜杠为转义符（如 E'it\'s'），与普通字符串语义不同
//   - 嵌套块注释：/* /* ; */ */ PG支持注释嵌套
type PgsqlSplitter struct {
	delimiter rune
}

func NewPgsqlSplitter() *PgsqlSplitter {
	return &PgsqlSplitter{delimiter: ';'}
}

var _ SQLSplitter = (*PgsqlSplitter)(nil)

// SplitSQL 实现 SQLSplitter 接口，按分号切割PG语句并移除注释
func (s *PgsqlSplitter) SplitSQL(r io.Reader, callback utils.StmtCallback) error {
	reader := bufio.NewReaderSize(r, 512*1024)
	buffer := new(bytes.Buffer)
	var currentStatement bytes.Buffer

	var inString bool       // ' 或 " 字符串/标识符
	var isEscapeString bool // 是否为 E'...' 转义字符串（反斜杠为转义符）
	var stringDelimiter rune
	var escapeNextChar bool
	var dollarTag []byte // 非空表示正处于 dollar-quoted 字符串内，值为闭合tag（如 $$ 或 $fn$）
	var blockCommentDepth int
	var inLineComment bool
	var prevChar rune // 前一个已写入语句的字符，用于识别 E'...' 的 E 前缀

	writeStmt := func(rr rune) {
		currentStatement.WriteRune(rr)
		prevChar = rr
	}

	for {
		data, err := reader.ReadBytes('\n')
		if err == io.EOF && len(data) == 0 {
			break
		}
		if err != nil && err != io.EOF {
			return err
		}
		buffer.Write(data)

		for buffer.Len() > 0 {
			ch, size := utf8.DecodeRune(buffer.Bytes())
			if ch == utf8.RuneError && size == 1 {
				// ReadBytes('\n')已读出完整行，而UTF8多字节序列不含换行符，
				// 故此处必为真实的非法字节而非被截断的多字节序列；
				// 若不消费该字节，buffer将永久滞留导致后续语句全部丢失，故原样容错处理
				if !inLineComment && blockCommentDepth == 0 {
					currentStatement.WriteByte(buffer.Bytes()[0])
				}
				buffer.Next(1)
				continue
			}

			switch {
			case inLineComment:
				if ch == '\n' {
					inLineComment = false
				}
				buffer.Next(size)

			case blockCommentDepth > 0:
				if ch == '/' && buffer.Len() >= 2 && buffer.Bytes()[1] == '*' {
					// PG支持嵌套块注释
					blockCommentDepth++
					buffer.Next(2)
				} else if ch == '*' && buffer.Len() >= 2 && buffer.Bytes()[1] == '/' {
					blockCommentDepth--
					buffer.Next(2)
				} else {
					buffer.Next(size)
				}

			case dollarTag != nil:
				writeStmt(ch)
				buffer.Next(size)
				// 当前语句尾部匹配到闭合tag（如 $fn$），结束dollar-quoted字符串
				if bytes.HasSuffix(currentStatement.Bytes(), dollarTag) {
					dollarTag = nil
				}

			case inString:
				switch {
				case isEscapeString && escapeNextChar:
					// 转义后的字符直接写入（如 E'\'' 中的引号不结束字符串）
					writeStmt(ch)
					escapeNextChar = false
					buffer.Next(size)
				case isEscapeString && ch == '\\':
					escapeNextChar = true
					writeStmt(ch)
					buffer.Next(size)
				case ch == stringDelimiter:
					if buffer.Len() >= 2 && buffer.Bytes()[1] == byte(stringDelimiter) {
						// 双写引号转义（'' / ""）：写入两个引号并额外消费第二个字符
						writeStmt(ch)
						buffer.Next(size)
						writeStmt(stringDelimiter)
						buffer.Next(1)
					} else {
						inString = false
						writeStmt(ch)
						buffer.Next(size)
					}
				default:
					writeStmt(ch)
					buffer.Next(size)
				}

			case ch == '/' && buffer.Len() >= 2 && buffer.Bytes()[1] == '*':
				blockCommentDepth = 1
				buffer.Next(2)

			case ch == '-' && buffer.Len() >= 2 && buffer.Bytes()[1] == '-':
				inLineComment = true
				buffer.Next(2)

			case ch == '\'' || ch == '"':
				inString = true
				stringDelimiter = ch
				// E/e 紧贴引号时为转义字符串（如 E'it\'s'），反斜杠语义生效
				isEscapeString = ch == '\'' && (prevChar == 'E' || prevChar == 'e')
				writeStmt(ch)
				buffer.Next(size)

			case ch == '$':
				if tag := readDollarTag(buffer); tag != nil {
					// dollar-quoted 字符串起始，tag为完整闭合序列（如 $fn$ 或 $$）；
					// 闭合由后续逐字符写入时的尾部匹配触发，开tag自身不触发闭合
					dollarTag = tag
					for _, tr := range string(tag) {
						writeStmt(tr)
					}
					buffer.Next(len(tag))
				} else {
					writeStmt(ch)
					buffer.Next(size)
				}

			case ch == s.delimiter && !inString && dollarTag == nil && blockCommentDepth == 0:
				sql := strings.TrimSpace(currentStatement.String())
				if sql != "" {
					if err := callback(sql); err != nil {
						return err
					}
				}
				currentStatement.Reset()
				prevChar = 0
				buffer.Next(size)

			default:
				writeStmt(ch)
				buffer.Next(size)
			}
		}

		if err == io.EOF && buffer.Len() == 0 {
			break
		}
	}

	if currentStatement.Len() > 0 {
		sql := strings.TrimSpace(currentStatement.String())
		if sql != "" {
			if err := callback(sql); err != nil {
				return err
			}
		}
	}
	return nil
}

// readDollarTag 从缓冲区尝试读取dollar-quote起始tag（如 $$ 或 $fn$），
// tag内容仅允许字母数字下划线且必须在当前行内闭合，读取失败返回nil（此时$按普通字符处理）
func readDollarTag(buffer *bytes.Buffer) []byte {
	// buffer.Bytes()[0]为当前'$'字符，从其后开始扫描到下一个'$'
	rest := buffer.Bytes()[1:]
	for i := 0; i < len(rest); i++ {
		ch := rest[i]
		if ch == '$' {
			// 找到闭合'$'，tag为 $...$ 完整序列
			tag := make([]byte, i+2)
			tag[0] = '$'
			copy(tag[1:], rest[:i+1])
			// 中间内容需为合法tag字符（字母数字下划线）
			for _, t := range tag[1 : len(tag)-1] {
				if !(t >= 'a' && t <= 'z' || t >= 'A' && t <= 'Z' || t >= '0' && t <= '9' || t == '_') {
					return nil
				}
			}
			return tag
		}
		// tag内不允许换行（跨行的$不是dollar-quote）
		if ch == '\n' {
			return nil
		}
	}
	return nil
}
