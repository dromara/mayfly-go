package trigger

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
)

// MaxNodeDepth 条件树最大嵌套层数，防止误配置出难以阅读与评估的巨型条件
const MaxNodeDepth = 8

// MaxNodeCount 条件树最大节点数
const MaxNodeCount = 100

// MaxSegmentDepth 可复用条件组最多嵌套引用的层数。
// 循环引用已由引用路径挡住，这里再限一层深度，避免一条长链把一次判定拖成深递归
const MaxSegmentDepth = 8

// eval 递归求值条件节点。
//
// 返回 error 表示策略不可评估或字段无法判定（操作符未注册、类型不匹配、派生字段求值失败、
// 条件组缺失或循环引用），调用方须按 fail-closed 处置；只有「调用方确实没提供该字段」才按真值表判该叶子不成立
func eval(node *entity.RuleNode, tc *Context) (bool, error) {
	return evalNode(node, tc, nil)
}

// evalNode 的 refs 是本次判定路径上已经展开过的条件组标识：
// 同一条件组在不同分支各出现一次是正常的（如「DBA 且 工作时段」与「DBA 且 非工作时段」），
// 只有沿路径再次展开同一个标识才是循环引用，因此按路径判重而不是全局计数
func evalNode(node *entity.RuleNode, tc *Context, refs []string) (bool, error) {
	if node == nil {
		return false, nil
	}
	switch node.Kind {
	case entity.NodeKindGroup:
		return evalGroup(node, tc, refs)
	case entity.NodeKindCondition:
		return evalCondition(node, tc)
	case entity.NodeKindSegment:
		return evalSegment(node, tc, refs)
	default:
		return false, invalid("", imsg.ReasonUnknownKind, "kind", node.Kind)
	}
}

func evalGroup(node *entity.RuleNode, tc *Context, refs []string) (bool, error) {
	if len(node.Items) == 0 {
		// 空分组不表达任何约束，避免「全不」在空集上恒真而意外放行
		return false, nil
	}
	switch node.Logic {
	case entity.LogicAll, entity.LogicAny, entity.LogicNone:
	default:
		return false, invalid("", imsg.ReasonUnknownLogic, "logic", node.Logic)
	}

	matchedCount := 0
	for _, item := range node.Items {
		matched, err := evalNode(item, tc, refs)
		if err != nil {
			return false, err
		}
		if matched {
			matchedCount++
		}
	}

	switch node.Logic {
	case entity.LogicAll:
		return matchedCount == len(node.Items), nil
	case entity.LogicAny:
		return matchedCount > 0, nil
	default:
		return matchedCount == 0, nil
	}
}

// evalSegment 展开一个可复用条件组并求值其条件树。
//
// 取不到条件组必须报错而不是判不成立：条件组被删掉后，引用它的规则会静默失去约束，
// 这与「查不到黑名单就放行」是同一类 fail-open
func evalSegment(node *entity.RuleNode, tc *Context, refs []string) (bool, error) {
	if node.Ref == "" {
		return false, invalid("", imsg.ReasonSegmentRefRequired)
	}
	for _, ref := range refs {
		if ref == node.Ref {
			return false, invalid(node.Ref, imsg.ReasonSegmentCycle, "ref", node.Ref)
		}
	}
	if len(refs) >= MaxSegmentDepth {
		return false, invalid(node.Ref, imsg.ReasonSegmentTooDeep, "limit", MaxSegmentDepth)
	}

	resolve := tc.SegmentResolver()
	if resolve == nil {
		return false, invalid(node.Ref, imsg.ReasonSegmentNoResolver, "ref", node.Ref)
	}
	def, ok := resolve(node.Ref)
	if !ok || def == nil {
		return false, invalid(node.Ref, imsg.ReasonSegmentUnregistered, "ref", node.Ref)
	}
	if def.BizType != tc.BizType {
		return false, invalid(node.Ref, imsg.ReasonSegmentCrossBiz, "ref", node.Ref, "owner", def.BizType, "used", tc.BizType)
	}
	return evalNode(def.Condition, tc, append(refs, node.Ref))
}

func evalCondition(node *entity.RuleNode, tc *Context) (bool, error) {
	field, registered := FieldOf(tc.BizType, node.Field)
	if !registered {
		tc.markUnknown(node.Field)
		return false, nil
	}

	op, registered := OpOf(node.Op)
	if !registered {
		return false, invalid(node.Field, imsg.ReasonOperatorUnregistered, "op", node.Op)
	}
	if !op.appliesTo(field.Type) {
		return false, invalid(node.Field, imsg.ReasonOperatorMismatch, "op", node.Op, "field", node.Field, "type", field.Type)
	}

	actual, available, err := tc.FieldValue(field)
	if err != nil {
		return false, err
	}
	if !available {
		return false, nil
	}
	return op.Match(actual, node.Value)
}

// countNodes 统计条件树节点数并返回最大嵌套深度，供保存校验使用。
// segment 节点只计自身：被引用条件组的规模由它自己的保存校验负责
func countNodes(node *entity.RuleNode) (count int, depth int) {
	if node == nil {
		return 0, 0
	}
	count = 1
	childDepth := 0
	for _, item := range node.Items {
		itemCount, itemDepth := countNodes(item)
		count += itemCount
		if itemDepth > childDepth {
			childDepth = itemDepth
		}
	}
	return count, childDepth + 1
}
