package application

import (
	"mayfly-go/pkg/ioc"
	"sync"
)

func InitIoc() {
	ioc.Register(new(machineAppImpl))
	ioc.Register(new(machineFileAppImpl))
	ioc.Register(new(machineScriptAppImpl))
	ioc.Register(new(machineCronJobAppImpl))
	ioc.Register(new(machineTermOpAppImpl))
	ioc.Register(new(machineCmdConfAppImpl))
	ioc.Register(new(machineHostKeyAppImpl))
	ioc.Register(new(machineBatchExecImpl))
	ioc.Register(new(machineMetricAppImpl))
	ioc.Register(new(machineDiskAppImpl))
	ioc.Register(new(machineBatchFileAppImpl))
}

func Init() {
	sync.OnceFunc(func() {
		GetMachineCronJobApp().InitCronJob()

		GetMachineApp().TimerUpdateStats()

		GetMachineTermOpApp().TimerDeleteTermOp()

		GetMachineMetricApp().TimerDeleteMetric()
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

func GetMachineCmdConfApp() MachineCmdConf {
	return ioc.Get[MachineCmdConf]()
}

func GetMachineHostKeyApp() MachineHostKey {
	return ioc.Get[MachineHostKey]()
}

func GetMachineBatchExecApp() MachineBatchExec {
	return ioc.Get[MachineBatchExec]()
}

func GetMachineMetricApp() MachineMetric {
	return ioc.Get[MachineMetric]()
}

func GetMachineDiskApp() MachineDisk {
	return ioc.Get[MachineDisk]()
}

func GetMachineBatchFileApp() MachineBatchFile {
	return ioc.Get[MachineBatchFile]()
}
