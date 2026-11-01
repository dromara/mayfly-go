package base

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// ddlColumnConstraintWords 列定义里的「类型之后」修饰词：类型文本到此为止。
var ddlColumnConstraintWords = map[string]bool{
	"NOT": true, "NULL": true, "DEFAULT": true, "PRIMARY": true, "UNIQUE": true,
	"KEY": true, "COMMENT": true, "COLLATE": true, "REFERENCES": true, "CHECK": true,
	"GENERATED": true, "AS": true, "AUTO_INCREMENT": true, "IDENTITY": true,
	"CONSTRAINT": true, "ON": true, "STORAGE": true, "ENCODE": true,
	"DISTKEY": true, "SORTKEY": true, "CHARACTER": true, "CHARSET": true,
}

// ddlTableConstraintWords 表级约束项（整段不是列定义）：PRIMARY KEY(a,b) / UNIQUE(...) /
// KEY idx(...) / INDEX ... / CONSTRAINT ... / FOREIGN KEY ... / CHECK (...) / EXCLUDE ...
var ddlTableConstraintWords = map[string]bool{
	"PRIMARY": true, "UNIQUE": true, "KEY": true, "INDEX": true, "CONSTRAINT": true,
	"FOREIGN": true, "CHECK": true, "FULLTEXT": true, "SPATIAL": true, "EXCLUDE": true,
	"LIKE": true, "PARTITION": true,
}

// ddlColumnDefs 提取 CREATE TABLE 的列定义（名 / 类型原文 / 行内或表级主键 / 显式可空性 / DEFAULT 原文）。
//
// 设计取舍：只建「被消费」的字段，不做完整列定义文法（约束表达式、生成列规则等仍留在原文里）。
// 取不到 CREATE TABLE 的列清单（`CREATE TABLE ... AS SELECT`、`CREATE TABLE ... LIKE`、
// 解析异常）时返回 nil——不猜。
//
// 不改动 l.Pos：DDL 骨架随后还要 SkipToNextStatement，位置必须保持不动。
func (l *Lexer) ddlColumnDefs() []sqlstmt.ColumnDef {
	open := l.findTableColumnsOpen()
	if open < 0 {
		return nil
	}
	closeIdx := l.matchParen(open)
	if closeIdx < 0 {
		return nil
	}

	segments := l.splitTopLevel(open+1, closeIdx)
	if len(segments) == 0 {
		return nil
	}

	defs := make([]sqlstmt.ColumnDef, 0, len(segments))
	// 表级主键可能写在列定义之后（`PRIMARY KEY (a, b)`），先记下待回填
	pendingPK := make([]string, 0, 2)

	for _, seg := range segments {
		segStart, segEnd := seg[0], seg[1] // [start, end)
		if segStart >= segEnd {
			continue
		}
		first := strings.ToUpper(l.Tokens[segStart].Value)
		if ddlTableConstraintWords[first] {
			// PRIMARY KEY (a, b)：把括号里的列名记为待标记主键
			if first == "PRIMARY" && l.wordAt(segStart+1, "KEY") {
				pendingPK = append(pendingPK, l.namesInParens(segStart, segEnd)...)
			}
			if first == "CONSTRAINT" {
				// CONSTRAINT pk PRIMARY KEY (a, b)：约束名之后仍可能是主键
				for i := segStart + 1; i < segEnd; i++ {
					if strings.EqualFold(l.Tokens[i].Value, "PRIMARY") && l.wordAt(i+1, "KEY") {
						pendingPK = append(pendingPK, l.namesInParens(i, segEnd)...)
						break
					}
				}
			}
			continue
		}

		def := sqlstmt.ColumnDef{Name: l.Unquote(l.Tokens[segStart].Value)}
		typeEnd := segEnd
		for i := segStart + 1; i < segEnd; i++ {
			upper := strings.ToUpper(l.Tokens[i].Value)
			if ddlColumnConstraintWords[upper] {
				typeEnd = i
				break
			}
		}
		if typeEnd > segStart+1 {
			def.Type = l.rawText(segStart+1, typeEnd)
		}

		for i := segStart + 1; i < segEnd; i++ {
			switch strings.ToUpper(l.Tokens[i].Value) {
			case "PRIMARY":
				if l.wordAt(i+1, "KEY") {
					def.Primary = true
				}
			case "UNIQUE":
				def.Unique = true
			case "NOT":
				if l.wordAt(i+1, "NULL") {
					notNull := false
					def.Nullable = &notNull
				}
			case "NULL":
				// 排除 "NOT NULL" 里的 NULL（上面已处理）
				if i > segStart && strings.EqualFold(l.Tokens[i-1].Value, "NOT") {
					continue
				}
				nullable := true
				def.Nullable = &nullable
			case "DEFAULT":
				def.Default = l.defaultText(i+1, segEnd)
			}
		}
		if def.Name != "" {
			defs = append(defs, def)
		}
	}

	// 回填表级主键
	if len(pendingPK) > 0 {
		for i := range defs {
			for _, pk := range pendingPK {
				if strings.EqualFold(defs[i].Name, pk) {
					defs[i].Primary = true
				}
			}
		}
	}
	return defs
}

