package imsg

import (
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
)

func init() {
	i18n.AppendLangMsg(i18n.Zh_CN, Zh_CN)
	i18n.AppendLangMsg(i18n.En, En)
}

const (
	LogAlertRuleSave = iota + consts.ImsgNumAlert
	LogAlertRuleDelete
	LogAlertRuleChangeStatus
	LogAlertEventAck
	LogAlertEventClose
	LogAlertSilenceSave
	LogAlertSilenceDelete
	LogAlertSilenceChangeStatus
	LogAlertEscalationSave
	LogAlertEscalationDelete
	LogAlertEscalationChangeStatus

	// ---- 业务校验错误 ----
	ErrRuleNameRequired
	ErrRuleResourceTypeRequired
	ErrRuleResourceTypeUnsupported
	ErrRuleConditionRequired
	ErrRuleConditionMetricRequired
	ErrRuleConditionMetricUnknown
	ErrRuleConditionCompareRequired
	ErrRuleConditionCompareInvalid
	ErrRuleConditionDurationInvalid
	ErrRuleOperatorInvalid
	ErrRuleScopeTypeInvalid
	ErrRuleScopeValueRequired
	ErrRuleScopeValueInvalid
	ErrRuleEvalIntervalInvalid
	ErrRuleCountInvalid
	ErrRuleStatusInvalid
	ErrRulePriorityInvalid
	ErrRuleConditionValueInvalid
	ErrEventNotFound
	ErrEventAckInvalidStatus
	ErrEventCloseInvalidStatus
	ErrRuleNotFound
	ErrSilenceNotFound
	ErrEscalationNotFound
	ErrSilenceNameRequired
	ErrSilenceResourceTypeRequired
	ErrSilenceTimeRequired
	ErrSilenceTimeInvalid
	ErrSilenceMatchLabelsInvalid
	ErrEscalationNameRequired
	ErrEscalationResourceTypeRequired
	ErrEscalationRulesRequired
	ErrEscalationRuleDelayInvalid
	ErrEscalationRuleReceiverRequired

	// 静默/升级策略的资源类型同样需要评估器支撑：无评估器时不会产生任何告警，配置形同虚设，
	// 保存时必须拒绝。新增常量只能追加在末尾，避免已入库日志的 msgId 漂移
	ErrSilenceResourceTypeUnsupported
	ErrEscalationResourceTypeUnsupported
	ErrEventDeleteActive

	ErrEscalationMatchLabelsInvalid

	// ---- 通知策略 ----
	LogAlertNotifyPolicySave
	LogAlertNotifyPolicyDelete
	LogAlertNotifyPolicyChangeStatus

	ErrNotifyPolicyNotFound
	ErrNotifyPolicyNameRequired
	ErrNotifyPolicyChannelRequired
	ErrNotifyPolicyMatchLabelsInvalid

	// 新增日志常量追加在末尾，避免已入库日志的 msgId 漂移
	LogAlertEventDelete
	LogAlertInhibitionSave
	LogAlertInhibitionDelete
	LogAlertInhibitionChangeStatus

	// ---- 指标名称（用于通知模板） ----
	MetricCpuRate
	MetricMemRate
	MetricDiskUsage
	MetricStatus

	// ---- 抑制规则 ----
	ErrInhibitionMatchLabelsInvalid

	// ---- 升级/抑制/通知策略状态校验 ----
	ErrEscalationStatusInvalid
	ErrInhibitionStatusInvalid
	ErrNotifyPolicyStatusInvalid

	// ---- 默认通知模板 ----
	TmplAlertNotifyName
	TmplAlertNotifyTitle
	TmplAlertRecoverName
	TmplAlertRecoverTitle
)
