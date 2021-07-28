package global

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	Log *logrus.Logger // 日志
	Db  *gorm.DB       // gorm
)
