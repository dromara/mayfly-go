package dbi

import (
	"fmt"
	"regexp"
	"strings"
)

// QuoteEscape 转义SQL字符串字面量内容（标准SQL语义：单引号以双写表示）
func QuoteEscape(str string) string {
	assertNoNulByte(str)
	return strings.Replace(str, `'`, `''`, -1)
}

// assertNoNulByte 拦截含NUL(0x00)字节的文本进入标准SQL字面量/脚本文本流。
//
// 标准SQL字符串字面量无NUL转义语法，SQL脚本文本流无法可靠承载原始NUL字节：
// sqlite tokenizer按C字符串语义遇0x00截断，会静默产生损坏的后续语句（实测）；
// pg的text类型本身禁止NUL。若不拦截，含NUL的数据会被静默写坏且可能连带损毁
// 后续语句（备份场景下不可接受），故必须快速失败并给出明确上下文。
// 注意：mysql语义的字面量请走QuoteEscapeBackslash（其\0转义可无损承载NUL）
func assertNoNulByte(str string) {
	if strings.IndexByte(str, 0x00) >= 0 {
		panic(fmt.Sprintf("text contains NUL(0x00) byte which cannot be carried by SQL script stream; refuse to generate corrupted SQL, value prefix: %.64q", str))
	}
}

// QuoteEscapeBackslash 转义mysql字符串字面量内容：在标准SQL的单引号双写基础上，
// 额外双写反斜杠。mysql默认未开启NO_BACKSLASH_ESCAPES，反斜杠是字面量内的转义符，
// 不双写会使含反斜杠的表名/列注释/默认值等失真（如 information_schema 条件匹配不到真实表）。
// 另外mysql语义支持\0转义，可将NUL字节无损嵌入字面量（标准SQL无此转义，见assertNoNulByte）
func QuoteEscapeBackslash(str string) string {
	// 顺序敏感：必须先双写反斜杠再替换NUL，否则NUL转义序列\0中的反斜杠会被再次双写，
	// 导致还原时得到字面"\0"文本而非NUL；替换后已无NUL，可安全进入QuoteEscape
	escaped := strings.ReplaceAll(str, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, "\x00", `\0`)
	return QuoteEscape(escaped)
}

// SanitizeCommentText 清洗要嵌入SQL行注释（-- 开头）的文本（如库表名）。
//
// 行注释的作用域到行尾结束，若嵌入的标识符含换行（MySQL/pg/sqlite均允许标识符内含换行），
// 注释会在换行处提前结束，剩余名称内容会成为独立可执行语句被导入执行（如表名为
// "a\nDROP TABLE x" 即相当于在导出脚本里注入了DROP语句），故必须将换行类字符归一为空格
func SanitizeCommentText(str string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\x00', '\u2028', '\u2029':
			return ' '
		default:
			return r
		}
	}, str)
}

// UnwrapSqlLiteral 还原SQL字符串字面量的原始值：仅剥去最外层的一对单引号，并把内部双写的单引号还原。
//
// 各数据库元数据对默认值的呈现不一致：MySQL 8.0起 information_schema.COLUMNS.COLUMN_DEFAULT 返回
// 未加引号的原始值（如内容含单引号的默认值直接呈现为 it+单引号+s），5.7/MariaDB与sqlite的
// dflt_value 仍返回外层带引号、内部单引号双写的字面量。此处对两种形态做统一：
//   - 带引号形态剥一层并还原双写，得到原始值；
//   - 未加引号形态原样返回。
//
// 必须只剥最外层一对：按字符集剥除所有首尾引号会把默认值本身以引号开头/结尾的内容当成语法
// 引用剥掉，使结构迁移生成的DDL默认值静默失真。
func UnwrapSqlLiteral(val string) string {
	if len(val) < 2 || val[0] != '\'' || val[len(val)-1] != '\'' {
		return val
	}
	return strings.ReplaceAll(val[1:len(val)-1], "''", "'")
}

// IsQuotedSqlLiteral 判断文本是否是一个完整且合法的SQL字符串字面量：首尾各一个单引号，且内部单引号均成对出现。
//
// 必须做完整的成对校验，不能用“首尾是引号”或子串正则（如 .+' 匹配）代替：
//   - 前者会把 a'b'c 这类未包引号的原始值误判为字面量而裸拼进DDL（语法错误）；
//   - 后者会把 'abc' + 'def' 这类拼接表达式误判为单个字面量。
func IsQuotedSqlLiteral(val string) bool {
	if len(val) < 2 || val[0] != '\'' || val[len(val)-1] != '\'' {
		return false
	}
	inner := val[1 : len(val)-1]
	for i := 0; i < len(inner); i++ {
		if inner[i] != '\'' {
			continue
		}
		// 遇单引号：要么是转义的双写（跳过下一个），要么是不成对的异或（如字面量结束后还有内容）
		if i+1 < len(inner) && inner[i+1] == '\'' {
			i++
			continue
		}
		return false
	}
	return true
}

// IsSqlFunctionExpr 判断默认值原文是否是函数调用形式的表达式（如 now()、abs(-1)、date_trunc(...)）。
//
// 结构迁移为避免跨源函数不支持而会跳过函数类默认值，但旧实现统一用“含左括号即视为函数”判定，
// 使内容含括号的字符串默认值（如 '(0)'、'unknown (pending)'、'待确认(必填)'）被静默丢弃，
// 迁移后目标表缺少本应存活的默认值。本函数要求“以右括号结尾，且首个左括号之前的前缀是合法函数名”，
// 前缀不允许空格（unknown (pending) 这类自文本的函数名与括号间有空格，不是合法函数调用）但允许句点
// （pg_catalog.now()、dbo.fn() 这类限定名），含引号/非ASCII开头的内容均不会被误判为函数。
func IsSqlFunctionExpr(val string) bool {
	val = strings.TrimSpace(val)
	idx := strings.IndexByte(val, '(')
	if idx <= 0 || !strings.HasSuffix(val, ")") {
		return false
	}
	for i := 0; i < idx; i++ {
		c := val[i]
		isIdentChar := c == '_' || c == '.' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if !isIdentChar {
			return false
		}
	}
	return true
}

