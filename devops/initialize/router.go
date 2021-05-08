package initialize

import (
	"mayfly-go/base/global"
	"mayfly-go/base/middleware"
	"mayfly-go/devops/routers"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	// server配置
	serverConfig := global.Config.Server
	gin.SetMode(serverConfig.Model)

	var router = gin.New()
	router.MaxMultipartMemory = 8 << 20
	// 设置静态资源
	if staticConfs := serverConfig.Static; staticConfs != nil {
		for _, scs := range *staticConfs {
			router.Static(scs.RelativePath, scs.Root)
		}

	}
	// 设置静态文件
	if staticFileConfs := serverConfig.StaticFile; staticFileConfs != nil {
		for _, sfs := range *staticFileConfs {
			router.StaticFile(sfs.RelativePath, sfs.Filepath)
		}
	}
	// 是否允许跨域
	if serverConfig.Cors {
		router.Use(middleware.Cors())
	}

	// 设置路由组
	api := router.Group("/api")
	{
		routers.InitAccountRouter(api) // 注册account路由
		routers.InitDbRouter(api)
		routers.InitMachineRouter(api)
		routers.InitMachineScriptRouter(api)
		routers.InitMachineFileRouter(api)
	}

	return router
}
