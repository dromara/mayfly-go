package application

import (
	"mayfly-go/internal/redis/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	persistence.Init()

	ioc.Register(new(redisAppImpl), ioc.WithComponentName("RedisApp"))
}
