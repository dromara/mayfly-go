package config

import (
	"cmp"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/bytex"
)

const (
	ConfigKeyMachine string = "MachineConfig" // 机器相关配置
)

type Machine struct {
	UploadMaxFileSize int64  // 允许上传的最大文件size
	TermOpSaveDays    int    // 终端记录保存天数
	MetricSaveDays    int    // 机器指标历史保存天数
	GuacdHost         string // guacd服务地址 默认 127.0.0.1
	GuacdPort         int    // guacd服务端口  默认 4822
	GuacdFilePath     string // guacd服务文件存储位置，用于挂载RDP文件夹
}

// 获取机器相关配置
func GetMachine() *Machine {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyMachine)
	jm := c.GetJsonM()

	mc := new(Machine)

	// 将1GB等字符串转为int64的byte
	uploadMaxFileSizeStr := jm.GetStr("uploadMaxFileSize")
	var uploadMaxFileSize int64 = 1 * bytex.GB
	if uploadMaxFileSizeStr != "" {
		var err error
		uploadMaxFileSize, err = bytex.ParseSize(uploadMaxFileSizeStr)
		if err != nil {
			logx.Errorf("解析机器配置的最大上传文件大小失败: uploadMaxFileSize=%s, 使用系统默认值1GB", uploadMaxFileSizeStr)
		}
	}
	mc.UploadMaxFileSize = uploadMaxFileSize
	mc.TermOpSaveDays = cmp.Or(jm.GetInt("termOpSaveDays"), 30)
	mc.MetricSaveDays = cmp.Or(jm.GetInt("metricSaveDays"), 7)
	// guacd
	mc.GuacdHost = jm.GetStr("guacdHost")
	mc.GuacdPort = cmp.Or(jm.GetInt("guacdPort"), 4822)
	mc.GuacdFilePath = jm.GetStr("guacdFilePath")

	return mc
}
