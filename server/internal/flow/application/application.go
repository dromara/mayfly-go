package application

import (
	"mayfly-go/internal/flow/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	persistence.Init()

	ioc.Register(new(procdefAppImpl), ioc.WithComponentName("ProcdefApp"))
	ioc.Register(new(procinstAppImpl), ioc.WithComponentName("ProcinstApp"))
}
