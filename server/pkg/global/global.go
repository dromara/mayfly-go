package global

import (
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	Log      *logrus.Logger // 日志
	Db       *gorm.DB       // gorm
	RedisCli *redis.Client  // redis
)
