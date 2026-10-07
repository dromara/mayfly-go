package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogProcdefSave:   "流程定义-保存",
	LogProcdefDelete: "流程定义-删除",

	ErrProcdefKeyExist:            "该流程实例key已存在",
	ErrProcinstTaskAlreadyHandled: "该审批任务已被其他审批人处理，请刷新待办列表",
	ErrProcdefFlowNotExist:        "流程审批流未设置",
	ErrExistProcinstRunning:       "存在运行中的流程实例，无法操作",
	ErrExistProcinstSuspended:     "存在挂起中的流程实例，无法操作",

	ErrTriggerPolicyInvalid:  "触发策略配置无效 [{{.source}}]：{{.reason}}",
	ErrTriggerPolicyRequired: "请先配置触发策略再试算",
	ErrOperationForbidden:    "该操作已被管理员禁止执行",
	// 自定义触发条件命中时的服务端原因：它没有像内置检查项那样固定的规则名，
	// 但仍必须给出一句可解释的话，否则拦截提示会变成空括号
	ErrBlockReasonSuffix:              "（{{.reason}}）",
	ErrNeedWarnConfirm:                "该操作命中了策略提醒，可直接执行，或提交工单审批后执行",
	ErrNeedWarnConfirmDirect:          "该操作命中了策略提醒，确认后将直接执行",
	TriggerWarnEcho:                   "策略提醒：命中 {{.reason}}，未阻断执行",
	TriggerReasonCustomCondition:      "命中自定义触发条件",
	TriggerReasonFallbackUnhonourable: "未配置规则时的兜底级别在该资源类型上无法落地（没有审批通道），已按最严格处置拦截",

	ErrUserTaskNodeCandidateNotEmpty: "用户任务节点 [{{.name}}] 的候选人不能为空",

	// procinst
	LogProcinstStart:  "流程-启动",
	LogProcinstCancel: "流程-取消",
	LogCompleteTask:   "流程-任务完成",
	LogRejectTask:     "流程-任务拒绝",
	LogBackTask:       "流程-任务驳回",

	ErrProcdefNotEnable:   "该流程定义非启用状态",
	ErrProcinstCancelSelf: "只能取消自己发起的流程",
	ErrProcinstCancelled:  "流程已取消",
	ErrBizHandlerFail:     "业务处理失败",

	ErrAiTaskNodeAuditRuleNotEmpty: "Ai任务节点 [{{.name}}] 的审核规则不能为空",

	ErrProcinstNotBackStatus: "该工单非退回状态，无法修改",
	ErrProcinstNotCreator:    "该工单非当前用户创建，无法修改",

	// 可复用条件组
	LogRuleSegmentSave:   "流程条件组-保存",
	LogRuleSegmentDelete: "流程条件组-删除",

	ErrRuleSegmentRefInvalid:        "条件组标识 [{{.ref}}] 格式无效：以小写字母或数字开头，只能包含小写字母、数字、下划线与中划线",
	ErrRuleSegmentRefExist:          "条件组标识 [{{.ref}}] 已存在，标识是引用方写进规则里的外键，不能重复",
	ErrRuleSegmentRefImmutable:      "条件组标识 [{{.ref}}] 已被规则引用，创建后不可修改",
	ErrRuleSegmentBizUnknown:        "条件组所属的业务场景 [{{.bizType}}] 未注册，无法校验其字段",
	ErrRuleSegmentConditionRequired: "请至少配置一个条件",
	ErrRuleSegmentReferenced:        "条件组 [{{.ref}}] 正被以下规则引用，无法删除：{{.referrers}}",

	ErrFlowConditionInvalid:           "流程条件配置无效 [{{.source}}]：{{.reason}}",
	ErrUserTaskNodeCompletionRequired: "用户任务节点 [{{.name}}] 必须配置审批完成条件（或签 / 会签 / 自定义）",

	// 校验失败原因
	ReasonPolicyVersion:        "策略模型版本 {{.version}} 高于当前引擎支持的 {{.support}}",
	ReasonSeverityInvalid:      "未配置规则时的兜底处置级别取值非法",
	ReasonCheckKeyRequired:     "检查项标识不能为空",
	ReasonCheckUnregistered:    "该检查项已下线或从未注册，请重新配置",
	ReasonCheckBizRequired:     "检查项必须声明生效的业务场景",
	ReasonBizUnregistered:      "业务场景 {{.bizType}} 未注册",
	ReasonCheckNotForBiz:       "该检查项不适用于业务场景 {{.bizType}}",
	ReasonCheckDuplicated:      "同一业务场景下该检查项重复配置",
	ReasonSeverityNoChannel:    "业务场景 {{.bizType}} 没有审批通道，配「需审批」无法落地",
	ReasonSeverityNotConfig:    "级别 {{.severity}} 不能配到规则上；要取消处置请移除该条规则",
	ReasonParamUndeclared:      "该参数不属于此检查项的声明",
	ReasonParamRequired:        "参数 {{.param}} 为必填",
	ReasonParamNotNumber:       "参数 {{.param}} 必须是数值",
	ReasonParamTooSmall:        "参数 {{.param}} 不得小于 {{.min}}",
	ReasonParamTooLarge:        "参数 {{.param}} 不得大于 {{.max}}",
	ReasonParamNotBool:         "参数 {{.param}} 必须是布尔值",
	ReasonParamOptionRequired:  "参数 {{.param}} 至少选择一个选项",
	ReasonParamOptionInvalid:   "参数 {{.param}} 的取值 {{.value}} 不在候选项内",
	ReasonParamValueRequired:   "参数 {{.param}} 至少需要一个取值",
	ReasonCustomRequired:       "触发条件与豁免条件至少填写一项",
	ReasonCustomDuplicated:     "同一业务场景只能配置一条自定义条件",
	ReasonNodeCountExceeded:    "条件共 {{.count}} 个节点，超出上限 {{.limit}}",
	ReasonNodeDepthExceeded:    "条件嵌套 {{.depth}} 层，超出上限 {{.limit}} 层",
	ReasonUnknownKind:          "无法识别的条件节点类型 {{.kind}}",
	ReasonUnknownLogic:         "无法识别的分组组合方式 {{.logic}}",
	ReasonGroupEmpty:           "条件分组至少需要一个条件",
	ReasonFieldRequired:        "请选择要判断的字段",
	ReasonFieldUnregistered:    "字段 {{.field}} 未在该场景的字段字典中注册",
	ReasonOperatorRequired:     "请选择判断条件",
	ReasonOperatorUnregistered: "判断条件 {{.op}} 未注册",
	ReasonOperatorMismatch:     "判断条件 {{.op}} 不适用于字段 {{.field}}（类型 {{.type}}）",
	ReasonOperatorNotAllowed:   "字段 {{.field}} 不允许使用判断条件 {{.op}}",
	ReasonValueRequired:        "请填写比较值",
	ReasonValueMultiRequired:   "判断条件 {{.op}} 至少需要一个比较值",
	ReasonValueRangeRequired:   "判断条件 {{.op}} 需要同时填写上下界",
	ReasonValueBoolExpected:    "布尔字段 {{.field}} 的比较值必须是真/假",
	ReasonValueNumberExpected:  "数值字段 {{.field}} 的比较值必须是数字",
	ReasonValueEnumInvalid:     "比较值 {{.value}} 不在字段 {{.field}} 的候选取值内",
	ReasonRegexInvalid:         "正则表达式无法编译：{{.detail}}",
	ReasonSegmentRefRequired:   "请选择要引用的条件组",
	ReasonSegmentUnregistered:  "引用的条件组 {{.ref}} 不存在",
	ReasonSegmentCycle:         "条件组 {{.ref}} 循环引用自身",
	ReasonSegmentTooDeep:       "条件组嵌套超过 {{.limit}} 层",
	ReasonSegmentNoResolver:    "无法解析引用的条件组 {{.ref}}",
	ReasonSegmentCrossBiz:      "条件组 {{.ref}} 属于场景 {{.owner}}，却被场景 {{.used}} 引用",
	ReasonScenarioNotGoverned:  "策略里配置了场景 {{.bizType}}，但生效资源里没有它治理的资源，这些规则永远不会命中",
}
