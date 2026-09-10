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

// exprNameKind 名称候选token的词性
type exprNameKind int8

const (
	nameNone    exprNameKind = iota // 非名称token（字面量/运算符/括号/数字等）
	nameIdent                       // 标识符（含加了引用符的关键字，如 `comment`）
	nameKeyword                     // 未加引用符的关键字（词法器无法判断其是否为列名，如 t.comment）
)

// exprNameKindOf 判断token的词性：字面量、注释、运算符、数字等均不是名称候选
func exprNameKindOf(tok tokenizer.Token) exprNameKind {
	switch tok.Type {
	case tokenizer.TokenIdentifier:
		return nameIdent
	case tokenizer.TokenKeyword:
		return nameKeyword
	}
	return nameNone
}

// ExtractExprQualifiedPairs 提取表达式文本中的限定名对（qualifier.col）。
// 基于tokenizer词法分析：字符串字面量/注释自动跳过；
// 连续的 名称(. 名称)+ 序列取最后两段（兼容 db.t.col 取 t.col 的语义，
// 与 ClassifySelectItem 的 TableAlias 规则一致）。
//
// 未加引用符的关键字形态列名（如 t.comment、t.key）同样作为候选对，但排在纯标识符对之后：
// 词法器按通用关键字表分类，而各方言的保留字表不同（comment/full/row 在mysql是非保留字，可直接作列名），
// 脱敏血缘属安全功能必须fail-closed——漏识别会把明文返回给用户；
// 而多识别一个候选不会产生误脱敏（调用方仍以表别名映射与脱敏计划过滤）。
// 保序是为了不改变已有行为：真实标识符的匹配优先级必须高于关键字兜底，避免关键字抢先命中导致错列归属
func ExtractExprQualifiedPairs(expr string) []ExprQualifiedPair {
	l := NewLexer(expr, exprDialectConfig)
	identPairs := make([]ExprQualifiedPair, 0)
	keywordPairs := make([]ExprQualifiedPair, 0)
	for i := 0; i < len(l.Tokens); i++ {
		kind := exprNameKindOf(l.Tokens[i])
		if kind == nameNone {
			continue
		}
		// 收集连续的 名称(. 名称)* 段序列，末两段为限定名对（并记住末两段的词性）
		secondLast, last := l.Tokens[i], l.Tokens[i]
		secondKind, lastKind := kind, kind
		j := i
		for j+2 < len(l.Tokens) && l.Tokens[j+1].Value == "." {
			nextKind := exprNameKindOf(l.Tokens[j+2])
			if nextKind == nameNone {
				break
			}
			secondLast, last = last, l.Tokens[j+2]
			secondKind, lastKind = lastKind, nextKind
			j += 2
		}
		if j > i {
			pair := ExprQualifiedPair{
				Qualifier: strings.ToLower(l.Unquote(secondLast.Value)),
				Column:    l.Unquote(last.Value),
			}
			if secondKind == nameIdent && lastKind == nameIdent {
				identPairs = append(identPairs, pair)
			} else {
				keywordPairs = append(keywordPairs, pair)
			}
		}
		i = j
	}
	return append(identPairs, keywordPairs...)
}

// ExtractExprIdentifiers 提取表达式文本中的名称token（跳过字符串字面量与注释，去引用符）。
// 关键字（SELECT/CASE/COMMENT等）**排在 identifiers 之后**返回：
// 未加引用符的关键字形态列名（comment/key/rows 等，mysql等非保留字）必须能被血缘识别到，
// 否则表达式内的脱敏列会以明文返回；而函数名、子句关键字等伪命中由调用方依据
// 查询内已知列与脱敏计划过滤，且排在最后不会抢占真实标识符的匹配
func ExtractExprIdentifiers(expr string) []string {
	l := NewLexer(expr, exprDialectConfig)
	idents := make([]string, 0, len(l.Tokens))
	keywords := make([]string, 0, 4)
	for _, tok := range l.Tokens {
		switch exprNameKindOf(tok) {
		case nameIdent:
			idents = append(idents, l.Unquote(tok.Value))
		case nameKeyword:
			keywords = append(keywords, l.Unquote(tok.Value))
		}
	}
	return append(idents, keywords...)
}
