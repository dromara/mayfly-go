package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newMachineRepo())
	ioc.Register(newMachineFileRepo())
	ioc.Register(newMachineScriptRepo())
	ioc.Register(newMachineCronJobRepo())
	ioc.Register(newMachineCronJobExecRepo())
	ioc.Register(newMachineTermOpRepoImpl())
	ioc.Register(newMachineCmdConfRepo())
}
