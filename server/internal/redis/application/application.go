package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(redisAppImpl))
}

func Init() {
	InitRedisFlowHandler()
}
