package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogMachineSave:         "机器-保存",
	LogMachineDelete:       "机器-删除",
	LogMachineChangeStatus: "机器-调整状态",
	LogMachineKillProcess:  "机器-终止进程",
	LogMachineTerminalOp:   "机器-终端操作",

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

	LogMachineSecurityCmdSave:   "机器-安全-保存命令配置",
	LogMachineSecurityCmdDelete: "机器-安全-删除命令配置",
	TerminalCmdDisable:          "该命令已被禁用...",
}
