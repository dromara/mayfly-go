package mask

import (
	"regexp"
	"sort"
	"strings"
)

// MatchType 规则匹配类型
type MatchType int8

const (
	MatchTypeRegex  MatchType = 1 // 正则匹配
	MatchTypeExact  MatchType = 2 // 精确匹配
	MatchTypePrefix MatchType = 3 // 前缀匹配
)

// TagAction 列标签动作
type TagAction int8

const (
	TagActionBind   TagAction = 1 // 绑定规则：对该列执行指定算法
	TagActionExempt TagAction = 2 // 豁免：该列不脱敏
)

// Rule 全局脱敏规则（已编译），由 application 层从实体转换而来
type Rule struct {
	Name      string
	MatchType MatchType
	Pattern   string
	Algorithm Algorithm
	Params    Params
}

// ColumnTag 列标签（已限定到实例维度），优先级高于全局规则
// 空的 DbName/TableName/ColumnName 视为通配
type ColumnTag struct {
	DbName     string
	TableName  string
	ColumnName string
	Action     TagAction
	Algorithm  Algorithm
	Params     Params
}

// compiledPrefix 编译后的前缀规则
type compiledPrefix struct {
	*Rule
}

// compiledRegex 编译后的正则规则
type compiledRegex struct {
	*Rule
	re *regexp.Regexp
}

// Plan 单个数据库实例维度的脱敏计划，构建时完成规则编译与优先级排序
// 优先级：列标签 > 全局规则；同级内按精确 > 前缀 > 正则
// 标签按特异性统一降序排列，同分时豁免优先（宁可少展示明文）
type Plan struct {
	tags        []ColumnTag      // 含绑定与豁免标签，按特异性降序排列
	exactRules  map[string]*Rule // key: 小写化后的列名模式
	prefixRules []compiledPrefix // 长度降序
	regexRules  []compiledRegex
}

// tagSpecificity 标签特异性得分，越大越具体
func tagSpecificity(tag ColumnTag) int {
	score := 0
	if tag.DbName != "" {
		score += 1
	}
	if tag.TableName != "" {
		score += 2
	}
	if tag.ColumnName != "" {
		score += 4
	}
	return score
}

// NewPlan 构建脱敏计划
func NewPlan(rules []*Rule, tags []*ColumnTag) (*Plan, error) {
	p := &Plan{exactRules: make(map[string]*Rule)}
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		switch rule.MatchType {
		case MatchTypeExact:
			if rule.Algorithm == nil {
				continue
			}
			// 小写化key实现大小写不敏感匹配（Oracle/PG结果列可能为大写），先声明者优先
			key := strings.ToLower(rule.Pattern)
			if _, ok := p.exactRules[key]; !ok {
				p.exactRules[key] = rule
			}
		case MatchTypePrefix:
			if rule.Algorithm == nil {
				continue
			}
			p.prefixRules = append(p.prefixRules, compiledPrefix{Rule: rule})
		default:
			// 未设置算法的规则不生效，直接跳过（包括未完成配置的规则，不阻塞其余规则生效）
			if rule.Algorithm == nil {
				continue
			}
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return nil, ErrInvalidPattern(rule.Pattern, err)
			}
			p.regexRules = append(p.regexRules, compiledRegex{Rule: rule, re: re})
		}
	}

	// 前缀按长度降序，长前缀优先命中
	sort.SliceStable(p.prefixRules, func(i, j int) bool {
		return len(p.prefixRules[i].Pattern) > len(p.prefixRules[j].Pattern)
	})

	for _, tag := range tags {
		if tag == nil {
			continue
		}
		// 豁免标签直接入列；绑定标签需已解析出算法，否则跳过
		if tag.Action != TagActionExempt && tag.Algorithm == nil {
			continue
		}
		p.tags = append(p.tags, *tag)
	}
	// 标签按特异性统一降序，同分时豁免优先：合并排序保证高特异性豁免能压过低特异性绑定
	sort.SliceStable(p.tags, func(i, j int) bool {
		si, sj := tagSpecificity(p.tags[i]), tagSpecificity(p.tags[j])
		if si != sj {
			return si > sj
		}
		return p.tags[i].Action == TagActionExempt && p.tags[j].Action != TagActionExempt
	})
	return p, nil
}

