package mcm

import (
	"regexp"
	"strings"
)

// ============================================================================
// 统一命令分析器
//
// 提供 shell 命令的词法分析、安全模式检测和白名单匹配能力。
// 所有命令执行入口（AI Agent、Web 终端、API RunCmd、脚本执行）共享此分析器，
// 确保命令解析逻辑一致，避免各入口各自实现导致安全策略不一致。
//
// 安全原则：
//   - 链式命令中的每一个命令段都必须独立通过白名单检测
//   - 命令替换（$()、反引号）和参数展开（${}）因无法静态分析内部命令，一律视为不安全
//   - 重定向（>、<、>>）意味着文件写入或非常规输入，视为不安全
//   - 有 shell 逃逸能力的命令（awk、find -exec、sed e 等）不应加入白名单
// ============================================================================

// CmdFilterRule 命令过滤规则（经编译的正则表达式 + 策略描述）
type CmdFilterRule struct {
	CmdRegexp *regexp.Regexp // 命令正则表达式
	Strategy  string         // 策略（如 "reject"、"approval" 等）
}

// WhitelistRule 白名单命令规则
type WhitelistRule struct {
	Command     string   // 命令名（如 "ls"、"cat"）
	AllowedArgs []string // 允许的参数模式（空/nil 表示所有参数都允许）
}

// Tokenize 将命令字符串分割成 tokens（导出版，供所有入口共享）
//
// 分割规则：
//   - 空白字符（空格、制表符）分隔参数
//   - 操作符字符（|、&、;、>、<、\n、\r）作为独立 token 切出
//   - 引号（单引号、双引号）内的内容保持完整
//   - 反斜杠转义下一个字符（单引号内不处理转义）
func Tokenize(cmd string) []string {
	var tokens []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && !inSingleQuote {
			current.WriteByte(ch)
			escaped = true
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			current.WriteByte(ch)
			continue
		}

		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			current.WriteByte(ch)
			continue
		}

		if inSingleQuote || inDoubleQuote {
			current.WriteByte(ch)
			continue
		}

		if isOperatorChar(ch) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}

			operator := string(ch)
			if i+1 < len(cmd) && isOperatorChar(cmd[i+1]) {
				nextCh := cmd[i+1]
				if ch == nextCh {
					operator += string(nextCh)
					i++
				}
			}

			tokens = append(tokens, operator)
			continue
		}

		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// HasUnsafePattern 检测命令文本中是否包含无法静态分析或可能修改系统的不安全 shell 模式：
//   - $(...) 命令替换
//   - `...` 反引号命令替换
//   - ${...} 参数展开（可嵌套命令替换）
//   - <(...) / >(...) 进程替换
//   - >、<、>> 重定向（白名单命令均为只读类，重定向意味着写文件或读非常规输入，需审批）
func HasUnsafePattern(cmd string) bool {
	// 检测 $ 后跟 ( 或 {（命令替换、参数展开）
	if strings.Contains(cmd, "$(") || strings.Contains(cmd, "${") {
		return true
	}
	// 检测反引号命令替换
	if strings.Contains(cmd, "`") {
		return true
	}
	// 检测重定向操作符（>、>>、<）：白名单命令均为只读类，重定向意味着文件写入或非常规输入
	for _, ch := range cmd {
		if ch == '>' || ch == '<' {
			return true
		}
	}
	return false
}

// IsWhitelistCommand 判断命令是否完全在给定白名单规则中，可以自动执行。
// 链式命令（;、&&、||、|、换行分隔）中的每一个命令段都必须独立通过白名单检测。
func IsWhitelistCommand(cmd string, rules []WhitelistRule) bool {
	if cmd == "" {
		return false
	}

	tokens := Tokenize(cmd)
	if len(tokens) == 0 {
		return false
	}

	segments := splitByTopLevelSeparators(tokens)
	for _, segment := range segments {
		if !validateWhitelistSegment(segment, rules) {
			return false
		}
	}
	return true
}

// MatchCmdFilters 检查命令是否匹配给定的过滤规则列表（正则匹配）。
// 返回第一个匹配的规则；若无匹配返回 nil。
// 用于 MachineCmdConf 等基于正则的命令过滤。
func MatchCmdFilters(cmd string, filters []*CmdFilterRule) *CmdFilterRule {
	for _, filter := range filters {
		if filter.CmdRegexp.MatchString(cmd) {
			return filter
		}
	}
	return nil
}

