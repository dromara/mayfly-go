package base

import (
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// ParseInsertValueRows 从当前 token（VALUES 关键字之后的第一个值组起始处）解析各行 VALUES 字面量，
// 并消费全部值组（解析位置等价于跳过整个 VALUES 段）。
//
// 基于词法单元而不是对原文逐字节扫描：字符串边界、转义、注释跳过等语义全部由该方言
// DialectConfig 产出的 token 流决定，本函数只做形如 「( 字面量, 字面量 ) , ...」的组结构识别。
//
// 返回值仅当每一格都是可确定求值的纯字面量（字符串/数字/NULL/布尔）时才非 nil；
// 函数、表达式、子查询、占位符、十六进制/位串等任何形态的最终写入值只有执行后才知道，
// 整体放弃而不是猜近似值——猜出的值会被写进回滚的定位条件，轻则命中 0 行，重则删错数据
func (l *Lexer) ParseInsertValueRows() []sqlstmt.InsertValueRow {
	rows := make([]sqlstmt.InsertValueRow, 0, 1)
	valid := true
	for l.Current().Value == "(" {
		l.Consume() // (
		row := make(sqlstmt.InsertValueRow, 0, 4)
		for {
			if l.Current().Value == ")" {
				// 空组（如 VALUES ()）：零格，交由消费方按列数判定
				l.Consume()
				break
			}
			if value, ok := l.parseInsertValueCell(); ok {
				row = append(row, value)
			} else {
				// 非确定字面量：整体放弃，仍继续消费完所有值组以保持解析终点位置
				valid = false
				l.skipInsertValueCell()
			}
			if l.Current().Value == "," {
				l.Consume()
				continue
			}
			if l.Current().Value == ")" {
				l.Consume()
				break
			}
			// 既非逗号也非右括号：形态异常（如中途 EOF）
			valid = false
			break
		}
		rows = append(rows, row)
		if l.Current().Value == "," {
			l.Consume()
			continue
		}
		break
	}
	if !valid {
		return nil
	}
	return rows
}

// parseInsertValueCell 解析一个单元格字面量并消费其 token；不是可确定形态时返回 ok=false 且不消费
func (l *Lexer) parseInsertValueCell() (sqlstmt.InsertValue, bool) {
	tok := l.Current()
	switch tok.Type {
	case tokenizer.TokenString:
		l.Consume()
		return sqlstmt.InsertValue{Text: tok.Value}, true
	case tokenizer.TokenNumber:
		if !isPlainNumberLiteral(tok.Value) {
			// 词法器对数字的取词较宽（如 1.2.3、1.），非法形状回放会生成语法错误的语句
			return sqlstmt.InsertValue{}, false
		}
		l.Consume()
		return sqlstmt.InsertValue{Text: tok.Value}, true
	case tokenizer.TokenKeyword:
		if tok.IsKeyword("NULL") {
			l.Consume()
			return sqlstmt.InsertValue{Text: "NULL", Null: true}, true
		}
		if tok.IsKeyword("TRUE", "FALSE") {
			l.Consume()
			return sqlstmt.InsertValue{Text: tok.Value}, true
		}
	case tokenizer.TokenOperator:
		// 带符号数字：词法器把 +/- 作独立运算符，与紧随的数字拼接为原文
		if tok.Value == "-" || tok.Value == "+" {
			next := l.Peek(1)
			if next.Type == tokenizer.TokenNumber && isPlainNumberLiteral(next.Value) {
				l.Consume()
				l.Consume()
				text := next.Value
				if tok.Value == "-" {
					text = "-" + text
				}
				return sqlstmt.InsertValue{Text: text}, true
			}
		}
	}
	return sqlstmt.InsertValue{}, false
}

// skipInsertValueCell 消费一个非字面量单元格：函数调用、表达式、子查询等都跳过到
// 本格边界（顶层逗号或右括号），嵌套括号整体跨过，保证 parseInsertValueCell 失败后解析位置仍正确
func (l *Lexer) skipInsertValueCell() {
	for !l.Current().IsEOF() {
		if l.Current().Value == "(" {
			l.SkipParentheses()
			continue
		}
		if l.Current().Value == "," || l.Current().Value == ")" {
			return
		}
		l.Consume()
	}
}

// isPlainNumberLiteral SQL 十进制数字原文的合法形状校验：整数[.小数][eE[+/-]指数]。
// 允许前导零（SQL 数字非 JSON 数字）；多小数点（1.2.3）、空小数（1.）、空指数（1e）等一律拒绝
func isPlainNumberLiteral(text string) bool {
	i := 0
	digits := 0
	for i < len(text) && text[i] >= '0' && text[i] <= '9' {
		i++
		digits++
	}
	if digits == 0 {
		return false
	}
	if i < len(text) && text[i] == '.' {
		i++
		frac := 0
		for i < len(text) && text[i] >= '0' && text[i] <= '9' {
			i++
			frac++
		}
		if frac == 0 {
			return false
		}
	}
	if i < len(text) && (text[i] == 'e' || text[i] == 'E') {
		i++
		if i < len(text) && (text[i] == '+' || text[i] == '-') {
			i++
		}
		exp := 0
		for i < len(text) && text[i] >= '0' && text[i] <= '9' {
			i++
			exp++
		}
		if exp == 0 {
			return false
		}
	}
	return i == len(text)
}
