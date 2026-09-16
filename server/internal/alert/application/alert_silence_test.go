package application

import (
	"testing"
)

// ==================== matchRuleLabelsMap 测试 ====================

func TestMatchLabels_EmptyMatchLabels(t *testing.T) {
	// 空 MatchLabels 匹配该资源范围内的所有规则
	ruleLabels := map[string]string{"env": "prod", "team": "infra"}
	if !matchRuleLabelsMap(map[string]string{}, ruleLabels) {
		t.Error("empty matchLabels should match all rules")
	}
}

func TestMatchLabels_NilMatchLabels(t *testing.T) {
	// nil matchLabels 匹配所有
	ruleLabels := map[string]string{"env": "prod"}
	if !matchRuleLabelsMap(nil, ruleLabels) {
		t.Error("nil matchLabels should match all")
	}
}

func TestMatchLabels_ExactMatch(t *testing.T) {
	ruleLabels := map[string]string{"env": "prod", "team": "infra"}
	if !matchRuleLabelsMap(map[string]string{"env": "prod"}, ruleLabels) {
		t.Error("exact label match should return true")
	}
}

func TestMatchLabels_MultiKeyMatch(t *testing.T) {
	ruleLabels := map[string]string{"env": "prod", "team": "infra"}
	if !matchRuleLabelsMap(map[string]string{"env": "prod", "team": "infra"}, ruleLabels) {
		t.Error("multi-key exact match should return true")
	}
}

func TestMatchLabels_PartialMatch(t *testing.T) {
	// matchLabels 中的键是 ruleLabels 的子集 → 匹配
	ruleLabels := map[string]string{"env": "prod", "team": "infra", "region": "us"}
	if !matchRuleLabelsMap(map[string]string{"env": "prod", "team": "infra"}, ruleLabels) {
		t.Error("subset match should return true")
	}
}

func TestMatchLabels_ValueMismatch(t *testing.T) {
	ruleLabels := map[string]string{"env": "prod"}
	if matchRuleLabelsMap(map[string]string{"env": "staging"}, ruleLabels) {
		t.Error("value mismatch should return false")
	}
}

func TestMatchLabels_KeyNotFound(t *testing.T) {
	ruleLabels := map[string]string{"env": "prod"}
	if matchRuleLabelsMap(map[string]string{"team": "infra"}, ruleLabels) {
		t.Error("key not found should return false")
	}
}

func TestMatchLabels_NilRuleLabels(t *testing.T) {
	// ruleLabels 为空但 matchLabels 非空 → 不匹配
	if matchRuleLabelsMap(map[string]string{"env": "prod"}, nil) {
		t.Error("nil ruleLabels with non-empty matchLabels should return false")
	}
	if matchRuleLabelsMap(map[string]string{"env": "prod"}, map[string]string{}) {
		t.Error("empty ruleLabels with non-empty matchLabels should return false")
	}
}

func TestMatchLabels_BothEmpty(t *testing.T) {
	if !matchRuleLabelsMap(nil, nil) {
		t.Error("both empty should match")
	}
	if !matchRuleLabelsMap(map[string]string{}, map[string]string{}) {
		t.Error("both empty maps should match")
	}
}

// ==================== parseRuleLabels 测试 ====================

func TestParseLabels_ValidJSON(t *testing.T) {
	labels := parseRuleLabels(`{"env":"prod","team":"infra"}`)
	if labels["env"] != "prod" {
		t.Errorf("expected env=prod, got %s", labels["env"])
	}
	if labels["team"] != "infra" {
		t.Errorf("expected team=infra, got %s", labels["team"])
	}
}

func TestParseLabels_EmptyString(t *testing.T) {
	if labels := parseRuleLabels(""); labels != nil {
		t.Error("empty string should return nil")
	}
}

func TestParseLabels_InvalidJSON(t *testing.T) {
	if labels := parseRuleLabels("not json"); labels != nil {
		t.Error("invalid JSON should return nil")
	}
}

func TestParseLabels_EmptyObject(t *testing.T) {
	if labels := parseRuleLabels("{}"); len(labels) != 0 {
		t.Errorf("empty object should return empty map, got %v", labels)
	}
}
