package starter

import (
	"mayfly-go/base/config"
	"mayfly-go/base/global"

	"github.com/gin-gonic/gin"
)

func RunWebServer(web *gin.Engine) {
	server := config.Conf.Server
	port := server.GetPort()
	if app := config.Conf.App; app != nil {
		global.Log.Infof("%s- Listening and serving HTTP on %s", app.GetAppInfo(), port)
	} else {
		global.Log.Infof("Listening and serving HTTP on %s", port)
	}

	if server.Tls != nil && server.Tls.Enable {
		web.RunTLS(port, server.Tls.CertFile, server.Tls.KeyFile)
	} else {
		web.Run(port)
	}
}