// matchTag 判断标签是否命中指定库表列
func matchTag(tag ColumnTag, db, table, column string) bool {
	if tag.DbName != "" && !strings.EqualFold(tag.DbName, db) {
		return false
	}
	if tag.TableName != "" && !strings.EqualFold(tag.TableName, table) {
		return false
	}
	if tag.ColumnName != "" && !strings.EqualFold(tag.ColumnName, column) {
		return false
	}
	return true
}

// Resolve 解析指定库表列应使用的脱敏算法，返回nil表示该列不脱敏
func (p *Plan) Resolve(db, table, column string) (Algorithm, Params) {
	if column == "" {
		return nil, nil
	}
	// 先匹配高特异性标签（含豁免，特异性高者先命中）
	for _, tag := range p.tags {
		if !matchTag(tag, db, table, column) {
			continue
		}
		if tag.Action == TagActionExempt {
			return nil, nil
		}
		return tag.Algorithm, tag.Params
	}

	// 全局规则：精确 > 前缀 > 正则
	if rule, ok := p.exactRules[strings.ToLower(column)]; ok {
		return rule.Algorithm, rule.Params
	}
	for _, rule := range p.prefixRules {
		if strings.HasPrefix(strings.ToLower(column), strings.ToLower(rule.Pattern)) {
			return rule.Algorithm, rule.Params
		}
	}
	for _, rule := range p.regexRules {
		if rule.re.MatchString(column) {
			return rule.Algorithm, rule.Params
		}
	}
	return nil, nil
}

// Empty 判断计划是否为空（无任何规则与标签），用于快速跳过
func (p *Plan) Empty() bool {
	return p == nil || (len(p.tags) == 0 && len(p.exactRules) == 0 && len(p.prefixRules) == 0 && len(p.regexRules) == 0)
}

// RowMasker 单次查询的行脱敏器，列级脱敏配置在构建时确定
type RowMasker struct {
	maskFns map[string]maskFn // key: 结果列名
}

type maskFn struct {
	alg    Algorithm
	params Params
}

// NewRowMasker 根据脱敏计划构建行脱敏器
//   - db: 当前库名
//   - tables: sql解析出的表名列表，可为空（表达式等场景）
//   - tableOf: 结果列名 -> 来源表名的精确映射（限定名场景如 t.col AS p）。键存在即锁定归属：
//     值为空串表示确定无来源表（如表达式无限定token），仅空表上下文解析（全局规则/通配标签）；
//     键不存在的列才按tables列表顺序尝试兑底
//   - resultColumns: 查询结果的列名列表
//   - srcColumnOf: 结果列名 -> 来源真实列名的映射（用于别名场景），未提供的列使用结果列名本身匹配
//
// 返回nil表示本次查询无需脱敏。构建时即标记命中列的Masked标识。
func NewRowMasker(plan *Plan, db string, tables []string, tableOf map[string]string, resultColumns []string, srcColumnOf map[string]string, markMasked func(colName string)) *RowMasker {
	if plan.Empty() {
		return nil
	}
	m := &RowMasker{maskFns: make(map[string]maskFn)}
	for _, col := range resultColumns {
		srcCol := col
		if mapped, ok := srcColumnOf[col]; ok && mapped != "" {
			srcCol = mapped
		}
		// 归属已锁定的列（tableOf键存在）仅用指定表解析，未命中时不回退到其他表，
		// 避免错误归属其他表的标签；值为空串表示确定无来源表，仅空表上下文（全局规则与表无关，不受影响）
		if table, locked := tableOf[col]; locked {
			alg, params := plan.Resolve(db, table, srcCol)
			if alg == nil {
				continue
			}
			m.maskFns[col] = maskFn{alg: alg, params: params}
			if markMasked != nil {
				markMasked(col)
			}
			continue
		}
		// 依次尝试各表上下文，未解析出表名时使用空表名（仅命中通配标签与全局规则）
		candidates := tables
		if len(candidates) == 0 {
			candidates = []string{""}
		}
		for _, table := range candidates {
			alg, params := plan.Resolve(db, table, srcCol)
			if alg == nil {
				continue
			}
			m.maskFns[col] = maskFn{alg: alg, params: params}
			if markMasked != nil {
				markMasked(col)
			}
			break
		}
	}
	if len(m.maskFns) == 0 {
		return nil
	}
	return m
}

// MaskRow 就地脱敏一行数据
func (m *RowMasker) MaskRow(row map[string]any) {
	for col, fn := range m.maskFns {
		if v, ok := row[col]; ok {
			row[col] = maskValue(fn.alg, fn.params, v)
		}
	}
}
