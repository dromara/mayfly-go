package application

import "mayfly-go/internal/machine/infrastructure/persistence"

var (
	machineApp       Machine       = newMachineApp(persistence.GetMachineRepo())
	machineFileApp   MachineFile   = newMachineFileApp(persistence.GetMachineFileRepo(), persistence.GetMachineRepo())
	machineScriptApp MachineScript = newMachineScriptApp(persistence.GetMachineScriptRepo(), persistence.GetMachineRepo())
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
