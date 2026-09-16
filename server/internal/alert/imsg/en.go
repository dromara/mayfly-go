package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogAlertRuleSave:               "Alert - Save Rule",
	LogAlertRuleDelete:             "Alert - Delete Rule",
	LogAlertRuleChangeStatus:       "Alert - Change Rule Status",
	LogAlertEventAck:               "Alert - Acknowledge Event",
	LogAlertEventClose:             "Alert - Close Event",
	LogAlertSilenceSave:            "Alert - Save Silence Rule",
	LogAlertSilenceDelete:          "Alert - Delete Silence Rule",
	LogAlertSilenceChangeStatus:    "Alert - Change Silence Rule Status",
	LogAlertEscalationSave:         "Alert - Save Escalation Policy",
	LogAlertEscalationDelete:       "Alert - Delete Escalation Policy",
	LogAlertEscalationChangeStatus: "Alert - Change Escalation Policy Status",

	ErrRuleNameRequired:                  "rule name is required",
	ErrRuleResourceTypeRequired:          "resource type is required",
	ErrRuleResourceTypeUnsupported:       "resource type [{{.type}}] has no available metric evaluator, cannot create alert rule",
	ErrRuleConditionRequired:             "at least one condition item is required",
	ErrRuleConditionMetricRequired:       "condition[{{.index}}] metric is required",
	ErrRuleConditionMetricUnknown:        "condition[{{.index}}] metric [{{.metric}}] is not supported by this resource type",
	ErrRuleConditionCompareRequired:      "condition[{{.index}}] compare operator is required",
	ErrRuleConditionCompareInvalid:       "condition[{{.index}}] compare operator [{{.compare}}] is invalid, expected one of gt/gte/lt/lte/eq/neq",
	ErrRuleConditionDurationInvalid:      "condition[{{.index}}] duration cannot be negative",
	ErrRuleOperatorInvalid:               "condition operator must be and or or",
	ErrRuleScopeTypeInvalid:              "invalid scope type",
	ErrRuleScopeValueRequired:            "scope value is required",
	ErrRuleScopeValueInvalid:             "scope value [{{.value}}] is invalid: a single resource requires a positive numeric id, a tag path requires a non-empty JSON array",
	ErrRuleEvalIntervalInvalid:           "evaluation interval must be between 10 and 86400 seconds",
	ErrRuleCountInvalid:                  "trigger count and recovery count must be between 1 and 1000",
	ErrRuleStatusInvalid:                 "rule status must be 1(enable) or -1(disable)",
	ErrRulePriorityInvalid:               "invalid priority, expected 0(P0) 1(P1) 2(P2) or 3(P3)",
	ErrRuleConditionValueInvalid:         "condition[{{.index}}] threshold [{{.value}}] is out of the valid range ({{.min}}~{{.max}}) for metric [{{.metric}}]",
	ErrEventNotFound:                     "alert event not found",
	ErrEventAckInvalidStatus:             "only a firing event can be acknowledged",
	ErrEventCloseInvalidStatus:           "the event is already closed",
	ErrRuleNotFound:                      "alert rule [{{.id}}] not found or already deleted",
	ErrSilenceNotFound:                   "silence rule [{{.id}}] not found or already deleted",
	ErrEscalationNotFound:                "escalation policy [{{.id}}] not found or already deleted",
	ErrSilenceNameRequired:               "silence name is required",
	ErrSilenceResourceTypeRequired:       "silence resource type is required",
	ErrSilenceTimeRequired:               "silence start time and end time are required",
	ErrSilenceTimeInvalid:                "silence end time must be later than start time",
	ErrSilenceMatchLabelsInvalid:         "match labels must be a valid JSON object, e.g. {\"env\":\"prod\"}",
	ErrEscalationNameRequired:            "escalation policy name is required",
	ErrEscalationResourceTypeRequired:    "escalation policy resource type is required",
	ErrEscalationRulesRequired:           "an escalation policy requires at least one escalation rule",
	ErrEscalationRuleDelayInvalid:        "escalation rule[{{.index}}] delay minutes must be greater than 0",
	ErrEscalationRuleReceiverRequired:    "escalation rule[{{.index}}] requires at least one channel or receiver, otherwise this level sends nothing",
	ErrSilenceResourceTypeUnsupported:    "silence resource type [{{.type}}] has no available metric evaluator, the silence rule takes effect on nothing",
	ErrEscalationResourceTypeUnsupported: "escalation resource type [{{.type}}] has no available metric evaluator, the escalation policy never matches",
	ErrEventDeleteActive:                 "only recovered or closed events can be deleted; please close active events first",
	ErrEscalationMatchLabelsInvalid:      "match labels must be a valid JSON object, e.g. {\"env\":\"prod\"}",
	LogAlertEventDelete:                  "Alert - Delete Event",
	LogAlertInhibitionSave:               "Alert - Save Inhibition Rule",
	LogAlertInhibitionDelete:             "Alert - Delete Inhibition Rule",
	LogAlertInhibitionChangeStatus:       "Alert - Change Inhibition Rule Status",

	// Notify Policy
	LogAlertNotifyPolicySave:          "Alert - Save Notify Policy",
	LogAlertNotifyPolicyDelete:        "Alert - Delete Notify Policy",
	LogAlertNotifyPolicyChangeStatus:  "Alert - Change Notify Policy Status",
	ErrNotifyPolicyNotFound:           "notify policy [{{.id}}] not found or already deleted",
	ErrNotifyPolicyNameRequired:       "notify policy name is required",
	ErrNotifyPolicyChannelRequired:    "notify policy requires at least one channel or receiver",
	ErrNotifyPolicyMatchLabelsInvalid: "notify policy match labels must be a valid JSON key-value object",

	// Metric names
	MetricCpuRate:   "CPU Usage",
	MetricMemRate:   "Memory Usage",
	MetricDiskUsage: "Disk Usage",
	MetricStatus:    "Online Status",

	// Inhibition rule
	ErrInhibitionMatchLabelsInvalid: "inhibition rule match labels must be a valid JSON key-value object",

	// Status validation
	ErrEscalationStatusInvalid:   "escalation policy status must be 1(enable) or -1(disable)",
	ErrInhibitionStatusInvalid:   "inhibition rule status must be 1(enable) or -1(disable)",
	ErrNotifyPolicyStatusInvalid: "notify policy status must be 1(enable) or -1(disable)",

	// Default notify templates
	TmplAlertNotifyName:   "Alert Notification Template",
	TmplAlertNotifyTitle:  "Alert - {{.ruleName}}",
	TmplAlertRecoverName:  "Alert Recovery Template",
	TmplAlertRecoverTitle: "Recovered - {{.ruleName}}",
}