// AsSqlStringLiteral 若文本是带引号的SQL字符串字面量（允许SQL Server/Oracle的N前缀），返回其原始值。
//
// 仅在确定身处定义原文的括号包装内部时调用，不对顶层裸值启用N前缀处理：
// 源库为MySQL 8.0时裸 N'abc' 意味着默认值内容就是 N'abc'，误剥会静默改变值
func AsSqlStringLiteral(val string) (string, bool) {
	v := stripNationalPrefix(strings.TrimSpace(val))
	if !IsQuotedSqlLiteral(v) {
		return "", false
	}
	return UnwrapSqlLiteral(v), true
}

// anyStringContains 判断s是否包含subs中的任一片段（子串语义，适用于类型名关键字判定）
func anyStringContains(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// isParenWrapped 判断文本是否整体被一对圆括号包裹
func isParenWrapped(val string) bool {
	return len(val) >= 2 && val[0] == '(' && val[len(val)-1] == ')'
}

// UnwrapOuterParens 剥去整体平衡包裹的最外层圆括号，仅当首个'('配对的')'恰好位于末尾时才剥除。
//
// 必须做括号平衡扫描（并跳过字符串字面量内部的括号），不能用“首尾是括号”判断代替：
// (1)+(2) 首尾看似成对，按字符集剪后会得到 1)+(2 这类既非合法字面量也非合法表达式的垃圾内容。
// 返回剥除后的内容（已TrimSpace）与是否成功剥除一层。
func UnwrapOuterParens(val string) (string, bool) {
	val = strings.TrimSpace(val)
	if len(val) < 2 || val[0] != '(' {
		return "", false
	}
	depth := 0
	inStr := false
	for i := 0; i < len(val); i++ {
		c := val[i]
		if inStr {
			if c != '\'' {
				continue
			}
			// 连续两个单引号是字面量内部的转义，仍处于字符串内
			if i+1 < len(val) && val[i+1] == '\'' {
				i++
				continue
			}
			inStr = false
			continue
		}
		switch c {
		case '\'':
			inStr = true
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return "", false
			}
			if depth == 0 {
				// 与首个'('配对的右括号不在末尾，说明不是整体包裹（如 (a)+(b)）
				if i != len(val)-1 {
					return "", false
				}
				return strings.TrimSpace(val[1:i]), true
			}
		}
	}
	return "", false
}

// stripNationalPrefix 剥去SQL Server/Oracle的 N'xxx' 国际字符前缀（不影响普通字符串内容）
func stripNationalPrefix(val string) string {
	if len(val) > 1 && (val[0] == 'N' || val[0] == 'n') && val[1] == '\'' {
		return val[1:]
	}
	return val
}

var dateTimeLiteralRegexp = regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}([ T]\d{1,2}:\d{1,2}(:\d{1,2}(\.\d+)?)?)?$`)

var plainSqlKeywords = map[string]struct{}{
	"CURRENT_TIMESTAMP": {},
	"CURRENT_DATE":      {},
	"CURRENT_TIME":      {},
	"SYSTIMESTAMP":      {},
	"SYSDATE":           {},
	"LOCALTIME":         {},
	"LOCALTIMESTAMP":    {},
	"NULL":              {},
	"TRUE":              {},
	"FALSE":             {},
}

// isHexLiteral 判断是否0x十六进制字面量（mysql的binary/varbinary默认值呈现形式）
func isHexLiteral(val string) bool {
	if len(val) <= 2 || val[0] != '0' || (val[1] != 'x' && val[1] != 'X') {
		return false
	}
	for i := 2; i < len(val); i++ {
		if c := val[i]; !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// isBitLiteral 判断是否b'01'位字面量（mysql的bit列默认值呈现形式）
func isBitLiteral(val string) bool {
	if len(val) <= 3 || (val[0] != 'b' && val[0] != 'B') || val[1] != '\'' || val[len(val)-1] != '\'' {
		return false
	}
	for i := 2; i < len(val)-1; i++ {
		if val[i] != '0' && val[i] != '1' {
			return false
		}
	}
	return true
}

// IsPlainSqlLiteral 判断元数据返回的默认值原文是否可安全直接拼入SQL（不加引号）。
//
// 仅数字字面量、0x十六进制字面量、b'01'位字面量与少量无括号的关键字（如CURRENT_TIMESTAMP）成立；
// 其余形态（含未知类型/枚举等字符串默认值）一律按字符串字面量引用后再输出，既修复了
// enum/set等未被引号类型清单覆盖的列直接裸拼产生的语法错误，也封堵了
// “元数据文本被当作SQL片段执行”的注入面。
//
// 注：本函数只判定形态，能否裸拼还取决于目标列类型（字符串类列必须引用，见GenColumnDefaultSql）
func IsPlainSqlLiteral(val string) bool {
	if val == "" {
		return false
	}
	if IsNumericLiteral(val) {
		return true
	}
	if _, ok := plainSqlKeywords[strings.ToUpper(val)]; ok {
		return true
	}
	if isHexLiteral(val) || isBitLiteral(val) {
		return true
	}
	return false
}
