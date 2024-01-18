package config

import (
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/bytex"
	"mayfly-go/pkg/utils/conv"
)

const (
	ConfigKeyMachine string = "MachineConfig" // 机器相关配置
)

type Machine struct {
	TerminalRecPath   string // 终端操作记录存储位置
	UploadMaxFileSize int64  // 允许上传的最大文件size
	TermOpSaveDays    int    // 终端记录保存天数
}

// 获取机器相关配置
func GetMachine() *Machine {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyMachine)
	jm := c.GetJsonMap()

	mc := new(Machine)

	terminalRecPath := jm["terminalRecPath"]
	if terminalRecPath == "" {
		terminalRecPath = "./rec"
	}
	mc.TerminalRecPath = terminalRecPath

	// 将1GB等字符串转为int64的byte
	uploadMaxFileSizeStr := jm["uploadMaxFileSize"]
	var uploadMaxFileSize int64 = 1 * bytex.GB
	if uploadMaxFileSizeStr != "" {
		var err error
		uploadMaxFileSize, err = bytex.ParseSize(uploadMaxFileSizeStr)
		if err != nil {
			logx.Errorf("解析机器配置的最大上传文件大小失败: uploadMaxFileSize=%s, 使用系统默认值1GB", uploadMaxFileSizeStr)
		}
	}
	mc.UploadMaxFileSize = uploadMaxFileSize
	mc.TermOpSaveDays = conv.Str2Int(jm["termOpSaveDays"], 30)
	return mc
}
