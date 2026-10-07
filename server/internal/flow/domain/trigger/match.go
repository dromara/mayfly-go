package trigger

import (
	"mayfly-go/internal/flow/domain/entity"
)

// MatchCondition 判定一棵条件树在给定字段取值下是否成立，供流程连线跳转条件、
// 用户任务完成条件这类「布尔判定」复用触发引擎。
//
// 没有配置条件时返回成立：流程图里一条没写条件的连线就是默认流转路径，
// 用户任务没写完成条件时按「首个审批通过即完成」（或签）推进，这也是两类条件的通用默认语义。
//
// 求值失败必须把 error 原样交给调用方，不能折算成「不成立」：
// 跳转条件判错会让流程走错分支（可能跳过审批节点），完成条件判错会让任务卡死或提前完成，
// 两者都比「本次操作直接报错、让人来看」更糟
func MatchCondition(node *entity.RuleNode, tc *Context) (bool, error) {
	if !HasCondition(node) {
		return true, nil
	}
	matched, err := eval(node, tc)
	if err != nil {
		return false, err
	}
	return matched, nil
}

// HasCondition 表示条件树是否真的表达了某个判断。
//
// 前端构建器会留下「有分组壳子、里面一个条件都没有」的草稿，这与完全没配置等价，
// 必须按默认语义处理，否则一条默认流转的连线会因为壳子存在而判出不成立
func HasCondition(node *entity.RuleNode) bool {
	if node == nil {
		return false
	}
	switch node.Kind {
	case entity.NodeKindGroup:
		for _, item := range node.Items {
			if HasCondition(item) {
				return true
			}
		}
		return false
	case entity.NodeKindCondition:
		return node.Field != ""
	case entity.NodeKindSegment:
		return node.Ref != ""
	default:
		return true
	}
}