// findTableColumnsOpen 找到 CREATE TABLE 的列清单左括号下标；不是该形态返回 -1。
func (l *Lexer) findTableColumnsOpen() int {
	tableSeen := false
	for i := l.Pos; i < l.Length; i++ {
		tok := l.Tokens[i]
		upper := strings.ToUpper(tok.Value)
		if tok.Type == tokenizer.TokenOperator && tok.Value == ";" {
			return -1
		}
		switch upper {
		case "TABLE":
			tableSeen = true
		case "AS", "LIKE", "SELECT":
			// CREATE TABLE ... AS SELECT / LIKE：没有列清单
			if tableSeen {
				return -1
			}
		case "(":
			if tableSeen {
				return i
			}
		}
	}
	return -1
}

// matchParen 返回与 open 配对的右括号下标（-1 表示未闭合）。
func (l *Lexer) matchParen(open int) int {
	depth := 0
	for i := open; i < l.Length; i++ {
		switch l.Tokens[i].Value {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// splitTopLevel 按括号深度 0 的逗号切分 [from, to)，返回每段的 [start, end)。
func (l *Lexer) splitTopLevel(from, to int) [][2]int {
	var segs [][2]int
	start := from
	depth := 0
	for i := from; i < to; i++ {
		switch l.Tokens[i].Value {
		case "(":
			depth++
		case ")":
			depth--
		case ",":
			if depth == 0 {
				segs = append(segs, [2]int{start, i})
				start = i + 1
			}
		}
	}
	if start < to {
		segs = append(segs, [2]int{start, to})
	}
	return segs
}

// namesInParens 取 [from, to) 里第一个括号内的名字列表（PRIMARY KEY (a, b) → [a b]）。
func (l *Lexer) namesInParens(from, to int) []string {
	open := -1
	for i := from; i < to; i++ {
		if l.Tokens[i].Value == "(" {
			open = i
			break
		}
	}
	if open < 0 {
		return nil
	}
	closeIdx := l.matchParen(open)
	if closeIdx < 0 || closeIdx > to {
		closeIdx = to
	}
	names := make([]string, 0, 2)
	for _, seg := range l.splitTopLevel(open+1, closeIdx) {
		if seg[0] < seg[1] {
			names = append(names, l.Unquote(l.Tokens[seg[0]].Value))
		}
	}
	return names
}

// wordAt 判断绝对下标处是否为指定词（大小写不敏感，仅标识符/关键字）。
func (l *Lexer) wordAt(idx int, words ...string) bool {
	if idx < 0 || idx >= l.Length {
		return false
	}
	tok := l.Tokens[idx]
	if tok.Type != tokenizer.TokenKeyword && tok.Type != tokenizer.TokenIdentifier {
		return false
	}
	for _, w := range words {
		if strings.EqualFold(tok.Value, w) {
			return true
		}
	}
	return false
}

// rawText 返回 [from, to) 的原始 SQL 文本（去首尾空白）。
func (l *Lexer) rawText(from, to int) string {
	if from >= to || from >= l.Length {
		return ""
	}
	end := to - 1
	if end >= l.Length {
		end = l.Length - 1
	}
	startPos, endPos := l.Tokens[from].Pos, l.Tokens[end].End
	if startPos < 0 || endPos > len(l.SQL) || startPos >= endPos {
		return ""
	}
	return strings.TrimSpace(l.SQL[startPos:endPos])
}

// defaultText 取 DEFAULT 之后的原文，到下一个修饰词前（含引号与函数调用）。
func (l *Lexer) defaultText(from, to int) string {
	if from >= to {
		return ""
	}
	end := from
	depth := 0
	for end < to {
		tok := l.Tokens[end]
		if tok.Value == "(" {
			depth++
		} else if tok.Value == ")" {
			depth--
		} else if depth == 0 && ddlColumnConstraintWords[strings.ToUpper(tok.Value)] {
			break
		}
		end++
	}
	return l.rawText(from, end)
}
