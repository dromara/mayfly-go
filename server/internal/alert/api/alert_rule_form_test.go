package api

import (
	"encoding/json"
	"testing"

	"mayfly-go/internal/alert/domain/entity"
)

// unmarshalRuleForm 走与 gin 绑定同一套 encoding/json 语义还原请求体
func unmarshalRuleForm(t *testing.T, body string) *alertRuleForm {
	t.Helper()
	form := new(alertRuleForm)
	if err := json.Unmarshal([]byte(body), form); err != nil {
		t.Fatalf("unmarshal %s failed: %v", body, err)
	}
	return form
}

// TestAlertRuleFormPriority P0 的枚举值就是 0，与 Go 零值同形。
// 表单必须用指针承接 priority 才能区分「用户选了 P0」与「用户没选」：
// 前者要落 0，后者要回退领域默认值 P2，两者混为一谈会让最高优先级规则静默降级
func TestAlertRuleFormPriority(t *testing.T) {
	tests := []struct {
		name string
		body string
		want entity.AlertPriority
	}{
		{"显式 P0", `{"name":"r","priority":0}`, entity.AlertPriorityCritical},
		{"显式 P1", `{"name":"r","priority":1}`, entity.AlertPriorityHigh},
		{"显式 P2", `{"name":"r","priority":2}`, entity.AlertPriorityMedium},
		{"显式 P3", `{"name":"r","priority":3}`, entity.AlertPriorityLow},
		{"缺省回退 P2", `{"name":"r"}`, entity.AlertPriorityMedium},
		{"null 回退 P2", `{"name":"r","priority":null}`, entity.AlertPriorityMedium},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unmarshalRuleForm(t, tt.body).toEntity().Priority
			if got != tt.want {
				t.Fatalf("priority=%d(%s) want=%d(%s)", got, got.Code(), tt.want, tt.want.Code())
			}
		})
	}
}

// TestAlertRuleFormCarriesOtherFields priority 被本层字段遮蔽后，其余字段仍须由嵌入实体承接，
// 否则整表单保存会丢配置
func TestAlertRuleFormCarriesOtherFields(t *testing.T) {
	body := `{"name":"r","status":-1,"resourceType":1,"scopeType":2,"scopeValue":"[\"a/b/\"]",
		"evalInterval":30,"triggerCount":3,"recoveryCount":2,
		"condition":{"operator":"or","items":[{"metric":"mem_rate","compare":"gte","value":80.5,"duration":15}]},
		"notifyConfig":{"groupWait":5,"groupInterval":60,"repeatInterval":600}}`

	rule := unmarshalRuleForm(t, body).toEntity()
	if rule.Name != "r" || rule.Status != entity.AlertRuleStatusDisable {
		t.Fatalf("基础字段丢失: name=%q status=%d", rule.Name, rule.Status)
	}
	if rule.ResourceType != 1 || rule.ScopeType != entity.AlertScopeTagPath || rule.ScopeValue != `["a/b/"]` {
		t.Fatalf("范围字段丢失: rt=%d scopeType=%d scopeValue=%q", rule.ResourceType, rule.ScopeType, rule.ScopeValue)
	}
	if rule.EvalInterval != 30 || rule.TriggerCount != 3 || rule.RecoveryCount != 2 {
		t.Fatalf("高级配置丢失: eval=%d trigger=%d recovery=%d", rule.EvalInterval, rule.TriggerCount, rule.RecoveryCount)
	}
	if rule.Condition == nil || rule.Condition.Operator != entity.ConditionOperatorOr || len(rule.Condition.Items) != 1 {
		t.Fatalf("告警条件丢失: %+v", rule.Condition)
	}
	if item := rule.Condition.Items[0]; item.Metric != "mem_rate" || item.Compare != "gte" || item.Value != 80.5 || item.Duration != 15 {
		t.Fatalf("条件项丢失: %+v", item)
	}
	if rule.NotifyConfig == nil || rule.NotifyConfig.GroupWait != 5 || rule.NotifyConfig.GroupInterval != 60 || rule.NotifyConfig.RepeatInterval != 600 {
		t.Fatalf("通知配置丢失: %+v", rule.NotifyConfig)
	}
}

// TestAlertRuleFormNormalizesEmptyLabels labels 是 JSON 列，空串不是合法 JSON，
// 归一化必须发生在入口，否则落库直接报错
func TestAlertRuleFormNormalizesEmptyLabels(t *testing.T) {
	if got := unmarshalRuleForm(t, `{"name":"r"}`).toEntity().Labels; got != "{}" {
		t.Fatalf("缺省 labels 未归一: got=%q", got)
	}
	if got := unmarshalRuleForm(t, `{"name":"r","labels":""}`).toEntity().Labels; got != "{}" {
		t.Fatalf("空串 labels 未归一: got=%q", got)
	}
	if got := unmarshalRuleForm(t, `{"name":"r","labels":"{\"env\":\"prod\"}"}`).toEntity().Labels; got != `{"env":"prod"}` {
		t.Fatalf("非空 labels 被改动: got=%q", got)
	}
}
