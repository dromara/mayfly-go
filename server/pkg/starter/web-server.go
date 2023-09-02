package starter

import (
	"mayfly-go/initialize"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func runWebServer() {
	// 设置gin日志输出器
	logOut := logx.GetConfig().GetLogOut()
	gin.DefaultErrorWriter = logOut
	gin.DefaultWriter = logOut

	// 权限处理器
	req.UseBeforeHandlerInterceptor(req.PermissionHandler)
	// 日志处理器
	req.UseAfterHandlerInterceptor(req.LogHandler)
	// 设置日志保存函数
	req.SetSaveLogFunc(initialize.InitSaveLogFunc())

	// 注册路由
	web := initialize.InitRouter()

	server := config.Conf.Server
	port := server.GetPort()
	logx.Infof("Listening and serving HTTP on %s", port)

	var err error
	if server.Tls != nil && server.Tls.Enable {
		err = web.RunTLS(port, server.Tls.CertFile, server.Tls.KeyFile)
	} else {
		err = web.Run(port)
	}
	biz.ErrIsNilAppendErr(err, "服务启动失败: %s")
}
