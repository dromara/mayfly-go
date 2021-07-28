package initialize

import (
	"fmt"
	"mayfly-go/base/config"
	"mayfly-go/base/middleware"
	devops_routers "mayfly-go/server/devops/routers"
	sys_routers "mayfly-go/server/sys/routers"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	// server配置
	serverConfig := config.Conf.Server
	gin.SetMode(serverConfig.Model)

	var router = gin.New()
	router.MaxMultipartMemory = 8 << 20

	// 没有路由即 404返回
	router.NoRoute(func(g *gin.Context) {
		g.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": fmt.Sprintf("not found '%s:%s'", g.Request.Method, g.Request.URL.Path)})
	})

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
		sys_routers.InitCaptchaRouter(api)

		sys_routers.InitAccountRouter(api) // 注册account路由
		sys_routers.InitResourceRouter(api)
		sys_routers.InitRoleRouter(api)

		devops_routers.InitProjectRouter(api)
		devops_routers.InitDbRouter(api)
		devops_routers.InitRedisRouter(api)
		devops_routers.InitMachineRouter(api)
		devops_routers.InitMachineScriptRouter(api)
		devops_routers.InitMachineFileRouter(api)
	}

	return router
}
