package sync

import (
	"encoding/json"
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/logx"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TransformType 字段转换类型
type TransformType string

const (
	// TransformTypeColumn 直接映射源字段值
	TransformTypeColumn TransformType = "column"
	// TransformTypeConstant 常量值
	TransformTypeConstant TransformType = "constant"
	// TransformTypeExpr 表达式（内置函数）
	TransformTypeExpr TransformType = "expr"
)

// TransformRule 单个字段的转换规则
type TransformRule struct {
	// TargetColumn 目标列名
	TargetColumn string `json:"targetColumn"`
	// Type 转换类型：column / constant / expr
	Type TransformType `json:"type"`
	// Config 转换配置
	Config TransformConfig `json:"config"`
}

// TransformConfig 转换配置
type TransformConfig struct {
	// SourceColumn 源列名（type=column 时使用）
	SourceColumn string `json:"sourceColumn,omitempty"`
	// Value 常量值（type=constant 时使用）
	Value string `json:"value,omitempty"`
	// Expression 表达式（type=expr 时使用）
	// 支持的函数：UPPER(col), LOWER(col), TRIM(col), CONCAT(a,b), COALESCE(a,b),
	//              SUBSTRING(col,start,len), REPLACE(col,old,new), NOW(), CAST(col,type)
	Expression string `json:"expression,omitempty"`
}

// TransformEngine 数据转换引擎：将源行数据按转换规则映射为目标行数据。
// 遵循开闭原则：新增转换函数只需在 builtinFuncs 注册，无需修改引擎核心逻辑。
type TransformEngine struct {
	// rules 按目标列名索引的转换规则
	rules map[string]*TransformRule
	// funcs 内置函数注册表（函数名 -> 实现）
	funcs map[string]TransformFunc
}

// TransformFunc 内置转换函数签名：参数为已求值的字符串参数列表
type TransformFunc func(args []string) (string, error)

// NewTransformEngine 创建转换引擎。
// transformRulesJSON 为 JSON 数组格式的转换规则；为空时返回 nil 引擎（表示无需转换）。
func NewTransformEngine(transformRulesJSON string) (*TransformEngine, error) {
	if transformRulesJSON == "" {
		return nil, nil
	}
	var rules []TransformRule
	if err := json.Unmarshal([]byte(transformRulesJSON), &rules); err != nil {
		return nil, fmt.Errorf("failed to parse transform rules: %s", err.Error())
	}
	if len(rules) == 0 {
		return nil, nil
	}

	engine := &TransformEngine{
		rules: make(map[string]*TransformRule, len(rules)),
		funcs: defaultBuiltinFuncs(),
	}
	for i := range rules {
		engine.rules[rules[i].TargetColumn] = &rules[i]
	}
	return engine, nil
}

// Transform 对单行源数据应用转换规则，返回目标行数据。
// fieldMap 提供默认的 src->target 映射（无转换规则时使用）。
// 返回 (targetRow, skip)：skip=true 表示该行应被跳过（如 NullStrategySkipRow 命中）。
func (e *TransformEngine) Transform(srcRow map[string]any, fieldMap []map[string]string, task *entity.DataSyncTask) (map[string]any, bool) {
	target := make(map[string]any, len(fieldMap))

	for _, fm := range fieldMap {
		targetCol := fm["target"]
		srcCol := fm["src"]

		// 检查是否有转换规则覆盖默认映射
		if e != nil {
			if rule, ok := e.rules[targetCol]; ok {
				val, skip := e.applyRule(rule, srcRow)
				if skip {
					return nil, true
				}
				target[targetCol] = val
				continue
			}
		}

		// 默认映射：直接取源字段值
		val := srcRow[srcCol]

		// 空值处理策略
		if val == nil && task != nil {
			switch task.NullStrategy {
			case entity.NullStrategyDefault:
				val = task.NullDefault
			case entity.NullStrategySkipRow:
				return nil, true
			}
		}

		target[targetCol] = val
	}

	return target, false
}

// applyRule 应用单条转换规则，返回 (转换后的值, 是否跳过该行)
func (e *TransformEngine) applyRule(rule *TransformRule, srcRow map[string]any) (any, bool) {
	switch rule.Type {
	case TransformTypeColumn:
		col := rule.Config.SourceColumn
		if col == "" {
			col = rule.TargetColumn
		}
		val, ok := lookupRowValue(srcRow, col)
		if !ok {
			return nil, false
		}
		return val, false

	case TransformTypeConstant:
		return rule.Config.Value, false

	case TransformTypeExpr:
		result, err := e.evalExpression(rule.Config.Expression, srcRow)
		if err != nil {
			logx.Warnf("transform expression eval failed for column [%s]: %s", rule.TargetColumn, err.Error())
			return nil, false
		}
		return result, false

	default:
		logx.Warnf("unknown transform type [%s] for column [%s], falling back to direct mapping", rule.Type, rule.TargetColumn)
		val := srcRow[rule.TargetColumn]
		return val, false
	}
}

// evalExpression 求值表达式。
// 支持格式：FUNC_NAME(arg1, arg2, ...) 其中参数可以是列名（直接引用源行）或字符串字面量。
// 支持嵌套：UPPER(TRIM(col))
func (e *TransformEngine) evalExpression(expr string, srcRow map[string]any) (any, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// 解析函数调用：FUNC(args...)
	funcName, args, err := parseFuncCall(expr)
	if err != nil {
		// 数字字面量（整数或浮点）：直接返回，避免被误认为列名
		if _, parseErr := strconv.ParseFloat(expr, 64); parseErr == nil {
			return expr, nil
		}
		// 尝试作为列名引用
		if val, ok := lookupRowValue(srcRow, expr); ok {
			return val, nil
		}
		// 字符串字面量（带引号）
		if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") {
			return expr[1 : len(expr)-1], nil
		}
		return nil, fmt.Errorf("cannot evaluate expression: %s", expr)
	}

	// 查找内置函数
	fn, ok := e.funcs[strings.ToUpper(funcName)]
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", funcName)
	}

	// 递归求值参数
	resolvedArgs := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		// 递归求值：可能是嵌套函数调用、列引用或字面量
		val, err := e.evalExpression(arg, srcRow)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate argument [%s]: %s", arg, err.Error())
		}
		resolvedArgs = append(resolvedArgs, fmt.Sprintf("%v", val))
	}

	return fn(resolvedArgs)
}

