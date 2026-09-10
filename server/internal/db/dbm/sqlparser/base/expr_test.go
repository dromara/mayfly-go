package base

// 表达式文本名称提取（脱敏血缘的列识别入口）的无环境依赖单测。
//
// 这两个函数只被 application/mask 的表达式列血缘调用，是数据安全的入口：
// 漏识别一个列引用，等于把该列明文返回给用户（fail-open）；
// 因此除了正向形态，本文件还钉死三类边界（字符串字面量/注释/$tag$内容不得成为列名）、
// 关键字形态列名必须可识别（MySQL 非保留字 comment/key/rows 等可作裸列名），
// 以及「标识符优先、关键字兜底」的返回顺序（关键字不得抢占真实列的匹配）。
//
// 运行：cd server && go test -count=1 ./internal/db/dbm/sqlparser/base/

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractExprQualifiedPairs(t *testing.T) {
	cases := []struct {
		name  string
		expr  string
		pairs []ExprQualifiedPair
	}{
		{"裸限定名", "u.phone", []ExprQualifiedPair{{"u", "phone"}}},
		{"反引号", "`t`.`col`", []ExprQualifiedPair{{"t", "col"}}},
		{"双引号", `"u"."PHONE"`, []ExprQualifiedPair{{"u", "PHONE"}}},
		// 库.表.列取末两段：脱敏归属看的是表（或表别名），库名不参与
		{"三段名取末两段", "db.t.col", []ExprQualifiedPair{{"t", "col"}}},
		{"函数包裹", "MAX(u.phone) + t.`key`", []ExprQualifiedPair{{"u", "phone"}, {"t", "key"}}},
		{"CASE表达式", "CASE WHEN a.b > 1 THEN 'x.y' END", []ExprQualifiedPair{{"a", "b"}}},
		// 字符串字面量内的点号绝不能被当成分隔符，否则值里的 'a.b' 会凭空造出一个血缘列
		{"字面量跳过", "CONCAT('a.b', x)", nil},
		{"字面量拼接", "u.phone||'-x'", []ExprQualifiedPair{{"u", "phone"}}},
		{"$tag$内容跳过", "$tag$u.phone$tag$", nil},
		{"行注释跳过", "-- c\nu.phone", []ExprQualifiedPair{{"u", "phone"}}},
		{"块注释跳过", "/* 'x.y' */ u.phone", []ExprQualifiedPair{{"u", "phone"}}},
		{"无限定名", "phone", nil},
		{"空表达式", "", nil},
		// 关键字形态列名（词法器按通用关键字表把它判成keyword）必须仍可识别为限定名，
		// 否则 SELECT CONCAT(t.comment,'x') 会绕过脱敏返回明文（本机真实MySQL脱敏IT已复现）
		{"关键字形态列名", "t.comment", []ExprQualifiedPair{{"t", "comment"}}},
		{"关键字列名排在标识符后", "u.phone + t.comment", []ExprQualifiedPair{{"u", "phone"}, {"t", "comment"}}},
	}
	for _, cc := range cases {
		cc := cc
		t.Run(cc.name, func(t *testing.T) {
			got := ExtractExprQualifiedPairs(cc.expr)
			if len(cc.pairs) == 0 {
				assert.Empty(t, got, "不应从%s中提取出限定名对", cc.expr)
				return
			}
			assert.Equal(t, cc.pairs, got)
		})
	}
}

func TestExtractExprIdentifiers(t *testing.T) {
	cases := []struct {
		name   string
		expr   string
		idents []string
	}{
		{"裸列名", "phone", []string{"phone"}},
		{"去引用符", "`t`.`col`", []string{"t", "col"}},
		{"函数名同为候选", "MAX(u.phone)", []string{"MAX", "u", "phone"}},
		// 字面量与注释内容不参与名称匹配：'13800001234' 之类的值不能被当作列名
		{"字面量跳过", "CONCAT('a.b', x)", []string{"CONCAT", "x"}},
		{"$tag$内容跳过", "$tag$u.phone$tag$", nil},
		{"注释跳过", "-- u.phone\nreal_col", []string{"real_col"}},
		{"块注释含字面量", "/* 'x.y' */ a", []string{"a"}},
		// 顺序即优先级：标识符全部排在前，关键字一律兜底在最后（关键字不得抢占真实列匹配）
		{"标识符优先于关键字", "CASE WHEN a = 1 THEN 'x' END", []string{"a", "CASE", "WHEN", "THEN", "END"}},
		{"关键字形态列名可识别", "t.default", []string{"t", "default"}},
		{"空表达式", "", nil},
	}
	for _, cc := range cases {
		cc := cc
		t.Run(cc.name, func(t *testing.T) {
			got := ExtractExprIdentifiers(cc.expr)
			if len(cc.idents) == 0 {
				assert.Empty(t, got, "不应从%s中提取出名称token", cc.expr)
				return
			}
			assert.Equal(t, cc.idents, got)
		})
	}
}
