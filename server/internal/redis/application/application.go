package application

import (
	"mayfly-go/internal/redis/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
)

func init() {
	persistence.Init()

	ioc.Register(new(redisAppImpl), ioc.WithComponentName("RedisApp"))
}

func GetRedisApp() Redis {
	return ioc.Get[Redis]("RedisApp")
}
