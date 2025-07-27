package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(NewInstanceRepo(), ioc.WithComponentName("EsInstanceRepo"))
}
