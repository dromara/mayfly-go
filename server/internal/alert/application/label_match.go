package application

import (
	"encoding/json"

	labelapp "mayfly-go/internal/label/application"
	"mayfly-go/pkg/logx"
)

// parseRuleLabels 解析规则 Labels JSON 为 map。
// JSON 为空或解析失败时返回 nil（视为无标签）。
func parseRuleLabels(labelsJSON string) map[string]string {
	if labelsJSON == "" {
		return nil
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
		logx.Warnf("[alert] parse rule labels[%s] error: %s", labelsJSON, err.Error())
		return nil
	}
	return labels
}

// matchRuleLabelsMap 标签子集匹配：
// matchLabels 为空表示匹配所有；非空要求 ruleLabels 包含 matchLabels 的全部键值对。
func matchRuleLabelsMap(matchLabels map[string]string, ruleLabels map[string]string) bool {
	if len(matchLabels) == 0 {
		return true
	}
	if len(ruleLabels) == 0 {
		return false
	}
	for k, v := range matchLabels {
		if ruleLabels[k] != v {
			return false
		}
	}
	return true
}

// bindingsToMap 将 LabelBindingVO 列表转为 key→value map
func bindingsToMap(bindings []labelapp.LabelBindingVO) map[string]string {
	m := make(map[string]string, len(bindings))
	for _, b := range bindings {
		m[b.LabelKey] = b.LabelValue
	}
	return m
}
