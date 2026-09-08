package utils

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode/utf8"
)

// StmtCallback stmt回调函数
type StmtCallback func(stmt string) error

// SplitOpts 语句切割选项
type SplitOpts struct {
	// BackslashEscape 是否将反斜杠视为字符串内的转义符（mysql语义，如 \' 不结束字符串）。
	// 标准SQL（sqlite/postgres/mssql等）中反斜杠为普通字符，必须设为false，
	// 否则形如 '\' 的完整字符串会被误判为未闭合导致后续语句被吞入字符串而错切
	BackslashEscape bool

	// HashComment 是否将 # 视为行注释起始符（mysql专属语法）
	HashComment bool

	// BacktickQuote 是否将反引号视为标识符引用符（mysql/clickhouse语义，如 `my;tbl`）。
	// 未开启时反引号内的分号会被误判为语句结束符，导致含特殊字符的表名/列名被错切；
	// 标准SQL（postgres/sqlite/oracle等）无反引号语法，必须保持false
	BacktickQuote bool
}

// SplitStmts 语句切割（用于以指定delimiter结尾为一条语句，并且去除// -- /**/等注释）主要由阿里通义灵码提供
// 默认采用mysql语义（反斜杠为转义符），标准SQL方言请使用 SplitStmtsWithOpts 指定 BackslashEscape=false
func SplitStmts(r io.Reader, delimiter rune, callback StmtCallback) error {
	return SplitStmtsWithOpts(r, delimiter, SplitOpts{BackslashEscape: true}, callback)
}

// SplitStmtsWithOpts 语句切割，支持按方言语义定制转义与注释行为
func SplitStmtsWithOpts(r io.Reader, delimiter rune, opts SplitOpts, callback StmtCallback) error {
	reader := bufio.NewReaderSize(r, 512*1024)
	buffer := new(bytes.Buffer) // 使用 bytes.Buffer 来处理数据
	var currentStatement bytes.Buffer
	var inString bool
	var inMultiLineComment bool
	var inSingleLineComment bool
	var stringDelimiter rune
	var escapeNextChar bool // 用于处理转义符

	for {
		// 读取数据到缓冲区
		data, err := reader.ReadBytes('\n') // 按行读取
		if err == io.EOF && len(data) == 0 {
			break
		}
		if err != nil && err != io.EOF {
			return err
		}
		buffer.Write(data)

		// 处理缓冲区中的数据
		for buffer.Len() > 0 {
			r, size := utf8.DecodeRune(buffer.Bytes())
			if r == utf8.RuneError && size == 1 {
				// ReadBytes('\n')已读出完整行，而UTF8多字节序列不含换行符，
				// 故此处必为真实的非法字节而非被截断的多字节序列；
				// 若不消费该字节，buffer将永久滞留导致后续语句全部丢失，故原样容错处理
				if !inMultiLineComment && !inSingleLineComment {
					currentStatement.WriteByte(buffer.Bytes()[0])
				}
				buffer.Next(1)
				continue
			}

			switch {
			case inMultiLineComment:
				if r == '*' && buffer.Len() >= 2 && buffer.Bytes()[1] == '/' {
					inMultiLineComment = false
					buffer.Next(2) // 跳过 '*/'
				} else {
					buffer.Next(size)
				}
			case inSingleLineComment:
				if r == '\n' {
					inSingleLineComment = false
				}
				buffer.Next(size)
			case inString:
				if escapeNextChar {
					// 当前字符是转义后的字符，直接写入。如后一个为" 避免进入r==stringDelimiter判断被当做字符串结束符中断
					currentStatement.WriteRune(r)
					escapeNextChar = false
				} else if opts.BackslashEscape && r == '\\' {
					// 当前字符是转义符，设置标志位并写入（仅mysql语义）
					escapeNextChar = true
					currentStatement.WriteRune(r)
				} else if r == stringDelimiter {
					// 当前字符是字符串结束符，结束字符串处理
					inString = false
					currentStatement.WriteRune(r)
				} else {
					// 其他字符，直接写入
					currentStatement.WriteRune(r)
				}
				buffer.Next(size)
			case r == '/' && buffer.Len() >= 2 && buffer.Bytes()[1] == '*':
				inMultiLineComment = true
				buffer.Next(2) // 跳过 '/*'
			case r == '-' && buffer.Len() >= 2 && buffer.Bytes()[1] == '-':
				inSingleLineComment = true
				buffer.Next(2) // 跳过 '--'
			case opts.HashComment && r == '#':
				// mysql的 # 行注释
				inSingleLineComment = true
				buffer.Next(size)
			case r == '\'' || r == '"':
				inString = true
				stringDelimiter = r
				currentStatement.WriteRune(r)
				buffer.Next(size)
			case opts.BacktickQuote && r == '`':
				// mysql反引号标识符：内部字符（含分号）不得参与语句切割，`` 为转义的反引号
				inString = true
				stringDelimiter = r
				currentStatement.WriteRune(r)
				buffer.Next(size)
			case r == delimiter && !inString && !inMultiLineComment && !inSingleLineComment:
				sql := strings.TrimSpace(currentStatement.String())
				if sql != "" {
					if err := callback(sql); err != nil {
						return err
					}
				}
				currentStatement.Reset()
				buffer.Next(size)
			default:
				currentStatement.WriteRune(r)
				buffer.Next(size)
			}
		}

		// 如果读取到 EOF 并且缓冲区为空，退出循环
		if err == io.EOF && buffer.Len() == 0 {
			break
		}
	}

	// 处理最后剩余的缓冲区
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
