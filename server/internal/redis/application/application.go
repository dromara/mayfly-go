package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(redisAppImpl))
	ioc.Register(new(keyValueAppImpl))
}

func Init() {
	InitRedisFlowHandler()
}
