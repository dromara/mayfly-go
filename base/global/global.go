package global

import (
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	Log      *logrus.Logger // 日志
	Db       *gorm.DB       // gorm
	RedisCli *redis.Client  // redis
)
