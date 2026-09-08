package base

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// exprDialectConfig 表达式token提取的通用方言配置：
// 同时启用反引号与双引号标识符（覆盖mysql/pgsql/dm/oracle各方言的引用符形态），
// 单引号字符串字面量、注释、$tag$引用由tokenizer统一跳过
var exprDialectConfig = tokenizer.DialectConfig{
	BacktickAsIdentifier:    true,
	DoubleQuoteAsIdentifier: true,
	DollarQuote:             true,
}

// ExprQualifiedPair 表达式文本中的限定名对（如 u.phone、`t`.`col`、"u"."PHONE"）
type ExprQualifiedPair struct {
	Qualifier string // 表限定符（已去引用符并小写）
	Column    string // 列名（已去引用符，保留原始大小写）
}

// ExtractExprQualifiedPairs 提取表达式文本中的限定名对（qualifier.col）。
// 基于tokenizer词法分析：字符串字面量/注释自动跳过，关键字不识别为标识符；
// 连续的 标识符(. 标识符)+ 序列取最后两段（兼容 db.t.col 取 t.col 的语义，
// 与 ClassifySelectItem 的 TableAlias 规则一致）
func ExtractExprQualifiedPairs(expr string) []ExprQualifiedPair {
	l := NewLexer(expr, exprDialectConfig)
	pairs := make([]ExprQualifiedPair, 0)
	for i := 0; i < len(l.Tokens); i++ {
		if !isExprIdentToken(l.Tokens[i]) {
			continue
		}
		// 收集连续的 标识符(. 标识符)* 段序列，末两段为限定名对
		last, secondLast := l.Tokens[i], l.Tokens[i]
		j := i
		for j+2 < len(l.Tokens) && l.Tokens[j+1].Value == "." && isExprIdentToken(l.Tokens[j+2]) {
			secondLast, last = last, l.Tokens[j+2]
			j += 2
		}
		if j > i {
			pairs = append(pairs, ExprQualifiedPair{
				Qualifier: strings.ToLower(l.Unquote(secondLast.Value)),
				Column:    l.Unquote(last.Value),
			})
		}
		i = j
	}
	return pairs
}

// ExtractExprIdentifiers 提取表达式文本中的标识符token（跳过字符串字面量与注释，去引用符）。
// 关键字（SELECT/FROM/CASE等）不属于标识符，不会返回，避免关键字误命中名称匹配模式
func ExtractExprIdentifiers(expr string) []string {
	l := NewLexer(expr, exprDialectConfig)
	idents := make([]string, 0, len(l.Tokens))
	for _, tok := range l.Tokens {
		if isExprIdentToken(tok) {
			idents = append(idents, l.Unquote(tok.Value))
		}
	}
	return idents
}

// isExprIdentToken 判断token是否为标识符（不含关键字）
func isExprIdentToken(tok tokenizer.Token) bool {
	return tok.Type == tokenizer.TokenIdentifier
}
