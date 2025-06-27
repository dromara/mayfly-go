package config

import (
	"cmp"
	sysapp "mayfly-go/internal/sys/application"

	"github.com/spf13/cast"
)

const (
	ConfigKeyFile string = "FileConfig" // 文件配置key
)

type FileConfig struct {
	BasePath string // 文件基础路径
}

func GetFileConfig() *FileConfig {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyFile)
	jm := c.GetJsonMap()

	fc := new(FileConfig)
	fc.BasePath = cmp.Or(cast.ToString(jm["basePath"]), "./file")
	return fc
}
