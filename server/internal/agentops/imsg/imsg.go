package imsg

import (
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
)

func init() {
	i18n.AppendLangMsg(i18n.Zh_CN, Zh_CN)
	i18n.AppendLangMsg(i18n.En, En)
}

// 智能运维（agentops）消息 id。
//
// 刻意按「资源域」分段连续声明，且中英文文案必须成对（由 imsg_test 钉住）：本模块的校验错误
// 会以 toast 直接呈现给运维人员，某一语言漏配就静默回退成另一种语言，中文界面上冒出英文串。
//
// 顺序约束：新增消息只在所属分段**末尾追加**，不要插到中间——iota 顺序编号，中间插入会让后续
// 所有 id 平移，而 id 已被写进历史操作日志（t_sys_log），平移后旧日志会集体错译成别的文案。
const (
	// ---- 操作日志（api 层 req.NewLogSaveI）----
	LogTaskSave = iota + consts.ImsgNumAgentops
	LogTaskDelete
	LogTaskRun
	LogTriggerSave
	LogTriggerDelete
	LogRunCancel
	LogRunFeedback
	LogRunEval
	LogOrchestrationRun
	LogProposalDelete
	LogGovernanceKillSwitch
	LogGovernanceCancelAll

	// ---- 运行身份与资源可见面 ----
	ErrLoginAccountRequired
	ErrNotPermittedImpersonate
	ErrOwnerAccountNotFound
	ErrResourceNotFound

	// ---- 运维任务 ----
	ErrTaskNil
	ErrTaskOwnerRequired
	ErrTaskInstructionRequired
	ErrTaskCronInvalid
	ErrTaskNotifyTmplRequired
	ErrTaskNotFound

	// ---- 事件触发器与触发源 ----
	ErrTriggerNil
	ErrTriggerNameRequired
	ErrTriggerSourceRequired
	ErrTriggerSourceUnknown
	ErrTriggerOwnerRequired
	ErrTriggerInstructionRequired
	ErrTriggerTaskNotFound
	ErrTriggerMatchCondInvalid
	ErrTriggerMatchCondTrailing
	ErrTriggerMatchCondUnusable
	ErrTriggerNotFound
	ErrTriggerTokenInUse
	ErrTriggerNotWebhook
	ErrTriggerRotateLost
	ErrTriggerTokenAllocateFailed
	ErrTriggerTokenGenerateFailed
	ErrTriggerEventNil
	ErrTriggerEventNotMatch
	ErrTriggerInCooldown
	ErrWebhookTokenRequired
	ErrWebhookTokenInvalid
	ErrWebhookTriggerDisabled
	ErrWebhookSourceNotRegistered
	ErrWebhookPayloadNotObject
	ErrAlertPayloadInvalid

	// ---- 运行生命周期 ----
	ErrRunNotFound
	ErrRunNotCancelable
	ErrRunEnqueueReqNil
	ErrRunEnqueueOwnerRequired
	ErrRunEnqueueInstructionRequired
	ErrRunReqNil
	ErrRunSyncOwnerRequired
	ErrRunSyncInstructionRequired
	ErrRunTerminalUndeterminable
	ErrRunPanicked

	// ---- 变更提案与受控执行 ----
	ErrProposalNotFound
	ErrProposalNotTerminal
	ErrProposalDraftNil
	ErrProposalActionRequired
	ErrProposalActionNotTool
	ErrProposalOwnerRequired
	ErrProposalFlowNotReady
	ErrProposalBizParamNil
	ErrProposalExecutorNotStarted
	ErrProposalExecutorStopped
	ErrProposalNoActionForGrant
	ErrProposalActionNotMutable

	// ---- 反馈与离线评测 ----
	ErrFeedbackValueInvalid
	ErrFeedbackRequiresTerminal
	ErrEvalRequiresTerminal
	ErrEvalRequiresOriginalInput
	ErrEvalNoOwner

	// ---- 多 Agent 编排 ----
	ErrOrchGoalRequired
	ErrOrchOwnerRequired
	ErrOrchConcurrencyLimit

	// ---- 档位派发守护 ----
	ErrProfileUnknown
	ErrProfileRequiresGrant

	// ---- 治理（急停）----
	ErrKillSwitchEngaged

	// ---- 结果通知配置 ----
	ErrTriggerNotifyTmplRequired

	// ---- webhook 推送体 ----
	// 只能往块尾追加：iota 从 base 起连续，中间插入会让已写进 t_sys_log 的历史 id 平移错译
	ErrWebhookPayloadUnparsable
)
