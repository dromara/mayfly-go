package initialize

import (
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/ctx"
)

func InitSaveLogFunc() ctx.SaveLogFunc {
	return sysapp.GetSyslogApp().SaveFromReq
}
