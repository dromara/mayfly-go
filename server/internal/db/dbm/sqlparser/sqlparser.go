package sqlparser

import (
	"bufio"
	"bytes"
	"io"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"strings"
	"unicode/utf8"
)

type DbDialect string

type SqlParser interface {

	// sql解析
	// @param stmt sql语句
	Parse(stmt string) ([]sqlstmt.Stmt, error)
}

// SQLSplit sql切割
func SQLSplit(r io.Reader, callback SQLCallback) error {
	return parseSQL(r, callback)
}

// SQLCallback 是解析出一条 SQL 语句后的回调函数
type SQLCallback func(sql string) error

// parseSQL 主要由阿里通义灵码提供
func parseSQL(r io.Reader, callback SQLCallback) error {
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
				// 如果解码出错，说明数据不完整，继续读取更多数据
				break
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
					currentStatement.WriteRune(r)
					escapeNextChar = false
				} else if r == '\\' {
					escapeNextChar = true
					currentStatement.WriteRune(r)
				} else if r == stringDelimiter {
					inString = false
					currentStatement.WriteRune(r)
				} else {
					currentStatement.WriteRune(r)
				}
				buffer.Next(size)
			case r == '/' && buffer.Len() >= 2 && buffer.Bytes()[1] == '*':
				inMultiLineComment = true
				buffer.Next(2) // 跳过 '/*'
			case r == '-' && buffer.Len() >= 2 && buffer.Bytes()[1] == '-':
				inSingleLineComment = true
				buffer.Next(2) // 跳过 '--'
			case r == '\'' || r == '"':
				inString = true
				stringDelimiter = r
				currentStatement.WriteRune(r)
				buffer.Next(size)
			case r == ';' && !inString && !inMultiLineComment && !inSingleLineComment:
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
