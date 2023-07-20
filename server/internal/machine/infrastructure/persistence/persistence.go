package persistence

import "mayfly-go/internal/machine/domain/repository"

var (
	machineRepo              repository.Machine              = newMachineRepo()
	machineFileRepo          repository.MachineFile          = newMachineFileRepo()
	machineScriptRepo        repository.MachineScript        = newMachineScriptRepo()
	authCertRepo             repository.AuthCert             = newAuthCertRepo()
	machineCropJobRepo       repository.MachineCronJob       = newMachineCronJobRepo()
	machineCropJobExecRepo   repository.MachineCronJobExec   = newMachineCronJobExecRepo()
	machineCronJobRelateRepo repository.MachineCronJobRelate = newMachineCropJobRelateRepo()
)

func GetMachineRepo() repository.Machine {
	return machineRepo
}

func GetMachineFileRepo() repository.MachineFile {
	return machineFileRepo
}

func GetMachineScriptRepo() repository.MachineScript {
	return machineScriptRepo
}

func GetAuthCertRepo() repository.AuthCert {
	return authCertRepo
}

func GetMachineCronJobRepo() repository.MachineCronJob {
	return machineCropJobRepo
}

func GetMachineCronJobExecRepo() repository.MachineCronJobExec {
	return machineCropJobExecRepo
}

func GetMachineCronJobRelateRepo() repository.MachineCronJobRelate {
	return machineCronJobRelateRepo
}
