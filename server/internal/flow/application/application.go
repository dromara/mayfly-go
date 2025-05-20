package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(procdefAppImpl), ioc.WithComponentName("ProcdefApp"))
	ioc.Register(new(procinstAppImpl), ioc.WithComponentName("ProcinstApp"))
	ioc.Register(new(executionAppImpl), ioc.WithComponentName("ExecutionApp"))

	ioc.Register(new(procinstTaskAppImpl), ioc.WithComponentName("ProcinstTaskApp"))
	ioc.Register(new(hisProcinstOpAppImpl), ioc.WithComponentName("HisProcinstOpApp"))
}

func Init() {
	GetExecutionApp().Init()
	GetProcinstTaskApp().Init()
}

func GetProcdefApp() Procdef {
	return ioc.Get[Procdef]("ProcdefApp")
}

func GetProcinstApp() Procinst {
	return ioc.Get[Procinst]("ProcinstApp")
}

func GetExecutionApp() Execution {
	return ioc.Get[Execution]("ExecutionApp")
}

func GetHisProcinstOpApp() HisProcinstOp {
	return ioc.Get[HisProcinstOp]("HisProcinstOpApp")
}

func GetProcinstTaskApp() ProcinstTask {
	return ioc.Get[ProcinstTask]("ProcinstTaskApp")
}
