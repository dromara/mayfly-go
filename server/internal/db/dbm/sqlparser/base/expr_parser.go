package base

// 表达式解析器：Pratt（优先级爬升）解析，将 WHERE/HAVING/ON 等条件表达式
// 解析为 sqlstmt.ExprNode AST。
//
// 设计要点：
//   - 运算符优先级集中在 infixPrecedence 表，新运算符通过 RegisterInfixOp 注册
//     （开闭原则：无需修改解析循环）
//   - 解析失败一律返回 ok=false 并由调用方恢复Pos降级为文本跳过（SkipExpr），
//     保证「宽容解析」语义：任何SQL不会因表达式建树失败而解析中断
//   - 子句边界（FROM/WHERE/GROUP...）直接终止解析，边界语义复用 IsExprEnd/IsCondEnd

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// 优先级命名常量：infixPrecedence 表的语义档位，供解析逻辑引用避免魔法数字
const (
	precLogicalOr  = 1
	precLogicalAnd = 2
	precComparison = 3
	precUnarySign  = 9 // 一元+/-绑定最紧
)

// infixPrecedence 二元运算符优先级表（值越大绑定越紧，同值左结合）
var infixPrecedence = map[string]int{
	"OR":  precLogicalOr,
	"AND": precLogicalAnd,
	// 比较与谓词类
	"=": precComparison, "!=": precComparison, "<>": precComparison, "<": precComparison, "<=": precComparison, ">": precComparison, ">=": precComparison,
	"LIKE": precComparison, "IN": precComparison, "BETWEEN": precComparison, "IS": precComparison,
	// 拼接/JSON（标准中高于比较）
	"||": 4, "->": 5, "->>": 5,
	// 算术
	"+": 6, "-": 6, "*": 7, "/": 7, "%": 7,
	// pg 类型转换（后缀二元，最高）
	"::": 8,
}

// RegisterInfixOp 注册新的二元运算符优先级（扩展点：新方言/新运算符无需改动解析循环）
func RegisterInfixOp(op string, precedence int) {
	infixPrecedence[strings.ToUpper(op)] = precedence
}

// SelectParser 子查询解析回调类型：由各方言Parser注入到 Lexer 实例
// （parseSelect的包装），使子查询可建 SelectStmt 树；为nil时子查询降级为文本节点
type SelectParser func() (*sqlstmt.SelectStmt, bool)

// ParseCondExpr 解析条件表达式（WHERE/HAVING/ON）为 Expr：尽力建树（Root），
// 失败恢复Pos降级为纯文本。on=true时使用ON场景边界（其后可能紧跟JOIN子句）
func (l *Lexer) ParseCondExpr(on bool) *sqlstmt.Expr {
	start := l.Pos
	root, ok := l.parseExprPrec(0, on)
	if ok && l.Pos > start {
		return &sqlstmt.Expr{Text: l.TextFromExclusive(start), Root: root}
	}
	// 降级：恢复位置走文本跳过
	l.Pos = start
	l.SkipExpr()
	return &sqlstmt.Expr{Text: l.TextFromExclusive(start)}
}

// atCondBoundary 表达式是否到达子句边界（on场景额外含JOIN前缀）
func (l *Lexer) atCondBoundary(on bool) bool {
	if l.Current().IsEOF() {
		return true
	}
	if on {
		return l.IsCondEnd()
	}
	return l.IsExprEnd()
}

// parseExprPrec Pratt主循环：解析优先级 >= minPrec 的中缀链
func (l *Lexer) parseExprPrec(minPrec int, on bool) (sqlstmt.ExprNode, bool) {
	if l.atCondBoundary(on) {
		return nil, false
	}
	left, ok := l.parsePrimary(on)
	if !ok {
		return nil, false
	}
	for !l.atCondBoundary(on) {
		op, prec, not, ok := l.peekInfixOp()
		if !ok || prec < minPrec {
			break
		}
		node, ok := l.parseInfix(left, op, prec, not, on)
		if !ok {
			return nil, false
		}
		left = node
	}
	return left, true
}

