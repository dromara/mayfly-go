package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogMachineSave:         "Machine - Save",
	LogMachineDelete:       "Machine - Delete",
	LogMachineChangeStatus: "Machine - Change Status",
	LogMachineKillProcess:  "Machine - Kill Process",
	LogMachineTerminalOp:   "Machine - Open Terminal",

	ErrMachineNotFound:   "the target machine does not exist",
	ErrMachineExist:      "The machine information already exists",
	ErrSshTunnelCircular: "Circular tunnel exists, please select tunnel machine again",

	// file
	LogMachineFileConfSave:     "Machine - New file config",
	LogMachineFileConfDelete:   "Machine - Delete file Config",
	LogMachineFileRead:         "Machine - Reading file contents",
	LogMachineFileDownload:     "Machine - File Download",
	LogMachineFileModify:       "Machine - Modifying file contents",
	LogMachineFileCreate:       "Machine - Create a file or directory",
	LogMachineFileUpload:       "Machine - File Upload",
	LogMachineFileUploadFolder: "Machine - Folder Upload",
	LogMachineFileDelete:       "Machine - Delete a file or directory",
	LogMachineFileCopy:         "Machine - Copy File",
	LogMachineFileMove:         "Machine - Move File",
	LogMachineFileRename:       "Machine - Rename File",

	ErrFileTooLargeUseDownload: "The file is over 1m, please use download to view",
	ErrUploadFileOutOfLimit:    "The file size cannot exceed {{.size}} bytes",
	ErrFileUploadFail:          "File upload failure",
	MsgUploadFileSuccess:       "File uploaded successfully",

	LogMachineCronJobSave:   "Machine - save cronjob",
	LogMachineCronJobDelete: "Machine - delete cronjob",
	LogMachineCronJobRun:    "Machine - run cronjob",
	ErrCronJobSpecInvalid:   "The cron expression [{{.cron}}] of the cron job is invalid, so it will never run as planned: {{.reason}}",

	LogMachineSecurityCmdSave:   "Machine - Security - Save command configuration",
	LogMachineSecurityCmdDelete: "Machine - Security - Delete command configuration",
	TerminalCmdDisable:          "this command has been disabled",

	TriggerReasonCmdBlacklisted: "it matches the administrator command blacklist",
	TriggerReasonCmdUnsafe:      "the command contains redirection or command substitution",

	LogMachineHostKeyDelete: "Machine - Revoke host key trust",
	ErrHostKeyMismatch:      "Host key fingerprint mismatch for {{.addr}}: trusted {{.oldFp}}, got {{.newFp}}. If the host was reinstalled or its key rotated, revoke the old fingerprint in Host Keys management and reconnect; otherwise investigate a possible man-in-the-middle attack immediately",
	LogMachineBatchRunCmd:   "Machine - Batch run command",
	ErrBatchMachineIdsEmpty: "please select at least one machine",
	ErrBatchCmdEmpty:        "the command content cannot be empty",
	ErrBatchExecMaxMachines: "the number of machines for batch execution cannot exceed {{.max}}",
	ErrBatchConnFailed:      "connection failed: {{.reason}}",
	BatchExecTotalTimeout:   "batch execution timed out overall: the command was not run on this machine",

	ErrMachineNotFoundById: "machine [{{.machineId}}] not found",

	LogMachineDiskAnalyze:         "Machine - Disk usage analysis",
	ErrDiskPathInvalid:            "invalid path: only letters, digits and . _ / - are allowed",
	LogMachineBatchFile:           "Machine - Batch file dispatch",
	ErrBatchFileKeyEmpty:          "upload the file to dispatch first",
	ErrBatchRemotePathEmpty:       "the target directory cannot be empty",
	ErrBatchRemotePathInvalid:     "invalid target directory: it must be an absolute path without metacharacters such as ; | & $ ` < > ( ) * ? quotes and backslash, and without ..",
	ErrBatchFileMaxMachines:       "the number of machines for batch dispatch cannot exceed {{.max}}",
	BatchDispatchTotalTimeout:     "batch dispatch timed out overall: the file was not dispatched to this machine",
	BatchDispatchOnlySsh:          "only ssh machines support batch file dispatch",
	BatchDispatchOpenSourceFailed: "failed to read the source file: {{.reason}}",
}
