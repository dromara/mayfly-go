package trigger

import (
	"context"
	"testing"

	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
)

// segmentBizType 专用于可复用条件组测试的场景，字段与 testBizType 一致但不共用注册，
// 保证「条件组属于哪个字段字典」这类判定是真的按场景比对，而不是靠巧合通过
const segmentBizType = "test_segment_flow"

func init() {
	RegisterBiz(BizMeta{
		BizType: segmentBizType,
		Fields: []TriggerField{
			{Key: "stmtType", TitleKey: "t", Group: "g", Type: TypeEnum, Options: []FieldOption{{Value: "update"}, {Value: "delete"}}},
			{Key: "tableCount", TitleKey: "t", Group: "g", Type: TypeNumber},
		},
	})
}

func segmentContext(attributes map[string]any, resolver SegmentResolver) *Context {
	tc := NewContext(context.Background(), segmentBizType, 0, nil, nil).WithAttributes(attributes)
	if resolver != nil {
		tc.WithSegments(resolver)
	}
	return tc
}

func segmentNode(ref string) *entity.RuleNode {
	return &entity.RuleNode{Kind: entity.NodeKindSegment, Ref: ref}
}

// resolverOf 用内存表驱动条件组判定，避免为测试拉起持久层
func resolverOf(segments map[string]*SegmentDefinition) SegmentResolver {
	return func(ref string) (*SegmentDefinition, bool) {
		def, ok := segments[ref]
		return def, ok
	}
}

func TestMatchConditionTreatsBlankTreeAsNoConstraint(t *testing.T) {
	cases := []struct {
		name string
		node *entity.RuleNode
	}{
		{"未配置", nil},
		{"空分组", group(entity.LogicAll)},
		{"只有壳子的嵌套分组", group(entity.LogicAny, group(entity.LogicAll))},
		{"没选字段的条件行", &entity.RuleNode{Kind: entity.NodeKindCondition}},
	}
	for _, tc := range cases {
		matched, err := MatchCondition(tc.node, segmentContext(map[string]any{"stmtType": "update"}, nil))
		if err != nil {
			t.Fatalf("%s: a blank condition must not fail, got %v", tc.name, err)
		}
		if !matched {
			t.Fatalf("%s: a blank condition must read as satisfied (default flow), got false", tc.name)
		}
	}
}

func TestMatchConditionEvaluatesRealConditions(t *testing.T) {
	tc := segmentContext(map[string]any{"stmtType": "update", "tableCount": 3}, nil)
	if matched, err := MatchCondition(condition("stmtType", OpEq, "update"), tc); err != nil || !matched {
		t.Fatalf("the condition must match, got %v %v", matched, err)
	}
	if matched, err := MatchCondition(condition("stmtType", OpEq, "delete"), tc); err != nil || matched {
		t.Fatalf("a different value must not match, got %v %v", matched, err)
	}
	// 未提供的字段按不成立处理，但不算求值失败：流程变量本来就可能还没有该字段
	if matched, err := MatchCondition(condition("ghostField", OpEq, "x"), tc); err != nil || matched {
		t.Fatalf("an unregistered field must read as not matched, got %v %v", matched, err)
	}
	// 操作符未注册属于「策略无法评估」，必须冒错让调用方中断，而不是判不成立后走默认分支
	if _, err := MatchCondition(condition("stmtType", "ghostOp", "update"), tc); err == nil {
		t.Fatalf("an unregistered operator must surface as an error")
	}
}

func TestSegmentResolvesIntoTheSameTree(t *testing.T) {
	segments := resolverOf(map[string]*SegmentDefinition{
		"dba": {Ref: "dba", BizType: segmentBizType, Condition: condition("stmtType", OpEq, "update")},
	})

	matched, err := eval(segmentNode("dba"), segmentContext(map[string]any{"stmtType": "update"}, segments))
	if err != nil || !matched {
		t.Fatalf("the referenced group must be evaluated, got %v %v", matched, err)
	}

	// 条件组与外层条件混在同一分组里，语义必须和普通子节点一致
	mixed := group(entity.LogicAll, segmentNode("dba"), condition("tableCount", OpGte, 2))
	if matched, err := eval(mixed, segmentContext(map[string]any{"stmtType": "update", "tableCount": 1}, segments)); err != nil || matched {
		t.Fatalf("every child of an all-group must hold, got %v %v", matched, err)
	}
	if matched, err := eval(mixed, segmentContext(map[string]any{"stmtType": "update", "tableCount": 5}, segments)); err != nil || !matched {
		t.Fatalf("the group must match when both children hold, got %v %v", matched, err)
	}
}

