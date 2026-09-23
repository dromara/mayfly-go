package sync

import (
	"fmt"
	"regexp"
	"strings"
)

// filterExpr 在任务开始时编译，行处理阶段只求值；解析错误不得变成全量放行或空保留集合。
type filterExpr struct {
	op       string
	field    string
	expected string
	children []*filterExpr
}

var filterLeafReg = regexp.MustCompile(`(?is)^([a-z_][a-z0-9_]*)\s*(==|!=|>=|<=|>|<|LIKE)\s*(.+)$`)

func parseFilterExpr(text string, depth int) (*filterExpr, error) {
	text = strings.TrimSpace(text)
	if text == "" || depth > 64 || len(text) > 16384 {
		return nil, fmt.Errorf("filter expression is empty or exceeds the size/depth limit")
	}
	// 先分解低优先级 OR，再处理 AND 和括号。
	for _, op := range []string{"OR", "AND"} {
		parts, err := splitFilterLogical(text, op)
		if err != nil {
			return nil, err
		}
		if len(parts) > 1 {
			node := &filterExpr{op: op}
			for _, part := range parts {
				child, err := parseFilterExpr(part, depth+1)
				if err != nil {
					return nil, err
				}
				node.children = append(node.children, child)
			}
			return node, nil
		}
	}
	if text[0] == '(' && text[len(text)-1] == ')' {
		return parseFilterExpr(text[1:len(text)-1], depth+1)
	}
	parts := filterLeafReg.FindStringSubmatch(text)
	if parts == nil {
		return nil, fmt.Errorf("invalid filter condition: %s", text)
	}
	expected := strings.TrimSpace(parts[3])
	if expected == "" {
		return nil, fmt.Errorf("filter comparison requires a value")
	}
	if expected[0] == '\'' || expected[0] == '"' {
		quote := expected[0]
		if len(expected) < 2 || expected[len(expected)-1] != quote {
			return nil, fmt.Errorf("unclosed filter string")
		}
		inner := expected[1 : len(expected)-1]
		var value strings.Builder
		for i := 0; i < len(inner); i++ {
			if inner[i] == quote {
				if i+1 >= len(inner) || inner[i+1] != quote {
					return nil, fmt.Errorf("filter string quotes must be doubled")
				}
				i++
			}
			value.WriteByte(inner[i])
		}
		expected = value.String()
	} else if strings.ContainsAny(expected, "()'\"<>=! \t\r\n") {
		return nil, fmt.Errorf("filter string values containing spaces or parentheses must be quoted")
	}
	return &filterExpr{field: parts[1], op: strings.ToUpper(parts[2]), expected: expected}, nil
}

func splitFilterLogical(text, op string) ([]string, error) {
	var parts []string
	depth, start := 0, 0
	var quote byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			if c == quote {
				if i+1 < len(text) && text[i+1] == quote {
					i++
				} else {
					quote = 0
				}
			}
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced filter parentheses")
			}
		}
		end := i + len(op)
		if depth == 0 && i > 0 && end < len(text) && filterSpace(text[i-1]) && filterSpace(text[end]) && strings.EqualFold(text[i:end], op) {
			parts = append(parts, text[start:i])
			start = end
			i = end - 1
		}
	}
	if quote != 0 || depth != 0 {
		return nil, fmt.Errorf("unbalanced filter quotes or parentheses")
	}
	return append(parts, text[start:]), nil
}

func filterSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func (e *filterExpr) validateFields(row map[string]any) error {
	if len(e.children) == 0 {
		if _, ok := lookupRowValue(row, e.field); !ok {
			return fmt.Errorf("filter field %q is absent from the source query", e.field)
		}
	}
	for _, child := range e.children {
		if err := child.validateFields(row); err != nil {
			return err
		}
	}
	return nil
}

func (e *filterExpr) match(f *FilterEngine, row map[string]any) bool {
	switch e.op {
	case "OR":
		for _, child := range e.children {
			if child.match(f, row) {
				return true
			}
		}
		return false
	case "AND":
		for _, child := range e.children {
			if !child.match(f, row) {
				return false
			}
		}
		return true
	default:
		actual, ok := lookupRowValue(row, e.field)
		return ok && f.evalCondition(fmt.Sprint(actual), e.op, e.expected)
	}
}