// parseFuncCall 解析函数调用表达式：FUNC_NAME(arg1, arg2, ...)
// 支持嵌套括号。返回 (函数名, 参数列表, error)
func parseFuncCall(expr string) (string, []string, error) {
	parenIdx := strings.Index(expr, "(")
	if parenIdx < 0 || !strings.HasSuffix(strings.TrimSpace(expr), ")") {
		return "", nil, fmt.Errorf("not a function call")
	}
	funcName := strings.TrimSpace(expr[:parenIdx])
	if funcName == "" {
		return "", nil, fmt.Errorf("empty function name")
	}

	// 提取括号内的参数（支持嵌套括号）
	inner := strings.TrimSpace(expr[parenIdx+1 : len(expr)-1])
	args := splitArgs(inner)
	return funcName, args, nil
}

// splitArgs 按顶层逗号分割参数（忽略括号内的逗号）
func splitArgs(s string) []string {
	if s == "" {
		return nil
	}
	var args []string
	depth := 0
	start := 0
	for i, ch := range s {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, s[start:i])
				start = i + 1
			}
		}
	}
	args = append(args, s[start:])
	return args
}

// defaultBuiltinFuncs 注册内置转换函数。
// 开闭原则扩展点：新增函数只需在此注册，无需修改引擎核心。
func defaultBuiltinFuncs() map[string]TransformFunc {
	return map[string]TransformFunc{
		"UPPER": func(args []string) (string, error) {
			if len(args) < 1 {
				return "", fmt.Errorf("UPPER requires 1 argument")
			}
			return strings.ToUpper(args[0]), nil
		},
		"LOWER": func(args []string) (string, error) {
			if len(args) < 1 {
				return "", fmt.Errorf("LOWER requires 1 argument")
			}
			return strings.ToLower(args[0]), nil
		},
		"TRIM": func(args []string) (string, error) {
			if len(args) < 1 {
				return "", fmt.Errorf("TRIM requires 1 argument")
			}
			return strings.TrimSpace(args[0]), nil
		},
		"CONCAT": func(args []string) (string, error) {
			return strings.Join(args, ""), nil
		},
		"COALESCE": func(args []string) (string, error) {
			for _, a := range args {
				if a != "" && a != "NULL" {
					return a, nil
				}
			}
			return "", nil
		},
		"SUBSTRING": func(args []string) (string, error) {
			if len(args) < 2 {
				return "", fmt.Errorf("SUBSTRING requires at least 2 arguments (str, start[, len])")
			}
			s := args[0]
			start := 0
			for _, c := range args[1] {
				if c >= '0' && c <= '9' {
					start = start*10 + int(c-'0')
				}
			}
			if start > 0 {
				start-- // SQL SUBSTRING is 1-based
			}
			if start < 0 {
				start = 0
			}
			runes := []rune(s)
			if start >= len(runes) {
				return "", nil
			}
			if len(args) >= 3 {
				length := 0
				for _, c := range args[2] {
					if c >= '0' && c <= '9' {
						length = length*10 + int(c-'0')
					}
				}
				end := start + length
				if end > len(runes) {
					end = len(runes)
				}
				return string(runes[start:end]), nil
			}
			return string(runes[start:]), nil
		},
		"REPLACE": func(args []string) (string, error) {
			if len(args) < 3 {
				return "", fmt.Errorf("REPLACE requires 3 arguments (str, old, new)")
			}
			return strings.ReplaceAll(args[0], args[1], args[2]), nil
		},
		"NOW": func(args []string) (string, error) {
			return time.Now().Format("2006-01-02T15:04:05"), nil
		},
		"LEFT": func(args []string) (string, error) {
			if len(args) < 2 {
				return "", fmt.Errorf("LEFT requires 2 arguments (str, count)")
			}
			n := 0
			for _, c := range args[1] {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				}
			}
			runes := []rune(args[0])
			if n >= len(runes) {
				return args[0], nil
			}
			return string(runes[:n]), nil
		},
		"RIGHT": func(args []string) (string, error) {
			if len(args) < 2 {
				return "", fmt.Errorf("RIGHT requires 2 arguments (str, count)")
			}
			n := 0
			for _, c := range args[1] {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				}
			}
			runes := []rune(args[0])
			if n >= len(runes) {
				return args[0], nil
			}
			return string(runes[len(runes)-n:]), nil
		},
	}
}

