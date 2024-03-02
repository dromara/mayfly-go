package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(procdefAppImpl), ioc.WithComponentName("ProcdefApp"))
	ioc.Register(new(procinstAppImpl), ioc.WithComponentName("ProcinstApp"))
}