// 引用的条件组不存在时必须报错：判不成立会让「删除条件组」变成一次静默放宽规则
func TestSegmentErrorsWhenUnresolvable(t *testing.T) {
	missing := resolverOf(map[string]*SegmentDefinition{})
	if _, err := eval(segmentNode("gone"), segmentContext(nil, missing)); err == nil {
		t.Fatalf("a missing condition group must surface as an error")
	}

	// 没有 resolver（精简部署、单测）同样属于「无法判定」
	if _, err := eval(segmentNode("dba"), segmentContext(nil, nil)); err == nil {
		t.Fatalf("a condition group cannot be decided without a resolver")
	}

	// 空引用是配置错误，不能当成「这一项没填」跳过
	if _, err := eval(&entity.RuleNode{Kind: entity.NodeKindSegment}, segmentContext(nil, missing)); err == nil {
		t.Fatalf("a segment node without a ref must be rejected")
	}
}

// 条件组之间可以互相引用，但成环必须挡住：
// 环会让求值无限递归，而运行期一次审批操作把进程拖垮比配置报错严重得多
func TestSegmentCycleIsRejected(t *testing.T) {
	a := &SegmentDefinition{Ref: "a", BizType: segmentBizType}
	b := &SegmentDefinition{Ref: "b", BizType: segmentBizType}
	a.Condition = group(entity.LogicAny, segmentNode("b"))
	b.Condition = group(entity.LogicAny, segmentNode("a"))
	cyclic := resolverOf(map[string]*SegmentDefinition{"a": a, "b": b})

	_, err := eval(segmentNode("a"), segmentContext(nil, cyclic))
	if err == nil {
		t.Fatalf("a reference cycle must be reported instead of recursing forever")
	}
	if policyErr := AsError(err); policyErr == nil || policyErr.MsgId != imsg.ReasonSegmentCycle {
		t.Fatalf("the error must be classified as a condition group cycle, got %v", err)
	}

	// 同一条件组在不同分支各出现一次是正常用法，不能被误判成环
	shared := resolverOf(map[string]*SegmentDefinition{
		"c": {Ref: "c", BizType: segmentBizType, Condition: condition("stmtType", OpEq, "update")},
	})
	tree := group(entity.LogicAny, segmentNode("c"), group(entity.LogicAll, segmentNode("c"), condition("tableCount", OpGte, 1)))
	if _, err := eval(tree, segmentContext(map[string]any{"stmtType": "update", "tableCount": 0}, shared)); err != nil {
		t.Fatalf("reusing a group in sibling branches is not a cycle, got %v", err)
	}
}

// 条件组只能在同场景的规则里引用：跨场景引用时字段名同名却不同义，
// 求值会拿 A 场景的取值去比 B 场景的条件
func TestSegmentRejectsForeignBizType(t *testing.T) {
	foreign := resolverOf(map[string]*SegmentDefinition{
		"dba": {Ref: "dba", BizType: testBizType, Condition: condition("stmtType", OpEq, "update")},
	})
	if _, err := eval(segmentNode("dba"), segmentContext(map[string]any{"stmtType": "update"}, foreign)); err == nil {
		t.Fatalf("referencing a group from another field dictionary must fail")
	}
}

func TestValidateConditionRejectsUnusableTrees(t *testing.T) {
	if err := ValidateCondition(segmentBizType, nil); err != nil {
		t.Fatalf("an absent condition is valid (it means the default flow), got %v", err)
	}
	if err := ValidateCondition(segmentBizType, condition("ghost", OpEq, "x")); err == nil {
		t.Fatalf("an unregistered field must be rejected at save time")
	}
	if err := ValidateCondition(segmentBizType, group(entity.LogicAll)); err == nil {
		t.Fatalf("an empty group must be rejected at save time")
	}

	withResolver := WithSegmentResolver(resolverOf(map[string]*SegmentDefinition{
		"dba": {Ref: "dba", BizType: segmentBizType, Condition: condition("stmtType", OpEq, "update")},
	}))
	if err := ValidateCondition(segmentBizType, segmentNode("dba"), withResolver); err != nil {
		t.Fatalf("a resolvable group must pass, got %v", err)
	}
	if err := ValidateCondition(segmentBizType, segmentNode("ghost"), withResolver); err == nil {
		t.Fatalf("a dangling group reference must be rejected before saving")
	}
	// 没有 resolver 时不能默认放行：保存下来的引用会在运行期变成一堆「无法判定」
	if err := ValidateCondition(segmentBizType, segmentNode("dba")); err == nil {
		t.Fatalf("a group reference cannot be validated without a resolver")
	}

	cyclicA := &SegmentDefinition{Ref: "a", BizType: segmentBizType}
	cyclicB := &SegmentDefinition{Ref: "b", BizType: segmentBizType}
	cyclicA.Condition = segmentNode("b")
	cyclicB.Condition = segmentNode("a")
	if err := ValidateCondition(segmentBizType, segmentNode("a"), WithSegmentResolver(resolverOf(map[string]*SegmentDefinition{"a": cyclicA, "b": cyclicB}))); err == nil {
		t.Fatalf("a reference cycle must be rejected at save time")
	}
}
