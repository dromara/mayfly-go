package initialize

import (
	sys_application "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/ctx"
)

func InitSaveLogFunc() ctx.SaveLogFunc {
	return sys_application.SyslogApp.SaveFromReq
}
