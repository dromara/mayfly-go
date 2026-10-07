package trigger

import (
	"fmt"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	"regexp"
	"strings"
	"sync"
)

// OpDef 操作符定义。新增操作符只需注册一项，求值器、保存校验与前端构建器都无需改动
type OpDef struct {
	// Name 操作符标识
	Name entity.OpName
	// LabelKey 前端展示文案的 i18n key
	LabelKey string
	// Types 适用的字段类型，为空表示适用全部类型
	Types []FieldType
	// ValueKind 期望值形态，决定前端渲染哪种值编辑器
	ValueKind ValueKind
	// Match 比较实际取值与期望值
	Match func(actual, expected any) (bool, error)
	// ValidateValue 保存时的静态校验（如正则必须可编译），为空表示无需额外校验
	ValidateValue func(expected any) error
}

// appliesTo 判断操作符是否适用于指定字段类型
func (d OpDef) appliesTo(fieldType FieldType) bool {
	if len(d.Types) == 0 {
		return true
	}
	for _, support := range d.Types {
		if support == fieldType {
			return true
		}
	}
	return false
}

// 内置操作符标识
const (
	OpEq         entity.OpName = "eq"
	OpNe         entity.OpName = "ne"
	OpIn         entity.OpName = "in"
	OpNotIn      entity.OpName = "notIn"
	OpEmpty      entity.OpName = "empty"
	OpNotEmpty   entity.OpName = "notEmpty"
	OpContains   entity.OpName = "contains"
	OpNotContain entity.OpName = "notContains"
	OpStartsWith entity.OpName = "startsWith"
	OpRegex      entity.OpName = "regex"
	OpGt         entity.OpName = "gt"
	OpGte        entity.OpName = "gte"
	OpLt         entity.OpName = "lt"
	OpLte        entity.OpName = "lte"
	OpBetween    entity.OpName = "between"
	OpAnyOf      entity.OpName = "anyOf"
	OpAllOf      entity.OpName = "allOf"
	OpNoneOf     entity.OpName = "noneOf"
)

func init() {
	registerBuiltinTypes()
	registerBuiltinOps()
}

// registerBuiltinTypes 注册内置字段类型及其默认操作符集合
func registerBuiltinTypes() {
	RegisterType(FieldTypeDef{Name: TypeEnum, DefaultOps: []entity.OpName{OpEq, OpNe, OpIn, OpNotIn}, EditorKey: "select"})
	RegisterType(FieldTypeDef{Name: TypeString, DefaultOps: []entity.OpName{OpEq, OpNe, OpIn, OpNotIn, OpEmpty, OpNotEmpty, OpContains, OpNotContain, OpStartsWith, OpRegex}, EditorKey: "input"})
	RegisterType(FieldTypeDef{Name: TypeNumber, DefaultOps: []entity.OpName{OpEq, OpNe, OpGt, OpGte, OpLt, OpLte, OpBetween, OpEmpty, OpNotEmpty}, EditorKey: "number", Numeric: true})
	RegisterType(FieldTypeDef{Name: TypeBool, DefaultOps: []entity.OpName{OpEq, OpNe}, EditorKey: "switch"})
	RegisterType(FieldTypeDef{Name: TypeStringList, DefaultOps: []entity.OpName{OpAnyOf, OpAllOf, OpNoneOf, OpEmpty, OpNotEmpty}, EditorKey: "tags"})
	RegisterType(FieldTypeDef{Name: TypeNumberList, DefaultOps: []entity.OpName{OpAnyOf, OpAllOf, OpNoneOf, OpEmpty, OpNotEmpty}, EditorKey: "tags"})
	// TypeTime/TypeDuration 的比较值是数值（时间戳、秒数），校验与求值都按数字处理，
	// 所以编辑器如实声明为 number：声明成 datetime/duration 会让前端渲染成日期控件、
	// 提交回来却是「必须是数字」的校验失败
	RegisterType(FieldTypeDef{Name: TypeTime, DefaultOps: []entity.OpName{OpGt, OpGte, OpLt, OpLte, OpBetween}, EditorKey: "number", Numeric: true})
	RegisterType(FieldTypeDef{Name: TypeDuration, DefaultOps: []entity.OpName{OpGt, OpGte, OpLt, OpLte, OpBetween}, EditorKey: "number", Numeric: true})
}

