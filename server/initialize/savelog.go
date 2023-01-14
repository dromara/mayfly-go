package initialize

import (
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/req"
)

func InitSaveLogFunc() req.SaveLogFunc {
	return sysapp.GetSyslogApp().SaveFromReq
}
