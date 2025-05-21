package application

import (
	"mayfly-go/pkg/ioc"
	"sync"
)

func InitIoc() {
	ioc.Register(new(instanceAppImpl), ioc.WithComponentName("EsInstanceApp"))
}

func Init() {
	sync.OnceFunc(func() {
	})()
}
