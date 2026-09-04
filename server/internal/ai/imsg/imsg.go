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
	MissingRequiredParams
	SqlExecApprovalReason
	ExecSqlToolDesc
	ExecSqlToolInfo
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
	// Resource tool
	ResourceListToolDesc
	ResourceListToolInfo
)
