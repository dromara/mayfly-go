package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(redisAppImpl), ioc.WithComponentName("RedisApp"))
}

func Init() {
	InitRedisFlowHandler()
}