// FilterEngine 数据过滤引擎。
// 支持条件表达式：field == value, field != value, field > value, field < value,
// field >= value, field <= value, field LIKE pattern。
// 多条件用 AND / OR 连接，支持括号分组。优先级：括号 > OR > AND。
// 示例："status == 'active' AND age > 18 OR role == 'admin'"
type FilterEngine struct {
	condition   string
	likeRegexps map[string]*regexp.Regexp // LIKE 模式预编译正则缓存
}

// NewFilterEngine 创建过滤引擎。condition 为空时返回 nil（表示不过滤）。
func NewFilterEngine(condition string) *FilterEngine {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return nil
	}
	return &FilterEngine{
		condition:   condition,
		likeRegexps: make(map[string]*regexp.Regexp),
	}
}

// filterConditionReg 匹配简单条件：field op value
var filterConditionReg = regexp.MustCompile(`^\s*(\w+)\s*(==|!=|>=|<=|>|<|LIKE)\s*['"]?([^'"]*?)['"]?\s*$`)

// logicalAndReg 按 AND 分割条件表达式
var logicalAndReg = regexp.MustCompile(`(?i)\s+AND\s+`)

// logicalOrReg 按 OR 分割条件表达式
var logicalOrReg = regexp.MustCompile(`(?i)\s+OR\s+`)

