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
	LogProcdefSave = iota + consts.ImsgNumFlow
	LogProcdefDelete

	ErrProcdefKeyExist
	ErrProcdefFlowNotExist
	ErrExistProcinstRunning
	ErrExistProcinstSuspended
	ErrProcinstTaskAlreadyHandled

	// trigger policy
	ErrTriggerPolicyInvalid
	ErrTriggerPolicyRequired
	ErrOperationForbidden

	ErrUserTaskNodeCandidateNotEmpty

	// procinst
	LogProcinstStart
	LogProcinstCancel
	LogCompleteTask
	LogRejectTask
	LogBackTask

	ErrProcdefNotEnable
	ErrProcinstCancelSelf
	ErrProcinstCancelled
	ErrBizHandlerFail

	ErrAiTaskNodeAuditRuleNotEmpty

	ErrProcinstNotBackStatus
	ErrProcinstNotCreator

	// 可复用条件组（规则片段）
	LogRuleSegmentSave
	LogRuleSegmentDelete

	ErrRuleSegmentRefInvalid
	ErrRuleSegmentRefExist
	ErrRuleSegmentRefImmutable
	ErrRuleSegmentBizUnknown
	ErrRuleSegmentConditionRequired
	ErrRuleSegmentReferenced

	// 流程内条件（连线跳转、节点完成）与策略变更留痕
	TriggerReasonCustomCondition
	// ErrBlockReasonSuffix 阻断提示后附的「命中原因」从句。
	// 由 flow 应用层统一拼装，各业务模块的消息里不再各写一遍括号，
	// 才能做到「原因非空才追加」，也不会出现某个入口漏带原因
	ErrBlockReasonSuffix
	// ErrNeedWarnConfirm 命中「仅提醒」且该入口能ask操作者是否转审批时使用
	ErrNeedWarnConfirm
	// ErrNeedWarnConfirmDirect 同上，但该入口给不出提单按钮（如 key 面板的结构化操作）：
	// 话术不能再写「或提交工单审批后执行」，那是在指一个界面上不存在的出口
	ErrNeedWarnConfirmDirect
	// TriggerWarnEcho 「仅提醒」命中但不支持确认的入口（机器终端）用的回显文本
	TriggerWarnEcho
	// TriggerReasonFallbackUnhonourable 兜底级别在该场景无法落地，已收敛为最严格的可用级别
	TriggerReasonFallbackUnhonourable

	ErrFlowConditionInvalid
	ErrUserTaskNodeCompletionRequired

	// 触发策略/条件校验失败的具体原因。
	//
	// 后端只负责定位与原因分类，中文与英文各一份：这些串会出现在保存被拒时的提示里，
	// 直接抛 Go 原生 fmt 文本会让中文界面出现半中半英的句子
	ReasonPolicyVersion        // 策略版本高于引擎
	ReasonSeverityInvalid      // 级别取值非法
	ReasonCheckKeyRequired     // 检查项标识缺失
	ReasonCheckUnregistered    // 检查项未注册
	ReasonCheckBizRequired     // 检查项未声明场景
	ReasonBizUnregistered      // 场景未注册
	ReasonCheckNotForBiz       // 检查项不适用于该场景
	ReasonCheckDuplicated      // 同一场景重复配置同一检查项
	ReasonSeverityNoChannel    // 该场景没有审批通道，不能配「需审批」
	ReasonSeverityNotConfig    // 该级别不能配到规则上
	ReasonParamUndeclared      // 参数未在检查项中声明
	ReasonParamRequired        // 参数必填
	ReasonParamNotNumber       // 参数必须是数值
	ReasonParamTooSmall        // 参数小于下限
	ReasonParamTooLarge        // 参数大于上限
	ReasonParamNotBool         // 参数必须是布尔
	ReasonParamOptionRequired  // 参数至少需要一个选项
	ReasonParamOptionInvalid   // 参数取值不在候选项内
	ReasonParamValueRequired   // 参数至少需要一个取值
	ReasonCustomRequired       // 触发与豁免条件至少填一项
	ReasonCustomDuplicated     // 同一场景重复配置自定义条件
	ReasonNodeCountExceeded    // 条件节点数超限
	ReasonNodeDepthExceeded    // 条件嵌套层数超限
	ReasonUnknownKind          // 未知条件节点类型
	ReasonUnknownLogic         // 未知分组组合方式
	ReasonGroupEmpty           // 条件组没有任何条件
	ReasonFieldRequired        // 未选择字段
	ReasonFieldUnregistered    // 字段未在该场景注册
	ReasonOperatorRequired     // 未选择条件
	ReasonOperatorUnregistered // 操作符未注册
	ReasonOperatorMismatch     // 操作符与字段类型不匹配
	ReasonOperatorNotAllowed   // 字段不允许该操作符
	ReasonValueRequired        // 缺少比较值
	ReasonValueMultiRequired   // 至少需要一个比较值
	ReasonValueRangeRequired   // 区间缺少上下界
	ReasonValueBoolExpected    // 布尔字段的比较值必须是真/假
	ReasonValueNumberExpected  // 数值字段的比较值必须是数字
	ReasonValueEnumInvalid     // 比较值不在字段候选取值内
	ReasonRegexInvalid         // 正则表达式不可编译
	ReasonSegmentRefRequired   // 未选择要引用的条件组
	ReasonSegmentUnregistered  // 引用的条件组不存在
	ReasonSegmentCycle         // 条件组循环引用
	ReasonSegmentTooDeep       // 条件组嵌套层数超限
	ReasonSegmentNoResolver    // 无法展开条件组引用
	ReasonSegmentCrossBiz      // 条件组跨场景引用
	ReasonScenarioNotGoverned  // 策略配了该场景，但生效资源里没有它治理的资源
)
