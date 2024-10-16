package sqlstmt

import (
	"reflect"

	"github.com/antlr4-go/antlr/v4"
)

type INode interface {
	GetText() string

	// GetStartIndex() int
	// GetStopIndex() int
}

type Node struct {
	// startIndex int
	// stopIndex  int

	parser      antlr.Parser
	ruleContext antlr.RuleContext
}

func (n *Node) GetText() string {
	if n == nil || n.ruleContext == nil || n.parser == nil {
		return ""
	}
	return n.parser.GetTokenStream().GetTextFromRuleContext(n.ruleContext)
}

// func (n *Node) GetStartIndex() int {
// 	return n.startIndex
// }

// func (n *Node) GetStopIndex() int {
// 	return n.stopIndex
// }

func NewNode(parser antlr.Parser, ruleContext antlr.RuleContext) *Node {
	return &Node{
		parser:      parser,
		ruleContext: ruleContext,
	}
}

// func NewNode(startIndex, stopIndex int) *Node {
// 	return &Node{
// 		startIndex: startIndex,
// 		stopIndex:  stopIndex,
// 	}
// }

// func GetText(sqlstmts string, node INode) string {
// 	return sqlstmts[node.GetStartIndex() : node.GetStopIndex()+1]
// }

type Stmt interface {
	INode
}

type SqlStmt struct {
	*Node
}

type DmlStmt struct {
	SqlStmt
}

type OtherReadStmt struct {
	*Node
}

func IsSelectStmt(stmt Stmt) bool {
	return reflect.TypeOf(stmt).AssignableTo(reflect.TypeOf(&SelectStmt{}))
}
