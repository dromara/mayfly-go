package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogMachineSave:         "机器-保存",
	LogMachineDelete:       "机器-删除",
	LogMachineChangeStatus: "机器-调整状态",
	LogMachineKillProcess:  "机器-终止进程",
	LogMachineTerminalOp:   "机器-终端操作",

	ErrMachineNotFound:   "目标机器不存在",
	ErrMachineExist:      "该机器信息已存在",
	ErrSshTunnelCircular: "存在循环隧道，请重新选择隧道机器",

	// file
	LogMachineFileConfSave:     "机器-新增文件配置",
	LogMachineFileConfDelete:   "机器-删除文件配置",
	LogMachineFileRead:         "机器-读取文件内容",
	LogMachineFileDownload:     "机器-文件下载",
	LogMachineFileModify:       "机器-修改文件内容",
	LogMachineFileCreate:       "机器-创建文件or目录",
	LogMachineFileUpload:       "机器-文件上传",
	LogMachineFileUploadFolder: "机器-文件夹上传",
	LogMachineFileDelete:       "机器-删除文件or文件夹",
	LogMachineFileCopy:         "机器-拷贝文件",
	LogMachineFileMove:         "机器-移动文件",
	LogMachineFileRename:       "机器-文件重命名",

	ErrFileTooLargeUseDownload: "该文件超过1m，请使用下载查看",
	ErrUploadFileOutOfLimit:    "文件大小不能超过{{.size}}字节",
	ErrFileUploadFail:          "文件上传失败",
	MsgUploadFileSuccess:       "文件上传成功",

	LogMachineCronJobSave:   "机器-保存计划任务",
	LogMachineCronJobDelete: "机器-删除计划任务",
	LogMachineCronJobRun:    "机器-执行计划任务",
	ErrCronJobSpecInvalid:   "计划任务的 cron 表达式【{{.cron}}】非法，任务不会按计划执行：{{.reason}}",

	LogMachineSecurityCmdSave:   "机器-安全-保存命令配置",
	LogMachineSecurityCmdDelete: "机器-安全-删除命令配置",
	TerminalCmdDisable:          "该命令已被禁用",

	TriggerReasonCmdBlacklisted: "命中管理员配置的命令黑名单",
	TriggerReasonCmdUnsafe:      "命令含重定向或命令替换等不可静态审计的结构",

	LogMachineHostKeyDelete: "机器-撤销主机密钥信任",
	ErrHostKeyMismatch:      "主机 {{.addr}} 公钥指纹校验失败：已信任 {{.oldFp}}，本次为 {{.newFp}}。若该主机确为重装或换钥，请在「主机密钥」管理中撤销旧指纹后重连；否则请立即排查是否存在中间人攻击",
	LogMachineBatchRunCmd:   "机器-批量执行命令",
	ErrBatchMachineIdsEmpty: "请至少选择一台机器",
	ErrBatchCmdEmpty:        "命令内容不能为空",
	ErrBatchExecMaxMachines: "批量执行机器数不能超过 {{.max}} 台",
	ErrBatchConnFailed:      "连接失败：{{.reason}}",
	BatchExecTotalTimeout:   "批量执行整体超时：该台未执行命令",

	ErrMachineNotFoundById: "机器 [{{.machineId}}] 不存在",

	LogMachineDiskAnalyze:         "机器-磁盘占用分析",
	ErrDiskPathInvalid:            "路径非法：仅允许字母、数字与 . _ / -",
	LogMachineBatchFile:           "机器-批量文件分发",
	ErrBatchFileKeyEmpty:          "请先上传待分发的文件",
	ErrBatchRemotePathEmpty:       "目标目录不能为空",
	ErrBatchRemotePathInvalid:     "目标目录非法：需为绝对路径，且不能含 ; | & $ ` < > ( ) * ? 引号与反斜杠等元字符或 ..",
	ErrBatchFileMaxMachines:       "批量分发机器数不能超过 {{.max}} 台",
	BatchDispatchTotalTimeout:     "批量分发整体超时：该台未分发文件",
	BatchDispatchOnlySsh:          "仅 SSH 机器支持批量文件分发",
	BatchDispatchOpenSourceFailed: "读取源文件失败：{{.reason}}",
}
