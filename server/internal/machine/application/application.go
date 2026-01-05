package application

import (
	"mayfly-go/pkg/ioc"
	"sync"
)

func InitIoc() {
	ioc.Register(new(machineAppImpl), ioc.WithComponentName("MachineApp"))
	ioc.Register(new(machineFileAppImpl), ioc.WithComponentName("MachineFileApp"))
	ioc.Register(new(machineScriptAppImpl), ioc.WithComponentName("MachineScriptApp"))
	ioc.Register(new(machineCronJobAppImpl), ioc.WithComponentName("MachineCronJobApp"))
	ioc.Register(new(machineTermOpAppImpl), ioc.WithComponentName("MachineTermOpApp"))
	ioc.Register(new(machineCmdConfAppImpl), ioc.WithComponentName("MachineCmdConfApp"))
}

func Init() {
	sync.OnceFunc(func() {
		GetMachineCronJobApp().InitCronJob()

		GetMachineApp().TimerUpdateStats()

		GetMachineTermOpApp().TimerDeleteTermOp()
	})()
}

func GetMachineApp() Machine {
	return ioc.Get[Machine]()
}

func GetMachineFileApp() MachineFile {
	return ioc.Get[MachineFile]()
}

func GetMachineScriptApp() MachineScript {
	return ioc.Get[MachineScript]()
}

func GetMachineCronJobApp() MachineCronJob {
	return ioc.Get[MachineCronJob]()
}

func GetMachineTermOpApp() MachineTermOp {
	return ioc.Get[MachineTermOp]()
}
