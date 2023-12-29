package initialize

import (
	dbApp "mayfly-go/internal/db/application"
	machineInit "mayfly-go/internal/machine/init"
)

func InitOther() {
	machineInit.Init()
	dbApp.Init()
}
