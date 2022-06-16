package initialize

import (
	"fmt"
	common_index_router "mayfly-go/internal/common/router"
	devops_router "mayfly-go/internal/devops/router"
	sys_router "mayfly-go/internal/sys/router"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/middleware"
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
		common_index_router.InitIndexRouter(api)

		sys_router.InitCaptchaRouter(api)
		sys_router.InitAccountRouter(api) // 注册account路由
		sys_router.InitResourceRouter(api)
		sys_router.InitRoleRouter(api)
		sys_router.InitSystemRouter(api)

		devops_router.InitProjectRouter(api)
		devops_router.InitDbRouter(api)
		devops_router.InitDbSqlExecRouter(api)
		devops_router.InitRedisRouter(api)
		devops_router.InitMachineRouter(api)
		devops_router.InitMachineScriptRouter(api)
		devops_router.InitMachineFileRouter(api)
		devops_router.InitMongoRouter(api)
	}

	return router
}
