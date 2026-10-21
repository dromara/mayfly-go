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
	InfoIncomplete = iota + consts.ImsgNumAi
	ParamCompletionTitle
	ApprovalTitle
	ApprovalDesc
	RejectReasonDefault
	TurnCancelledByUser
	TurnRejectedByUser
	MissingRequiredParams
	SQLExecApprovalReason
	ExecSQLToolDesc
	ExecSQLToolInfo
	DbQueryDataToolDesc
	DbQueryDataToolInfo
	DbQueryTableDDLToolDesc
	DbQueryTableDDLToolInfo
	DbInfoIncomplete
	DbQueryTablesToolDesc
	DbQueryTablesToolInfo
	// Machine tools
	MachineCommandExecToolDesc
	MachineCommandExecToolInfo
	MachineInfoIncomplete
	CommandExecApprovalReason
	InterruptExpired
	// 会话级互斥：该会话已有运行中 turn（原为哨兵 errors.New 的英文串直透前端）
	TurnAlreadyRunning
	// Resource tool
	ResourceListToolDesc
	ResourceListToolInfo

	// 面向用户的入口错误（原为硬编码英文串，直透前端 toast）
	// 会话授权：三条 Assert* 的共同拒绝语义
	ErrConvLoginRequired
	ErrConvNotFound
	ErrConvReadDenied
	ErrConvModifyDenied
	ErrConvWriteDenied
	ErrConvSystemReadOnly
	// headless 运行入口
	ErrHeadlessReqNil
	ErrHeadlessOwnerRequired
	ErrUnknownAgentProfile
	// 技能 zip 上传
	ErrSkillZipTooLarge
	// 监控工具
	ErrAlertEventAppNotReady
)
