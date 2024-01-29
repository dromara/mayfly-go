package application

import (
	"mayfly-go/internal/machine/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	persistence.Init()

	ioc.Register(new(machineAppImpl), ioc.WithComponentName("MachineApp"))
	ioc.Register(new(machineFileAppImpl), ioc.WithComponentName("MachineFileApp"))
	ioc.Register(new(machineScriptAppImpl), ioc.WithComponentName("MachineScriptApp"))
	ioc.Register(new(authCertAppImpl), ioc.WithComponentName("AuthCertApp"))
	ioc.Register(new(machineCronJobAppImpl), ioc.WithComponentName("MachineCronJobApp"))
	ioc.Register(new(machineTermOpAppImpl), ioc.WithComponentName("MachineTermOpApp"))
}

func GetMachineApp() Machine {
	return ioc.Get[Machine]("MachineApp")
}

func GetMachineFileApp() MachineFile {
	return ioc.Get[MachineFile]("MachineFileApp")
}

func GetMachineScriptApp() MachineScript {
	return ioc.Get[MachineScript]("MachineScriptApp")
}

func GetMachineCronJobApp() MachineCronJob {
	return ioc.Get[MachineCronJob]("MachineCronJobApp")
}

func GetMachineTermOpApp() MachineTermOp {
	return ioc.Get[MachineTermOp]("MachineTermOpApp")
}