// peekInfixOp 识别当前位置的中缀运算符：符号类（TokenOperator）与关键字类
// （AND/OR/LIKE/IN/BETWEEN/IS，含 NOT IN / NOT LIKE / NOT BETWEEN 组合）
func (l *Lexer) peekInfixOp() (op string, prec int, not bool, ok bool) {
	tok := l.Current()
	if tok.Type == tokenizer.TokenOperator {
		op = strings.ToUpper(tok.Value)
		prec, exists := infixPrecedence[op]
		if !exists {
			return "", 0, false, false
		}
		return op, prec, false, true
	}
	if tok.Type != tokenizer.TokenKeyword {
		return "", 0, false, false
	}
	// NOT IN / NOT LIKE / NOT BETWEEN 组合
	if tok.IsKeyword("NOT") {
		next := l.Peek(1)
		if next.IsKeyword("IN", "LIKE", "BETWEEN") {
			return strings.ToUpper(next.Value), precComparison, true, true
		}
		return "", 0, false, false
	}
	upper := strings.ToUpper(tok.Value)
	if prec, exists := infixPrecedence[upper]; exists && tok.IsKeyword("AND", "OR", "LIKE", "IN", "BETWEEN", "IS") {
		return upper, prec, false, true
	}
	return "", 0, false, false
}

// parseInfix 消费运算符并解析右操作数，构造对应节点
func (l *Lexer) parseInfix(left sqlstmt.ExprNode, op string, prec int, not bool, on bool) (sqlstmt.ExprNode, bool) {
	startTok := l.Pos
	if not {
		l.Consume() // NOT（NOT IN/NOT LIKE/NOT BETWEEN组合：NOT已由peek定位）
	}

	switch op {
	case "IS":
		l.Consume() // IS
		isNot := false
		if l.Current().IsKeyword("NOT") {
			l.Consume()
			isNot = true
		}
		if !l.Current().IsKeyword("NULL") {
			return nil, false // IS后仅支持NULL（IS TRUE等罕见形态走降级）
		}
		l.Consume() // NULL（必须消费，否则主循环停在此处终止优先级爬升）
		node := &sqlstmt.IsNullExpr{Operand: left, Not: isNot}
		node.Raw = l.rawTextFrom(startTok)
		return node, true
	case "IN":
		l.Consume() // IN
		if l.Current().Value != "(" {
			return nil, false
		}
		l.Consume()
		node := &sqlstmt.InExpr{Operand: left, Not: not}
		if l.Current().IsKeyword("SELECT", "WITH") && l.SelectParser != nil {
			sub, ok := l.SelectParser()
			if !ok {
				return nil, false
			}
			node.Subquery = sub
		} else {
			for !l.Current().IsEOF() && l.Current().Value != ")" {
				if l.Current().Value == "," {
					l.Consume()
					continue
				}
				e, ok := l.parseExprPrec(0, false)
				if !ok {
					return nil, false
				}
				node.List = append(node.List, e)
			}
		}
		if l.Current().Value != ")" {
			return nil, false
		}
		l.Consume()
		node.Raw = l.rawTextFrom(startTok)
		return node, true
	case "BETWEEN":
		l.Consume() // BETWEEN
		low, ok := l.parseExprPrec(prec+1, on)
		if !ok || !l.Current().IsKeyword("AND") {
			return nil, false
		}
		l.Consume() // AND
		high, ok := l.parseExprPrec(prec+1, on)
		if !ok {
			return nil, false
		}
		node := &sqlstmt.BetweenExpr{Operand: left, Not: not, Low: low, High: high}
		node.Raw = l.rawTextFrom(startTok)
		return node, true
	case "LIKE":
		l.Consume() // LIKE
		pattern, ok := l.parseExprPrec(prec+1, on)
		if !ok {
			return nil, false
		}
		node := &sqlstmt.LikeExpr{Operand: left, Not: not, Pattern: pattern}
		if l.Current().IsKeyword("ESCAPE") {
			l.Consume()
			esc, ok := l.parsePrimary(on)
			if !ok {
				return nil, false
			}
			node.Escape = esc
		}
		node.Raw = l.rawTextFrom(startTok)
		return node, true
	}

	// 通用二元：符号类与AND/OR
	l.Consume() // 运算符
	// NOT（一元）出现在右操作数前：a = NOT b（罕见）不支持，交由parsePrimary处理普通形态
	right, ok := l.parseExprPrec(prec+1, on)
	if !ok {
		return nil, false
	}
	node := &sqlstmt.BinaryExpr{Op: op, Left: left, Right: right}
	node.Raw = l.rawTextFrom(startTok)
	return node, true
}