// ExtractCommandNames 提取命令字符串中所有命令段的命令名（去除路径前缀）。
// 用于审计日志、命令分类等场景。
func ExtractCommandNames(cmd string) []string {
	tokens := Tokenize(cmd)
	segments := splitByTopLevelSeparators(tokens)

	var names []string
	for _, segment := range segments {
		if name := extractCommandName(segment); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// ============================================================================
// 内部辅助函数
// ============================================================================

// splitByTopLevelSeparators 将 token 列表按顶层命令分隔符拆分为独立命令段。
// 仅按 ;、&&、||、|、换行 拆分（这些是真正的命令分隔符）；
// 重定向符 >、< 不作为分隔符（属于命令内部语法，由 HasUnsafePattern 检测并拦截）。
func splitByTopLevelSeparators(tokens []string) [][]string {
	var segments [][]string
	var current []string

	for _, token := range tokens {
		if isCommandSeparator(token) {
			if len(current) > 0 {
				segments = append(segments, current)
				current = nil
			}
			continue
		}
		current = append(current, token)
	}
	if len(current) > 0 {
		segments = append(segments, current)
	}
	return segments
}

// validateWhitelistSegment 验证单个命令段是否在白名单中。
// 检查顺序：1. 不安全模式检测 → 2. 命令名白名单匹配 → 3. 参数合规性校验
func validateWhitelistSegment(tokens []string, rules []WhitelistRule) bool {
	if len(tokens) == 0 {
		return true // 空段（如连续分隔符产生）视为无害
	}

	// 重组为文本做不安全模式检测（命令替换、反引号、重定向等无法静态分析内部命令）
	segmentText := strings.Join(tokens, " ")
	if HasUnsafePattern(segmentText) {
		return false
	}

	// 找到第一个非引号 token 作为命令名
	cmdIdx := -1
	for i, token := range tokens {
		if !isQuotedString(token) {
			cmdIdx = i
			break
		}
	}
	if cmdIdx < 0 {
		return false // 全是引号，异常输入
	}

	cmdToken := tokens[cmdIdx]
	cmdName := ExtractCmdName(cmdToken)

	// 在规则列表中查找匹配的命令
	for _, rule := range rules {
		if cmdName != rule.Command {
			continue
		}

		// 命令匹配成功：如果没有参数限制（AllowedArgs 为空），该命令直接通过
		if len(rule.AllowedArgs) == 0 {
			return true
		}

		// 有参数限制：仅校验第一个非选项参数（子命令），
		// 后续位置参数（包名、文件名等）不做限制。
		// 例如：yum list installed → 仅检查 "list" 是否在 allowedArgs 中
		if !validateFirstNonFlagArg(tokens[cmdIdx+1:], rule.AllowedArgs) {
			return false
		}
		return true
	}

	// 命令不在白名单中
	return false
}

// extractCommandName 从单个命令段的 tokens 中提取命令名（去除路径前缀）。
func extractCommandName(tokens []string) string {
	for _, token := range tokens {
		if !isQuotedString(token) && !isOperator(token) {
			return ExtractCmdName(token)
		}
	}
	return ""
}

// ExtractCmdName 从命令 token 中提取命令名（去除路径前缀）。
// 例如："/usr/bin/ls" → "ls"，"cat" → "cat"
func ExtractCmdName(cmdToken string) string {
	if idx := strings.LastIndex(cmdToken, "/"); idx >= 0 {
		return cmdToken[idx+1:]
	}
	return cmdToken
}

// validateFirstNonFlagArg 校验 tokens 中第一个非选项参数（子命令）是否在允许列表中。
// 选项标志（- 开头的）跳过；找到第一个非选项参数后检查是否在 allowedArgs 中。
// 后续位置参数（包名、文件名等）不做限制。
func validateFirstNonFlagArg(tokens []string, allowedArgs []string) bool {
	for _, token := range tokens {
		// 跳过操作符和引号
		if isOperator(token) || isQuotedString(token) {
			continue
		}
		// 跳过选项标志（- 开头的）
		if strings.HasPrefix(token, "-") {
			continue
		}
		// 找到第一个非选项参数，检查是否在允许列表中
		for _, allowed := range allowedArgs {
			if token == allowed {
				return true
			}
		}
		return false // 第一个非选项参数不在允许列表中
	}
	return true // 没有非选项参数，视为通过
}

// isCommandSeparator 判断 token 是否为命令分隔符（;、&&、||、|、换行）。
// 注意：>、< 是重定向符，不是命令分隔符。
func isCommandSeparator(token string) bool {
	if len(token) == 0 {
		return false
	}
	switch token {
	case ";", "&&", "||", "|", "\n", "\r":
		return true
	}
	return false
}

// isOperatorChar 判断字符是否为操作符字符。
// 包含 shell 命令分隔符（;、换行）、管道/逻辑运算符（|、&）、重定向符（>、<）
func isOperatorChar(ch byte) bool {
	return ch == '|' || ch == '&' || ch == ';' || ch == '>' || ch == '<' ||
		ch == '\n' || ch == '\r'
}

// isOperator 判断 token 是否为操作符
func isOperator(token string) bool {
	if len(token) == 0 {
		return false
	}
	return isOperatorChar(token[0])
}

// isQuotedString 判断是否为引号字符串
func isQuotedString(token string) bool {
	if len(token) < 2 {
		return false
	}
	return (token[0] == '\'' && token[len(token)-1] == '\'') ||
		(token[0] == '"' && token[len(token)-1] == '"')
}
