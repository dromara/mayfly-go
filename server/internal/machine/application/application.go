package application

import (
	"mayfly-go/internal/machine/infrastructure/persistence"
	tagapp "mayfly-go/internal/tag/application"
)

var (
	machineApp Machine = newMachineApp(
		persistence.GetMachineRepo(),
		GetAuthCertApp(),
		tagapp.GetTagTreeApp(),
	)

	machineFileApp MachineFile = newMachineFileApp(
		persistence.GetMachineFileRepo(),
		GetMachineApp(),
	)

	machineScriptApp MachineScript = newMachineScriptApp(
		persistence.GetMachineScriptRepo(),
		GetMachineApp(),
	)

	authCertApp AuthCert = newAuthCertApp(persistence.GetAuthCertRepo())

	machineCropJobApp MachineCronJob = newMachineCronJobApp(
		persistence.GetMachineCronJobRepo(),
		persistence.GetMachineCronJobRelateRepo(),
		persistence.GetMachineCronJobExecRepo(),
		GetMachineApp(),
	)

	machineTermOpApp MachineTermOp = newMachineTermOpApp(persistence.GetMachineTermOpRepo())
)

func GetMachineApp() Machine {
	return machineApp
}

func GetMachineFileApp() MachineFile {
	return machineFileApp
}

func GetMachineScriptApp() MachineScript {
	return machineScriptApp
}

func GetAuthCertApp() AuthCert {
	return authCertApp
}

func GetMachineCronJobApp() MachineCronJob {
	return machineCropJobApp
}

func GetMachineTermOpApp() MachineTermOp {
	return machineTermOpApp
}
