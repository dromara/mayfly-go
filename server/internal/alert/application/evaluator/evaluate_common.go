package evaluator

import (
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
)

// 各评估器共用的判定原语。
//
// 放在这里而不是每个评估器里各写一份：AND/OR 组合语义、下标对齐与「不可判定」处理是告警引擎的
// 统一约定（见 service.EvalResult 的不变式注释）。多份实现必然随规则语义演进而漂移，
// 导致同一条件组合在不同资源类型上得出不同结论。

// isTriggered 按组合方式判定是否触发：
//   - or：任一条件满足即触发；无任一满足时不触发（结论是否可靠由 hasUnevaluableItem 另行标记）
//   - 其余（默认 and）：所有条件都满足才触发；条件为空视为不触发
func isTriggered(operator string, items []service.ConditionEval) bool {
	if operator == "or" {
		for _, item := range items {
			if item.Satisfied {
				return true
			}
		}
		return false
	}
	// 默认 and
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !item.Satisfied {
			return false
		}
	}
	return true
}

// hasUnevaluableItem 是否存在无法取值的条件项
func hasUnevaluableItem(items []service.ConditionEval) bool {
	for _, item := range items {
		if item.Err != "" {
			return true
		}
	}
	return false
}

// usesMetric 规则条件里是否用到了某个指标。
//
// 用途：数据源整体取不到时，只有「取不到也能给出结论」的存活类指标（机器在线状态、DB 可达性）
// 仍可继续评估，其余指标必须让本次判定标记为不可靠，不能把「没采到数据」当成「已恢复」
func usesMetric(condition *entity.AlertCondition, metric string) bool {
	if condition == nil {
		return false
	}
	for _, item := range condition.Items {
		if item.Metric == metric {
			return true
		}
	}
	return false
}
