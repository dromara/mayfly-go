package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newMachineRepo(), ioc.WithComponentName("MachineRepo"))
	ioc.Register(newMachineFileRepo(), ioc.WithComponentName("MachineFileRepo"))
	ioc.Register(newMachineScriptRepo(), ioc.WithComponentName("MachineScriptRepo"))
	ioc.Register(newMachineCronJobRepo(), ioc.WithComponentName("MachineCronJobRepo"))
	ioc.Register(newMachineCronJobExecRepo(), ioc.WithComponentName("MachineCronJobExecRepo"))
	ioc.Register(newMachineTermOpRepoImpl(), ioc.WithComponentName("MachineTermOpRepo"))
	ioc.Register(newMachineCmdConfRepo(), ioc.WithComponentName("MachineCmdConfRepo"))
}
