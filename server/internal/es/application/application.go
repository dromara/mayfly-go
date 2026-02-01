package application

import (
	"mayfly-go/pkg/ioc"
	"sync"
)

func InitIoc() {
	ioc.Register(new(instanceAppImpl))
}

func Init() {
	sync.OnceFunc(func() {
	})()
}
