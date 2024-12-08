package imsg

import (
	"mayfly-go/internal/common/consts"
	"mayfly-go/pkg/i18n"
)

func init() {
	i18n.AppendLangMsg(i18n.Zh_CN, Zh_CN)
	i18n.AppendLangMsg(i18n.En, En)
}

const (
	LogMachineSave = iota + consts.ImsgNumMachine
	LogMachineDelete
	LogMachineChangeStatus
	LogMachineKillProcess
	LogMachineTerminalOp

	ErrMachineExist
	ErrSshTunnelCircular

	// file
	LogMachineFileConfSave
	LogMachineFileConfDelete
	LogMachineFileRead
	LogMachineFileDownload
	LogMachineFileModify
	LogMachineFileCreate
	LogMachineFileUpload
	LogMachineFileUploadFolder
	LogMachineFileDelete
	LogMachineFileCopy
	LogMachineFileMove
	LogMachineFileRename

	ErrFileTooLargeUseDownload
	ErrUploadFileOutOfLimit
	ErrFileUploadFail
	MsgUploadFileSuccess

	// security
	LogMachineSecurityCmdSave
	LogMachineSecurityCmdDelete

	TerminalCmdDisable
)
