package sqlstmt

import "strings"

// ExprNode 条件表达式AST节点接口（WHERE/HAVING/ON等场景的结构化表示）。
// 每个节点保留其覆盖的原始SQL文本（Text），供上层回填与降级使用；
// String() 返回规范化序列化形式，供跨方言一致性断言与调试
type ExprNode interface {
	// Text 节点覆盖的原始SQL文本
	Text() string
	// String 规范化序列化（大小写归一，用于结构比对）
	String() string
}

// NodeBase 节点公共部分：节点覆盖的原始SQL文本（解析器构造节点时填充）
type NodeBase struct {
	Raw string
}

func (n NodeBase) Text() string { return n.Raw }

// Literal 字面量：数字/字符串/NULL/TRUE/FALSE/占位参数
type Literal struct {
	NodeBase
	Value string // 原始字面量文本（含引号）
}

func (n *Literal) String() string { return n.Value }

// ColumnRef 列引用：支持 a.b.c 限定形式，Parts保留原始片段（含引号原文）
type ColumnRef struct {
	NodeBase
	Parts []string
}

func (n *ColumnRef) String() string { return strings.Join(n.Parts, ".") }

// Name 返回去引号后的列名（最后一段），供脱敏等按列名匹配的场景使用
func (n *ColumnRef) Name() string {
	if len(n.Parts) == 0 {
		return ""
	}
	last := n.Parts[len(n.Parts)-1]
	last = strings.TrimSuffix(strings.TrimPrefix(last, "`"), "`")
	last = strings.TrimSuffix(strings.TrimPrefix(last, `"`), `"`)
	return last
}

// UnaryExpr 一元运算：NOT / - / +
type UnaryExpr struct {
	NodeBase
	Op      string
	Operand ExprNode
}

func (n *UnaryExpr) String() string {
	return "(" + n.Op + " " + n.Operand.String() + ")"
}

// BinaryExpr 二元运算：算术/比较/逻辑/拼接/JSON等（AND OR = <> < <= > >= + - * / || :: -> ->> ...）
type BinaryExpr struct {
	NodeBase
	Op    string // 规范化为大写
	Left  ExprNode
	Right ExprNode
}

func (n *BinaryExpr) String() string {
	return "(" + n.Left.String() + " " + n.Op + " " + n.Right.String() + ")"
}

// ParenExpr 括号表达式（优先级覆盖）
type ParenExpr struct {
	NodeBase
	Inner ExprNode
}

func (n *ParenExpr) String() string { return "(" + n.Inner.String() + ")" }

// FuncCall 函数调用
type FuncCall struct {
	NodeBase
	Name     string // 大写
	Args     []ExprNode
	Star     bool // COUNT(*) 形态
	Distinct bool // COUNT(DISTINCT a) 形态
}

func (n *FuncCall) String() string {
	parts := make([]string, 0, len(n.Args))
	for _, a := range n.Args {
		parts = append(parts, a.String())
	}
	inner := strings.Join(parts, ", ")
	if n.Star {
		inner = "*"
	} else if n.Distinct {
		inner = "DISTINCT " + inner
	}
	return n.Name + "(" + inner + ")"
}

// CaseExpr CASE WHEN表达式
type CaseExpr struct {
	NodeBase
	Operand  ExprNode // 可为nil（搜索形态 CASE WHEN cond...）
	Whens    []CaseWhen
	ElseExpr ExprNode // 可为nil
}

type CaseWhen struct {
	When ExprNode
	Then ExprNode
}

func (n *CaseExpr) String() string {
	var b strings.Builder
	b.WriteString("CASE")
	if n.Operand != nil {
		b.WriteString(" " + n.Operand.String())
	}
	for _, w := range n.Whens {
		b.WriteString(" WHEN " + w.When.String() + " THEN " + w.Then.String())
	}
	if n.ElseExpr != nil {
		b.WriteString(" ELSE " + n.ElseExpr.String())
	}
	b.WriteString(" END")
	return b.String()
}

// InExpr IN / NOT IN（列表或子查询二选一）
type InExpr struct {
	NodeBase
	Operand  ExprNode
	Not      bool
	List     []ExprNode  // IN (1, 2, 3)
	Subquery *SelectStmt // IN (SELECT ...)
}

func (n *InExpr) String() string {
	op := "IN"
	if n.Not {
		op = "NOT IN"
	}
	if n.Subquery != nil {
		return "(" + n.Operand.String() + " " + op + " (SUBQUERY))"
	}
	parts := make([]string, 0, len(n.List))
	for _, e := range n.List {
		parts = append(parts, e.String())
	}
	return "(" + n.Operand.String() + " " + op + " (" + strings.Join(parts, ", ") + "))"
}

// BetweenExpr BETWEEN AND / NOT BETWEEN
type BetweenExpr struct {
	NodeBase
	Operand ExprNode
	Not     bool
	Low     ExprNode
	High    ExprNode
}

func (n *BetweenExpr) String() string {
	op := "BETWEEN"
	if n.Not {
		op = "NOT BETWEEN"
	}
	return "(" + n.Operand.String() + " " + op + " " + n.Low.String() + " AND " + n.High.String() + ")"
}

// LikeExpr LIKE / NOT LIKE（可含ESCAPE）
type LikeExpr struct {
	NodeBase
	Operand ExprNode
	Not     bool
	Pattern ExprNode
	Escape  ExprNode // 可为nil
}

func (n *LikeExpr) String() string {
	op := "LIKE"
	if n.Not {
		op = "NOT LIKE"
	}
	s := "(" + n.Operand.String() + " " + op + " " + n.Pattern.String()
	if n.Escape != nil {
		s += " ESCAPE " + n.Escape.String()
	}
	return s + ")"
}

// IsNullExpr IS NULL / IS NOT NULL
type IsNullExpr struct {
	NodeBase
	Operand ExprNode
	Not     bool
}

func (n *IsNullExpr) String() string {
	if n.Not {
		return "(" + n.Operand.String() + " IS NOT NULL)"
	}
	return "(" + n.Operand.String() + " IS NULL)"
}

// ExistsExpr [NOT] EXISTS (子查询)
type ExistsExpr struct {
	NodeBase
	Not      bool
	Subquery *SelectStmt
}

func (n *ExistsExpr) String() string {
	if n.Not {
		return "(NOT EXISTS (SUBQUERY))"
	}
	return "(EXISTS (SUBQUERY))"
}

// SubQueryExpr 子查询表达式（标量子查询/括号内SELECT）。
// SelectParser回调不可用或解析失败时降级为仅含原文文本的节点
type SubQueryExpr struct {
	NodeBase
	Select *SelectStmt // 可为nil
}

func (n *SubQueryExpr) String() string { return "(SUBQUERY)" }
