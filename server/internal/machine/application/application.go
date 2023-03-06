package application

import (
	"mayfly-go/internal/machine/infrastructure/persistence"
)

var (
	machineFileApp MachineFile = newMachineFileApp(persistence.GetMachineFileRepo(), persistence.GetMachineRepo())

	machineScriptApp MachineScript = newMachineScriptApp(persistence.GetMachineScriptRepo(), persistence.GetMachineRepo())

	authCertApp AuthCert = newAuthCertApp(persistence.GetAuthCertRepo())

	machineApp Machine = newMachineApp(
		persistence.GetMachineRepo(),
		GetAuthCertApp(),
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
