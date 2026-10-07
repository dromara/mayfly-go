package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogProcdefSave:   "ProcDef - Save",
	LogProcdefDelete: "ProcDef - Delete",

	ErrProcdefKeyExist:            "the process instance key already exists",
	ErrProcinstTaskAlreadyHandled: "this approval task was already handled by another approver, please refresh the task list",
	ErrProcdefFlowNotExist:        "The process definition does not exist",
	ErrExistProcinstRunning:       "There is a running process instance that cannot be manipulated",
	ErrExistProcinstSuspended:     "There is a pending process instance that cannot be manipulated",

	ErrTriggerPolicyInvalid:  "the trigger policy is invalid [{{.source}}]: {{.reason}}",
	ErrTriggerPolicyRequired: "configure the trigger policy before simulating it",
	ErrOperationForbidden:    "the operation is forbidden by the administrator",
	// Server-side reason for a matched custom condition: unlike a built-in check it has no
	// fixed rule name, but it must still explain itself instead of leaving empty parentheses.
	ErrBlockReasonSuffix:              " ({{.reason}})",
	ErrNeedWarnConfirm:                "the operation matched a warning-only rule; run it as-is or submit a work ticket for approval first",
	ErrNeedWarnConfirmDirect:          "the operation matched a warning-only rule; it runs as-is once confirmed",
	TriggerWarnEcho:                   "policy reminder: matched {{.reason}}, executed without blocking",
	TriggerReasonCustomCondition:      "matched a custom trigger condition",
	TriggerReasonFallbackUnhonourable: "the fallback severity cannot be honoured for this resource type (no approval channel), so the strictest available action is applied",

	ErrUserTaskNodeCandidateNotEmpty: "The candidate of the user task node [{{.name}}] cannot be empty",

	// procinst
	LogProcinstStart:  "Process - Start",
	LogProcinstCancel: "Process - Cancel",
	LogCompleteTask:   "Process - Completion of task",
	LogRejectTask:     "Process - Task rejection",
	LogBackTask:       "Process - Task rejection",

	ErrProcdefNotEnable:   "The process defines a non-enabled state",
	ErrProcinstCancelSelf: "You can only cancel processes you initiated",
	ErrProcinstCancelled:  "Process has been cancelled",
	ErrBizHandlerFail:     "Business process failure",

	ErrAiTaskNodeAuditRuleNotEmpty: "The audit rule of the AI task node [{{.name}}] cannot be empty",

	ErrProcinstNotBackStatus: "The work order is not in returned status, cannot modify",
	ErrProcinstNotCreator:    "The work order was not created by the current user, cannot modify",

	// reusable rule segments
	LogRuleSegmentSave:   "Flow Rule Segment - Save",
	LogRuleSegmentDelete: "Flow Rule Segment - Delete",

	ErrRuleSegmentRefInvalid:        "the condition group ref [{{.ref}}] is invalid: it must start with a lowercase letter or digit and contain only lowercase letters, digits, underscores and hyphens",
	ErrRuleSegmentRefExist:          "the condition group ref [{{.ref}}] already exists, a ref is the foreign key written into rules and must be unique",
	ErrRuleSegmentRefImmutable:      "the condition group ref [{{.ref}}] is referenced by rules and cannot be changed after creation",
	ErrRuleSegmentBizUnknown:        "the biz type [{{.bizType}}] of the condition group is not registered, its fields cannot be validated",
	ErrRuleSegmentConditionRequired: "configure at least one condition",
	ErrRuleSegmentReferenced:        "the condition group [{{.ref}}] is still referenced by: {{.referrers}}",

	ErrFlowConditionInvalid:           "the flow condition is invalid [{{.source}}]: {{.reason}}",
	ErrUserTaskNodeCompletionRequired: "the user task node [{{.name}}] must declare a completion condition (or-sign / and-sign / custom)",

	// 校验失败原因
	ReasonPolicyVersion:        "policy model version {{.version}} is newer than this engine supports ({{.support}})",
	ReasonSeverityInvalid:      "the fallback severity used when no rule matches is invalid",
	ReasonCheckKeyRequired:     "the check key is required",
	ReasonCheckUnregistered:    "this check has been retired or was never registered, configure it again",
	ReasonCheckBizRequired:     "a check must declare the biz type it applies to",
	ReasonBizUnregistered:      "the biz type {{.bizType}} is not registered",
	ReasonCheckNotForBiz:       "this check does not apply to the biz type {{.bizType}}",
	ReasonCheckDuplicated:      "this check is configured twice for the same biz type",
	ReasonSeverityNoChannel:    "the biz type {{.bizType}} declares no approval channel, so requiring a work order cannot be fulfilled",
	ReasonSeverityNotConfig:    "the severity {{.severity}} cannot be configured on a rule, remove the rule to disable it",
	ReasonParamUndeclared:      "this parameter is not declared by the check",
	ReasonParamRequired:        "the parameter {{.param}} is required",
	ReasonParamNotNumber:       "the parameter {{.param}} must be a number",
	ReasonParamTooSmall:        "the parameter {{.param}} must be >= {{.min}}",
	ReasonParamTooLarge:        "the parameter {{.param}} must be <= {{.max}}",
	ReasonParamNotBool:         "the parameter {{.param}} must be a boolean",
	ReasonParamOptionRequired:  "the parameter {{.param}} needs at least one option",
	ReasonParamOptionInvalid:   "{{.value}} is not one of the options declared by the parameter {{.param}}",
	ReasonParamValueRequired:   "the parameter {{.param}} needs at least one value",
	ReasonCustomRequired:       "at least one of when and unless is required",
	ReasonCustomDuplicated:     "the custom condition is configured twice for the same biz type",
	ReasonNodeCountExceeded:    "the condition holds {{.count}} nodes, the limit is {{.limit}}",
	ReasonNodeDepthExceeded:    "the condition is {{.depth}} levels deep, the limit is {{.limit}} levels",
	ReasonUnknownKind:          "unknown condition node kind {{.kind}}",
	ReasonUnknownLogic:         "unknown group logic {{.logic}}",
	ReasonGroupEmpty:           "a condition group requires at least one condition",
	ReasonFieldRequired:        "choose the field to compare",
	ReasonFieldUnregistered:    "the field {{.field}} is not registered in this scenario",
	ReasonOperatorRequired:     "choose the operator",
	ReasonOperatorUnregistered: "the operator {{.op}} is not registered",
	ReasonOperatorMismatch:     "the operator {{.op}} does not apply to the field {{.field}} of type {{.type}}",
	ReasonOperatorNotAllowed:   "the field {{.field}} does not allow the operator {{.op}}",
	ReasonValueRequired:        "enter the value to compare with",
	ReasonValueMultiRequired:   "the operator {{.op}} requires at least one expected value",
	ReasonValueRangeRequired:   "the operator {{.op}} requires both a lower and an upper bound",
	ReasonValueBoolExpected:    "the field {{.field}} is boolean, its expected value must be true or false",
	ReasonValueNumberExpected:  "the field {{.field}} is numeric, its expected value must be a number",
	ReasonValueEnumInvalid:     "{{.value}} is not one of the values the field {{.field}} can take",
	ReasonRegexInvalid:         "the regular expression cannot be compiled: {{.detail}}",
	ReasonSegmentRefRequired:   "choose the condition group to reference",
	ReasonSegmentUnregistered:  "the referenced condition group {{.ref}} does not exist",
	ReasonSegmentCycle:         "the condition group {{.ref}} references itself",
	ReasonSegmentTooDeep:       "the condition group nesting exceeds {{.limit}} levels",
	ReasonSegmentNoResolver:    "the referenced condition group {{.ref}} cannot be resolved",
	ReasonSegmentCrossBiz:      "the condition group {{.ref}} belongs to {{.owner}} but is referenced from {{.used}}",
	ReasonScenarioNotGoverned:  "the policy configures scenario {{.bizType}}, but none of the bound resources is governed by it, so those rules can never match",
}
