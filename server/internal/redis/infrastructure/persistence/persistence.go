package persistence

import (
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newRedisRepo(), ioc.WithComponentName("RedisRepo"))
}
