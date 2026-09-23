package mask

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// 查询文本/AST 的轻量判断与清洗辅助。

// extractParenInner 返回最外层括号内的内容；非括号起始或括号不配对返回空串。
// 需跳过字符串字面量内的括号，避免 `'(a)'` 之类误判配对。
func extractParenInner(text string) string {
	if !strings.HasPrefix(text, "(") {
		return ""
	}
	depth := 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\'': // 跳过单引号字符串字面量（含''转义），避免字面量括号干扰配对
			for i++; i < len(text); i++ {
				if text[i] == '\'' {
					if i+1 < len(text) && text[i+1] == '\'' {
						i++
						continue
					}
					break
				}
			}
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return text[1:i]
			}
		}
	}
	return ""
}

// stripTableAlias 去除列名的表别名前缀与引用符，如 t.phone -> phone、t."PHONE" -> PHONE
func stripTableAlias(column string) string {
	if idx := lastDotIdx(column); idx >= 0 {
		column = column[idx+1:]
	}
	return strings.Trim(column, "`\"'")
}

// stripTableName 去除表名的库名前缀与引用符，如 `db`.`t_user` -> t_user
func stripTableName(table string) string {
	table = strings.Trim(table, "`\"'")
	if idx := lastDotIdx(table); idx >= 0 {
		table = table[idx+1:]
	}
	return table
}

func lastDotIdx(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

// stripExprAliasSuffix 去除select项文本尾部的 AS 别名（大小写不敏感），返回表达式主体
func stripExprAliasSuffix(text string) string {
	if idx := strings.LastIndex(strings.ToUpper(text), " AS "); idx >= 0 {
		return strings.TrimSpace(text[:idx])
	}
	return strings.TrimSpace(text)
}

// exprColRef 查询内列引用的血缘信息（供表达式项token匹配与派生表血缘传播）
type exprColRef struct {
	colName string // 源列名（保留原始大小写）
	table   string // 来源表（限定名场景）；空串且locked=true表示确定无来源表，仅全局规则解析
	locked  bool   // 血缘是否锁定（锁定后不再按tables列表顺序逐表尝试兜底）
}

// isExprItem 判断select项是否为表达式/函数项。
// 项类型分类已收敛到解析器（base.ClassifySelectItem，跨方言Kind语义一致），应用层仅消费Kind
func isExprItem(item sqlstmt.SelectItem) bool {
	return item.Kind == sqlstmt.SelectItemExpr || item.Kind == sqlstmt.SelectItemFunction
}
