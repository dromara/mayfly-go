package starter

import (
	"mayfly-go/initialize"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/ctx"
	"mayfly-go/pkg/global"
)

func runWebServer() {
	// 权限处理器
	ctx.UseBeforeHandlerInterceptor(ctx.PermissionHandler)
	// 日志处理器
	ctx.UseAfterHandlerInterceptor(ctx.LogHandler)
	// 设置日志保存函数
	ctx.SetSaveLogFunc(initialize.InitSaveLogFunc())

	// 注册路由
	web := initialize.InitRouter()

	server := config.Conf.Server
	port := server.GetPort()
	global.Log.Infof("Listening and serving HTTP on %s", port)

	var err error
	if server.Tls != nil && server.Tls.Enable {
		err = web.RunTLS(port, server.Tls.CertFile, server.Tls.KeyFile)
	} else {
		err = web.Run(port)
	}
	biz.ErrIsNilAppendErr(err, "服务启动失败: %s")
}
