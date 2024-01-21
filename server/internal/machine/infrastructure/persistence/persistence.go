package persistence

import (
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newMachineRepo(), ioc.WithComponentName("MachineRepo"))
	ioc.Register(newMachineFileRepo(), ioc.WithComponentName("MachineFileRepo"))
	ioc.Register(newMachineScriptRepo(), ioc.WithComponentName("MachineScriptRepo"))
	ioc.Register(newAuthCertRepo(), ioc.WithComponentName("AuthCertRepo"))
	ioc.Register(newMachineCronJobRepo(), ioc.WithComponentName("MachineCronJobRepo"))
	ioc.Register(newMachineCronJobExecRepo(), ioc.WithComponentName("MachineCronJobExecRepo"))
	ioc.Register(newMachineCronJobRelateRepo(), ioc.WithComponentName("MachineCronJobRelateRepo"))
	ioc.Register(newMachineTermOpRepoImpl(), ioc.WithComponentName("MachineTermOpRepo"))
}

func GetMachineRepo() repository.Machine {
	return ioc.Get[repository.Machine]("MachineRepo")
}

func GetMachineFileRepo() repository.MachineFile {
	return ioc.Get[repository.MachineFile]("MachineFileRepo")
}

func GetMachineScriptRepo() repository.MachineScript {
	return ioc.Get[repository.MachineScript]("MachineScriptRepo")
}

func GetAuthCertRepo() repository.AuthCert {
	return ioc.Get[repository.AuthCert]("AuthCertRepo")
}

func GetMachineCronJobRepo() repository.MachineCronJob {
	return ioc.Get[repository.MachineCronJob]("MachineCronJobRepo")
}

func GetMachineCronJobExecRepo() repository.MachineCronJobExec {
	return ioc.Get[repository.MachineCronJobExec]("MachineCronJobExecRepo")
}

func GetMachineCronJobRelateRepo() repository.MachineCronJobRelate {
	return ioc.Get[repository.MachineCronJobRelate]("MachineCropJobRelateRepo")
}

func GetMachineTermOpRepo() repository.MachineTermOp {
	return ioc.Get[repository.MachineTermOp]("MachineTermOpRepo")
}
