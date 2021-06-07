package global

import (
	"mayfly-go/base/config"
	"mayfly-go/base/logger"

	"gorm.io/gorm"
)

// 日志
var Log = logger.Log

// config.yml配置文件映射对象
var Config = config.Conf

// gorm
var Db *gorm.DB
