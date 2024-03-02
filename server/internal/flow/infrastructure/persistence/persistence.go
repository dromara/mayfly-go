package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newProcdefRepo(), ioc.WithComponentName("ProcdefRepo"))
	ioc.Register(newProcinstRepo(), ioc.WithComponentName("ProcinstRepo"))
	ioc.Register(newProcinstTaskRepo(), ioc.WithComponentName("ProcinstTaskRepo"))
}
