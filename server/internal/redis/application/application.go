package application

import (
	"mayfly-go/internal/redis/infrastructure/persistence"
	tagapp "mayfly-go/internal/tag/application"
)

var (
	redisApp Redis = newRedisApp(persistence.GetRedisRepo(), tagapp.GetTagTreeApp())
)

func GetRedisApp() Redis {
	return redisApp
}