// registerBuiltinOps 注册内置操作符。文本类比较统一忽略大小写：
// SQL 关键字与 Redis 命令名本身大小写不敏感，区分大小写会让「包含 truncate」这类规则静默漏判
func registerBuiltinOps() {
	ops := []OpDef{
		{Name: OpEq, LabelKey: "flow.op.eq", ValueKind: ValueSingle, Match: func(actual, expected any) (bool, error) { return looseEq(actual, expected), nil }},
		{Name: OpNe, LabelKey: "flow.op.ne", ValueKind: ValueSingle, Match: func(actual, expected any) (bool, error) { return !looseEq(actual, expected), nil }},
		{
			Name: OpIn, LabelKey: "flow.op.in", Types: []FieldType{TypeEnum, TypeString, TypeNumber, TypeBool}, ValueKind: ValueMulti,
			Match: func(actual, expected any) (bool, error) {
				candidates := toList(expected)
				if len(candidates) == 0 {
					return false, nil
				}
				for _, candidate := range candidates {
					if looseEq(actual, candidate) {
						return true, nil
					}
				}
				return false, nil
			},
		},
		{
			Name: OpNotIn, LabelKey: "flow.op.notIn", Types: []FieldType{TypeEnum, TypeString, TypeNumber, TypeBool}, ValueKind: ValueMulti,
			Match: func(actual, expected any) (bool, error) {
				candidates := toList(expected)
				if len(candidates) == 0 {
					return false, nil
				}
				for _, candidate := range candidates {
					if looseEq(actual, candidate) {
						return false, nil
					}
				}
				return true, nil
			},
		},
		{Name: OpEmpty, LabelKey: "flow.op.empty", ValueKind: ValueNone, Match: func(actual, _ any) (bool, error) { return isEmptyValue(actual), nil }},
		{Name: OpNotEmpty, LabelKey: "flow.op.notEmpty", ValueKind: ValueNone, Match: func(actual, _ any) (bool, error) { return !isEmptyValue(actual), nil }},
		{
			Name: OpContains, LabelKey: "flow.op.contains", Types: []FieldType{TypeString}, ValueKind: ValueSingle,
			Match: func(actual, expected any) (bool, error) {
				return strings.Contains(lowerText(asText(actual)), lowerText(asText(expected))), nil
			},
		},
		{
			Name: OpNotContain, LabelKey: "flow.op.notContains", Types: []FieldType{TypeString}, ValueKind: ValueSingle,
			Match: func(actual, expected any) (bool, error) {
				return !strings.Contains(lowerText(asText(actual)), lowerText(asText(expected))), nil
			},
		},
		{
			Name: OpStartsWith, LabelKey: "flow.op.startsWith", Types: []FieldType{TypeString}, ValueKind: ValueSingle,
			Match: func(actual, expected any) (bool, error) {
				return strings.HasPrefix(lowerText(asText(actual)), lowerText(asText(expected))), nil
			},
		},
		{
			Name: OpRegex, LabelKey: "flow.op.regex", Types: []FieldType{TypeString}, ValueKind: ValueSingle,
			ValidateValue: func(expected any) error {
				_, err := compileRegex(asText(expected))
				return err
			},
			Match: func(actual, expected any) (bool, error) {
				matched, err := compileRegex(asText(expected))
				if err != nil {
					return false, err
				}
				return matched.MatchString(lowerText(asText(actual))), nil
			},
		},
		numberOp(OpGt, "gt", func(left, right float64) bool { return left > right }),
		numberOp(OpGte, "gte", func(left, right float64) bool { return left >= right }),
		numberOp(OpLt, "lt", func(left, right float64) bool { return left < right }),
		numberOp(OpLte, "lte", func(left, right float64) bool { return left <= right }),
		{
			Name: OpBetween, LabelKey: "flow.op.between", Types: []FieldType{TypeNumber, TypeTime, TypeDuration}, ValueKind: ValueRange,
			ValidateValue: func(expected any) error {
				_, _, ok := splitRange(expected)
				if !ok {
					return fmt.Errorf("the range value requires both a lower and an upper bound")
				}
				return nil
			},
			Match: func(actual, expected any) (bool, error) {
				lowerValue, upperValue, ok := splitRange(expected)
				if !ok {
					return false, fmt.Errorf("the range value requires both a lower and an upper bound")
				}
				number, ok := toFloat(actual)
				if !ok {
					return false, nil
				}
				lower, lowerOk := toFloat(lowerValue)
				upper, upperOk := toFloat(upperValue)
				if !lowerOk || !upperOk {
					return false, nil
				}
				return number >= lower && number <= upper, nil
			},
		},
		{
			Name: OpAnyOf, LabelKey: "flow.op.anyOf", Types: []FieldType{TypeStringList, TypeNumberList, TypeString, TypeEnum}, ValueKind: ValueMulti,
			Match: func(actual, expected any) (bool, error) { return intersects(actual, expected), nil },
		},
		{
			Name: OpAllOf, LabelKey: "flow.op.allOf", Types: []FieldType{TypeStringList, TypeNumberList}, ValueKind: ValueMulti,
			Match: func(actual, expected any) (bool, error) {
				expectedItems := toList(expected)
				if len(expectedItems) == 0 {
					return false, nil
				}
				for _, want := range expectedItems {
					if !intersects(actual, []any{want}) {
						return false, nil
					}
				}
				return true, nil
			},
		},
		{
			Name: OpNoneOf, LabelKey: "flow.op.noneOf", Types: []FieldType{TypeStringList, TypeNumberList, TypeString, TypeEnum}, ValueKind: ValueMulti,
			Match: func(actual, expected any) (bool, error) { return !intersects(actual, expected), nil },
		},
	}

	for _, op := range ops {
		RegisterOp(op)
	}
}

