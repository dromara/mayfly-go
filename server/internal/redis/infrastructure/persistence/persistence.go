package persistence

import (
	"mayfly-go/internal/redis/domain/repository"
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newRedisRepo(), ioc.WithComponentName("RedisRepo"))
}

func GetRedisRepo() repository.Redis {
	return ioc.Get[repository.Redis]("RedisRepo")
}