// rawTextFrom 截取 startTok 起至当前token结束的原始SQL文本（去首尾空白）。
// 各节点Raw统一经此生成，保证文本截取口径一致
func (l *Lexer) rawTextFrom(startTok int) string {
	return strings.TrimSpace(l.SQL[l.Tokens[startTok].Pos:l.Current().End])
}

// parsePrimary 解析初级表达式（前缀/原子）
func (l *Lexer) parsePrimary(on bool) (sqlstmt.ExprNode, bool) {
	startTok := l.Pos
	rawOf := func() string {
		return l.rawTextFrom(startTok)
	}
	tok := l.Current()

	switch {
	case tok.Type == tokenizer.TokenNumber || tok.Type == tokenizer.TokenString:
		l.Consume()
		return &sqlstmt.Literal{Value: tok.Value, NodeBase: sqlstmt.NodeBase{Raw: rawOf()}}, true

	case tok.Type == tokenizer.TokenKeyword:
		switch {
		case tok.IsKeyword("NULL", "TRUE", "FALSE"):
			l.Consume()
			return &sqlstmt.Literal{Value: strings.ToUpper(tok.Value), NodeBase: sqlstmt.NodeBase{Raw: rawOf()}}, true
		case tok.IsKeyword("NOT"):
			l.Consume()
			// NOT的绑定力介于AND(2)与比较(3)之间：操作数minPrec=比较级3，
			// 保证 NOT a=1 AND b=2 解析为 (NOT a=1) AND b=2（而非吞掉AND）
			operand, ok := l.parseExprPrec(precComparison, on)
			if !ok {
				return nil, false
			}
			return &sqlstmt.UnaryExpr{Op: "NOT", Operand: operand, NodeBase: sqlstmt.NodeBase{Raw: rawOf()}}, true
		case tok.IsKeyword("EXISTS"):
			l.Consume()
			if l.Current().Value != "(" {
				return nil, false
			}
			l.Consume()
			node := &sqlstmt.ExistsExpr{}
			if l.SelectParser != nil {
				sub, ok := l.SelectParser()
				if !ok {
					return nil, false
				}
				node.Subquery = sub
			} else {
				l.SkipParentheses()
			}
			if l.Current().Value != ")" {
				return nil, false
			}
			l.Consume()
			node.Raw = rawOf()
			return node, true
		case tok.IsKeyword("CASE"):
			return l.parseCaseExpr(on)
		}
		// 其余关键字形token：可能是非保留字列名（oracle/dm的left等），按列引用处理

	case tok.Value == "(":
		l.Consume()
		// 子查询
		if l.Current().IsKeyword("SELECT", "WITH") {
			if l.SelectParser == nil {
				l.SkipParentheses()
				return &sqlstmt.SubQueryExpr{NodeBase: sqlstmt.NodeBase{Raw: rawOf()}}, true
			}
			sub, ok := l.SelectParser()
			if !ok || l.Current().Value != ")" {
				return nil, false
			}
			l.Consume()
			return &sqlstmt.SubQueryExpr{Select: sub, NodeBase: sqlstmt.NodeBase{Raw: rawOf()}}, true
		}
		inner, ok := l.parseExprPrec(0, on)
		if !ok || l.Current().Value != ")" {
			return nil, false
		}
		l.Consume()
		if b, isBinary := inner.(*sqlstmt.BinaryExpr); isBinary && b.Op == "::" {
			return inner, true // pg :: 的右操作数是类型名，无需括号包装
		}
		return &sqlstmt.ParenExpr{Inner: inner, NodeBase: sqlstmt.NodeBase{Raw: rawOf()}}, true

	case tok.Value == "-" || tok.Value == "+":
		l.Consume()
		operand, ok := l.parseExprPrec(precUnarySign, on) // 一元符号绑定最紧
		if !ok {
			return nil, false
		}
		return &sqlstmt.UnaryExpr{Op: tok.Value, Operand: operand, NodeBase: sqlstmt.NodeBase{Raw: rawOf()}}, true
	}

	// 标识符：列引用或函数调用
	if tok.Type == tokenizer.TokenIdentifier || tok.Type == tokenizer.TokenKeyword {
		parts := []string{tok.Value}
		l.Consume()
		for l.Current().Value == "." {
			l.Consume()
			next := l.Current()
			if next.Type != tokenizer.TokenIdentifier && next.Type != tokenizer.TokenKeyword && next.Type != tokenizer.TokenString {
				return nil, false
			}
			parts = append(parts, next.Value)
			l.Consume()
		}
		if l.Current().Value == "(" {
			return l.parseFuncCall(parts, startTok)
		}
		return &sqlstmt.ColumnRef{Parts: parts, NodeBase: sqlstmt.NodeBase{Raw: rawOf()}}, true
	}

	return nil, false
}

