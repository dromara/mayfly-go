package initialize

import (
	dbInit "mayfly-go/internal/db/init"
	machineInit "mayfly-go/internal/machine/init"
)

func InitOther() {
	machineInit.Init()
	dbInit.Init()
}
