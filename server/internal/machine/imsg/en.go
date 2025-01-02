package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogMachineSave:         "Machine - Save",
	LogMachineDelete:       "Machine - Delete",
	LogMachineChangeStatus: "Machine - Change Status",
	LogMachineKillProcess:  "Machine - Kill Process",
	LogMachineTerminalOp:   "Machine - Open Terminal",

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

	LogMachineSecurityCmdSave:   "Machine - Security - Save command configuration",
	LogMachineSecurityCmdDelete: "Machine - Security - Delete command configuration",
	TerminalCmdDisable:          "This command has been disabled...",
}
