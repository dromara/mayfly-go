package starter

import (
	"mayfly-go/base/global"
	"mayfly-go/devops/initialize"
)

func RunServer() {
	web := initialize.InitRouter()
	port := global.Config.Server.GetPort()
	if app := global.Config.App; app != nil {
		global.Log.Infof("%s- Listening and serving HTTP on %s", app.GetAppInfo(), port)
	} else {
		global.Log.Infof("Listening and serving HTTP on %s", port)
	}
	web.Run(port)
}
