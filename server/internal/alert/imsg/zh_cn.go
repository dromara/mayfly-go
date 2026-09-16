package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogAlertRuleSave:               "告警-保存规则",
	LogAlertRuleDelete:             "告警-删除规则",
	LogAlertRuleChangeStatus:       "告警-调整规则状态",
	LogAlertEventAck:               "告警-确认事件",
	LogAlertEventClose:             "告警-关闭事件",
	LogAlertSilenceSave:            "告警-保存静默规则",
	LogAlertSilenceDelete:          "告警-删除静默规则",
	LogAlertSilenceChangeStatus:    "告警-调整静默规则状态",
	LogAlertEscalationSave:         "告警-保存升级策略",
	LogAlertEscalationDelete:       "告警-删除升级策略",
	LogAlertEscalationChangeStatus: "告警-调整升级策略状态",

	ErrRuleNameRequired:                  "规则名称不能为空",
	ErrRuleResourceTypeRequired:          "资源类型不能为空",
	ErrRuleResourceTypeUnsupported:       "资源类型[{{.type}}]暂无可用的指标评估器，无法创建告警规则",
	ErrRuleConditionRequired:             "至少需要配置一个告警条件",
	ErrRuleConditionMetricRequired:       "条件[{{.index}}]的指标不能为空",
	ErrRuleConditionMetricUnknown:        "条件[{{.index}}]的指标[{{.metric}}]不被该资源类型支持",
	ErrRuleConditionCompareRequired:      "条件[{{.index}}]的比较方式不能为空",
	ErrRuleConditionCompareInvalid:       "条件[{{.index}}]的比较方式[{{.compare}}]非法，可选：gt/gte/lt/lte/eq/neq",
	ErrRuleConditionDurationInvalid:      "条件[{{.index}}]的持续时间不能为负数",
	ErrRuleOperatorInvalid:               "条件组合方式必须为 and 或 or",
	ErrRuleScopeTypeInvalid:              "关联范围类型非法",
	ErrRuleScopeValueRequired:            "关联范围值不能为空",
	ErrRuleScopeValueInvalid:             "关联范围值[{{.value}}]非法：指定资源需为正整数ID，标签路径需为非空JSON数组",
	ErrRuleEvalIntervalInvalid:           "评估间隔需在 10~86400 秒之间",
	ErrRuleCountInvalid:                  "触发次数与恢复次数需在 1~1000 之间",
	ErrRuleStatusInvalid:                 "规则状态必须为 1(启用) 或 -1(禁用)",
	ErrRulePriorityInvalid:               "优先级非法，可选：0(P0紧急) 1(P1重要) 2(P2一般) 3(P3提示)",
	ErrRuleConditionValueInvalid:         "条件[{{.index}}]的阈值[{{.value}}]超出指标[{{.metric}}]的合法取值范围（{{.min}}~{{.max}}）",
	ErrEventNotFound:                     "告警事件不存在",
	ErrEventAckInvalidStatus:             "仅「告警中」的事件可确认，当前事件不可确认",
	ErrEventCloseInvalidStatus:           "该事件已关闭，无需重复关闭",
	ErrRuleNotFound:                      "告警规则[{{.id}}]不存在或已被删除",
	ErrSilenceNotFound:                   "静默规则[{{.id}}]不存在或已被删除",
	ErrEscalationNotFound:                "升级策略[{{.id}}]不存在或已被删除",
	ErrSilenceNameRequired:               "静默名称不能为空",
	ErrSilenceResourceTypeRequired:       "静默资源类型不能为空",
	ErrSilenceTimeRequired:               "静默生效开始与结束时间不能为空",
	ErrSilenceTimeInvalid:                "静默结束时间必须晚于开始时间",
	ErrSilenceMatchLabelsInvalid:         "匹配标签必须为合法的 JSON 对象，如 {\"env\":\"prod\"}",
	ErrEscalationNameRequired:            "升级策略名称不能为空",
	ErrEscalationResourceTypeRequired:    "升级策略资源类型不能为空",
	ErrEscalationRulesRequired:           "升级策略至少需要配置一级升级规则",
	ErrEscalationRuleDelayInvalid:        "升级规则[{{.index}}]的延迟分钟数必须大于 0",
	ErrEscalationRuleReceiverRequired:    "升级规则[{{.index}}]需至少指定一个通知渠道或接收人，否则该级别不会产生任何通知",
	ErrSilenceResourceTypeUnsupported:    "资源类型[{{.type}}]暂无可用的指标评估器，静默规则不会生效",
	ErrEscalationResourceTypeUnsupported: "资源类型[{{.type}}]暂无可用的指标评估器，升级策略不会生效",
	ErrEventDeleteActive:                 "仅可删除已恢复或已关闭的事件，活跃事件请先关闭后再删除",
	ErrEscalationMatchLabelsInvalid:      "匹配标签必须为合法的 JSON 对象，如 {\"env\":\"prod\"}",
	LogAlertEventDelete:                  "告警 - 删除事件",
	LogAlertInhibitionSave:               "告警 - 保存抑制规则",
	LogAlertInhibitionDelete:             "告警 - 删除抑制规则",
	LogAlertInhibitionChangeStatus:       "告警 - 调整抑制规则状态",

	// 通知策略
	LogAlertNotifyPolicySave:          "告警 - 保存通知策略",
	LogAlertNotifyPolicyDelete:        "告警 - 删除通知策略",
	LogAlertNotifyPolicyChangeStatus:  "告警 - 调整通知策略状态",
	ErrNotifyPolicyNotFound:           "通知策略[{{.id}}]不存在或已被删除",
	ErrNotifyPolicyNameRequired:       "通知策略名称不能为空",
	ErrNotifyPolicyChannelRequired:    "通知策略需至少指定一个通知渠道或接收人",
	ErrNotifyPolicyMatchLabelsInvalid: "通知策略关联标签格式错误，需为合法的 JSON key-value 对象",

	// 指标名称
	MetricCpuRate:   "CPU 使用率",
	MetricMemRate:   "内存使用率",
	MetricDiskUsage: "磁盘使用率",
	MetricStatus:    "在线状态",

	// 抑制规则
	ErrInhibitionMatchLabelsInvalid: "抑制规则匹配标签格式错误，需为合法的 JSON key-value 对象",

	// 状态校验
	ErrEscalationStatusInvalid:   "升级策略状态必须为 1(启用) 或 -1(禁用)",
	ErrInhibitionStatusInvalid:   "抑制规则状态必须为 1(启用) 或 -1(禁用)",
	ErrNotifyPolicyStatusInvalid: "通知策略状态必须为 1(启用) 或 -1(禁用)",

	// 默认通知模板
	TmplAlertNotifyName:   "告警通知模板",
	TmplAlertNotifyTitle:  "告警通知-{{.ruleName}}",
	TmplAlertRecoverName:  "告警恢复模板",
	TmplAlertRecoverTitle: "告警恢复-{{.ruleName}}",
}
