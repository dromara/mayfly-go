package tokenizer

import (
	"strings"
	"unicode/utf8"
)

// MaskSqlComments 按方言语义把 SQL 文本中的普通注释区域替换为等长空白（换行/回车/制表符原样保留），
// 字面量与引用标识符保留原文；可执行注释（mysql 的 /*!40101 ... */）解去外壳与版本门控号，仅保留其内容。
//
// 切割器保留注释原文后（见 StatementScanner），任何「按语句文本判断语句类型」的逻辑都可能命中注释内容：
// 如 dump 产物的分段注释头与其后的 BEGIN 同属一条语句（"-- Data: t\nBEGIN"），直接取前缀/整体匹配会误判。
// 可执行注释的内容会被服务端执行，故参与判定（否则 /*!40101 SET autocommit=0 */ 会绕过事务控制语句过滤）。
//
// 与前端补全用的 maskSqlComments 保持同样的等长约束（输出与输入等长，字节偏移与行列语义不变），
// 区别仅在于本函数会把可执行注释解壳参与判定，前端掩码则整段保留（补全需要其内容仍可见）。
// 未闭合的块注释掩码到文本末尾（其后内容无法判定归属，宁可当作注释）
func MaskSqlComments(sql string, cfg DialectConfig) string {
	// 快速路径：不存在任何注释起始字符时原样返回（绝大多数单条业务语句走此分支）
	if !strings.ContainsAny(sql, "-/#") {
		return sql
	}

	var sb strings.Builder
	sb.Grow(len(sql))
	for i := 0; i < len(sql); {
		r, size := utf8.DecodeRuneInString(sql[i:])
		switch {
		case cfg.IsLineCommentStart(sql, i):
			end := cfg.SkipLineComment(sql, i)
			writeMaskBlank(&sb, sql[i:end])
			i = end
		case cfg.IsBlockCommentStart(sql, i):
			end, closed := cfg.SkipBlockComment(sql, i)
			if !closed {
				end = len(sql)
			}
			body := sql[i:end]
			if !cfg.IsExecutableComment(sql, i) {
				writeMaskBlank(&sb, body)
				i = end
				continue
			}
			// 可执行注释：掩码 "/*!" 与其后的版本门控号，保留注释体，掩码尾部闭合符
			prefixLen := 3
			for prefixLen < len(body) && body[prefixLen] >= '0' && body[prefixLen] <= '9' {
				prefixLen++
			}
			content := strings.TrimSuffix(body, "*/")
			if prefixLen > len(content) {
				prefixLen = len(content)
			}
			writeMaskBlank(&sb, body[:prefixLen])
			sb.WriteString(content[prefixLen:])
			writeMaskBlank(&sb, body[len(content):])
			i = end
		case cfg.AltQuoteLiteral && IsWordStart(i, sql):
			// Oracle q'[..]' 整体是字面量，其中的 -- 与 /* 属于数据
			if end, _, _, ok := cfg.SkipAltQuote(sql, i); ok {
				sb.WriteString(sql[i:end])
				i = end
				continue
			}
			sb.WriteString(sql[i : i+size])
			i += size
		case cfg.IsQuoteStart(r) || (r == '$' && cfg.DollarQuote):
			// 字符串/引用标识符/dollar-quote 区域整体保留原文
			end, _, _ := cfg.SkipQuoted(sql, i)
			sb.WriteString(sql[i:end])
			i = end
		default:
			sb.WriteString(sql[i : i+size])
			i += size
		}
	}
	return sb.String()
}

// writeMaskBlank 以等长空白替换注释内容：非空白控制符换为空格，换行/回车/制表符保留，
// 使掩码结果的长度与行号语义与原文一致
func writeMaskBlank(sb *strings.Builder, comment string) {
	for i := 0; i < len(comment); i++ {
		switch c := comment[i]; c {
		case '\n', '\r', '\t':
			sb.WriteByte(c)
		default:
			sb.WriteByte(' ')
		}
	}
}
