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
	LogMachineSave = iota + consts.ImsgNumMachine
	LogMachineDelete
	LogMachineChangeStatus
	LogMachineKillProcess
	LogMachineTerminalOp

	ErrMachineExist
	ErrMachineNotFound
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

	LogMachineCronJobSave
	LogMachineCronJobDelete
	LogMachineCronJobRun
	ErrCronJobSpecInvalid

	// security
	LogMachineSecurityCmdSave
	LogMachineSecurityCmdDelete

	TerminalCmdDisable

	// 触发策略检查项命中原因（用于拦截提示）
	TriggerReasonCmdBlacklisted
	TriggerReasonCmdUnsafe

	// 主机密钥
	LogMachineHostKeyDelete
	ErrHostKeyMismatch

	// 批量命令执行
	LogMachineBatchRunCmd
	ErrBatchMachineIdsEmpty
	ErrBatchCmdEmpty
	ErrBatchExecMaxMachines
	ErrBatchConnFailed
	BatchExecTotalTimeout

	ErrMachineNotFoundById

	// 磁盘占用分析
	LogMachineDiskAnalyze
	ErrDiskPathInvalid

	// 批量文件分发
	LogMachineBatchFile
	ErrBatchFileKeyEmpty
	ErrBatchRemotePathEmpty
	ErrBatchRemotePathInvalid
	ErrBatchFileMaxMachines
	BatchDispatchTotalTimeout
	BatchDispatchOnlySsh
	BatchDispatchOpenSourceFailed
)