// parseFuncCall 解析函数调用（当前token为"("），parts为函数名段
func (l *Lexer) parseFuncCall(nameParts []string, startTok int) (sqlstmt.ExprNode, bool) {
	l.Consume() // (
	node := &sqlstmt.FuncCall{
		Name: strings.ToUpper(nameParts[len(nameParts)-1]),
	}
	// COUNT(*) / COUNT(DISTINCT a)
	if l.Current().Value == "*" {
		l.Consume()
		node.Star = true
	} else {
		if l.Current().IsKeyword("DISTINCT") {
			l.Consume()
			node.Distinct = true
		}
		for !l.Current().IsEOF() && l.Current().Value != ")" {
			if l.Current().Value == "," {
				l.Consume()
				continue
			}
			e, ok := l.parseExprPrec(0, false)
			if !ok {
				return nil, false
			}
			node.Args = append(node.Args, e)
		}
	}
	if l.Current().Value != ")" {
		return nil, false
	}
	l.Consume()
	node.Raw = l.rawTextFrom(startTok)
	return node, true
}

// parseCaseExpr 解析CASE WHEN表达式（当前token为CASE）
func (l *Lexer) parseCaseExpr(on bool) (sqlstmt.ExprNode, bool) {
	startTok := l.Pos
	l.Consume() // CASE
	node := &sqlstmt.CaseExpr{}
	// 简单形态：CASE operand WHEN...
	if !l.Current().IsKeyword("WHEN") {
		operand, ok := l.parseExprPrec(0, on)
		if !ok {
			return nil, false
		}
		node.Operand = operand
	}
	for l.Current().IsKeyword("WHEN") {
		l.Consume()
		when, ok := l.parseExprPrec(0, on)
		if !ok || !l.Current().IsKeyword("THEN") {
			return nil, false
		}
		l.Consume()
		then, ok := l.parseExprPrec(0, on)
		if !ok {
			return nil, false
		}
		node.Whens = append(node.Whens, sqlstmt.CaseWhen{When: when, Then: then})
	}
	if len(node.Whens) == 0 {
		return nil, false
	}
	if l.Current().IsKeyword("ELSE") {
		l.Consume()
		els, ok := l.parseExprPrec(0, on)
		if !ok {
			return nil, false
		}
		node.ElseExpr = els
	}
	if !l.Current().IsKeyword("END") {
		return nil, false
	}
	l.Consume()
	node.Raw = l.rawTextFrom(startTok)
	return node, true
}
