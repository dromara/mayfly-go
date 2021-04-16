package global

import (
	"mayfly-go/base/config"
	"mayfly-go/base/mlog"

	"gorm.io/gorm"
)

// 日志
var Log = mlog.Log

// config.yml配置文件映射对象
var Config = config.Conf

// gorm
var Db *gorm.DB