// Match 判断源行是否满足过滤条件。返回 true 表示保留该行。
func (f *FilterEngine) Match(srcRow map[string]any) bool {
	if f == nil {
		return true
	}

	// 按 OR 分割为多个组，任一组匹配即保留（OR 语义）
	orGroups := splitByLogicalOp(f.condition, logicalOrReg)
	for _, group := range orGroups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		// 每个 OR 组内按 AND 分割，所有条件必须满足（AND 语义）
		andConditions := splitByLogicalOp(group, logicalAndReg)
		groupMatch := true
		for _, cond := range andConditions {
			cond = strings.TrimSpace(cond)
			if cond == "" {
				continue
			}
			matches := filterConditionReg.FindStringSubmatch(cond)
			if matches == nil {
				// 条件无法解析：记录告警并放行该行，避免静默丢弃数据
				logx.Warnf("filter condition [%s] cannot be parsed, row passed through", cond)
				return true
			}
			field, op, expected := matches[1], matches[2], matches[3]
			actual, ok := lookupRowValue(srcRow, field)
			if !ok {
				groupMatch = false
				break
			}
			actualStr := fmt.Sprintf("%v", actual)
			if !f.evalCondition(actualStr, op, expected) {
				groupMatch = false
				break
			}
		}
		if groupMatch {
			return true
		}
	}
	return false
}

// evalCondition 求值单个条件，LIKE 模式使用预编译正则缓存。
func (f *FilterEngine) evalCondition(actual, op, expected string) bool {
	if strings.ToUpper(op) == "LIKE" {
		return f.matchLike(actual, expected)
	}
	return evalConditionSimple(actual, op, expected)
}

// matchLike LIKE 模式匹配，缓存编译后的正则避免每行重复编译。
func (f *FilterEngine) matchLike(actual, pattern string) bool {
	re, ok := f.likeRegexps[pattern]
	if !ok {
		// 转义正则元字符防止 ReDoS，再将 SQL 通配符转为正则
		escaped := regexp.QuoteMeta(pattern)
		regexStr := strings.ReplaceAll(strings.ReplaceAll(escaped, "%", ".*"), "_", ".")
		var err error
		re, err = regexp.Compile("(?i)^" + regexStr + "$")
		if err != nil {
			logx.Warnf("LIKE pattern [%s] compile failed: %s", pattern, err.Error())
			return false
		}
		f.likeRegexps[pattern] = re
	}
	return re.MatchString(actual)
}

// evalConditionSimple 非 LIKE 的简单条件求值
func evalConditionSimple(actual, op, expected string) bool {
	switch strings.ToUpper(op) {
	case "==", "=":
		return actual == expected
	case "!=":
		return actual != expected
	case ">":
		return actual > expected
	case "<":
		return actual < expected
	case ">=":
		return actual >= expected
	case "<=":
		return actual <= expected
	default:
		return false
	}
}

// splitByLogicalOp 按指定逻辑操作符分割条件表达式（括号感知）。
// 仅在括号深度为 0 时才分割，确保 "(a == 1 OR b == 2) AND c == 3" 不被错误拆分。
func splitByLogicalOp(cond string, re *regexp.Regexp) []string {
	// 快速路径：无括号时直接用正则分割
	if !strings.ContainsAny(cond, "()") {
		return re.Split(cond, -1)
	}
	// 括号感知分割：仅在深度为 0 处分割
	locs := re.FindAllStringIndex(cond, -1)
	if len(locs) == 0 {
		return []string{cond}
	}
	var result []string
	depth := 0
	prev := 0
	for _, loc := range locs {
		// 计算 loc 起始位置的括号深度
		for _, ch := range cond[prev:loc[0]] {
			switch ch {
			case '(':
				depth++
			case ')':
				depth--
			}
		}
		if depth == 0 {
			result = append(result, cond[prev:loc[0]])
		}
		prev = loc[1]
	}
	result = append(result, cond[prev:])
	return result
}