// numberOp 构造一个数值比较操作符
func numberOp(name entity.OpName, labelKey string, compare func(left, right float64) bool) OpDef {
	return OpDef{
		Name:      name,
		LabelKey:  "flow.op." + labelKey,
		Types:     []FieldType{TypeNumber, TypeTime, TypeDuration},
		ValueKind: ValueSingle,
		Match: func(actual, expected any) (bool, error) {
			left, leftOk := toFloat(actual)
			right, rightOk := toFloat(expected)
			if !leftOk || !rightOk {
				return false, nil
			}
			return compare(left, right), nil
		},
	}
}

// intersects 判断实际取值（列表或标量）与期望列表是否存在交集
func intersects(actual, expected any) bool {
	actualItems := toList(actual)
	if len(actualItems) == 0 {
		return false
	}
	for _, want := range toList(expected) {
		for _, item := range actualItems {
			if looseEq(item, want) {
				return true
			}
		}
	}
	return false
}

// regexCache 复用已编译的正则：求值发生在业务执行路径上，不应每条语句重新编译
var regexCache sync.Map

func compileRegex(pattern string) (*regexp.Regexp, error) {
	if cached, ok := regexCache.Load(pattern); ok {
		matched, _ := cached.(*regexp.Regexp)
		return matched, nil
	}
	compiled, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		// 存量的坏正则在求值期才会暴露，同样给出可本地化的原因而不是裸 Go 文本
		return nil, invalid(pattern, imsg.ReasonRegexInvalid, "detail", err.Error())
	}
	regexCache.Store(pattern, compiled)
	return compiled, nil
}

func lowerText(text string) string {
	return strings.ToLower(text)
}
