package application

import (
	"mayfly-go/internal/machine/infrastructure/persistence"
)

var (
	machineApp Machine = newMachineApp(
		persistence.GetMachineRepo(),
		GetAuthCertApp(),
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
