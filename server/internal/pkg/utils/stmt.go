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

// SplitStmts 语句切割（用于以;结尾为一条语句，并且去除// -- /**/等注释）主要由阿里通义灵码提供
func SplitStmts(r io.Reader, callback StmtCallback) error {
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
					// 当前字符是转义后的字符，直接写入。如后一个为" 避免进入r==stringDelimiter判断被当做字符串结束符中断
					currentStatement.WriteRune(r)
					escapeNextChar = false
				} else if r == '\\' {
					// 当前字符是转义符，设置标志位并写入
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
