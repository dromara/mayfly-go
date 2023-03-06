package persistence

import "mayfly-go/internal/machine/domain/repository"

var (
	machineRepo       repository.Machine       = newMachineRepo()
	machineFileRepo   repository.MachineFile   = newMachineFileRepo()
	machineScriptRepo repository.MachineScript = newMachineScriptRepo()
	authCertRepo                               = newAuthCertRepo()
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
