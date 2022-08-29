package initialize

import (
	"fmt"
	"io/fs"
	common_router "mayfly-go/internal/common/router"
	devops_router "mayfly-go/internal/devops/router"
	sys_router "mayfly-go/internal/sys/router"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/middleware"
	"mayfly-go/static"
	"net/http"

	"github.com/gin-gonic/gin"
)

func WrapStaticHandler(h http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", `public, max-age=31536000`)
		h.ServeHTTP(c.Writer, c.Request)
	}
}

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

	// 使用embed打包静态资源至二进制文件中
	fsys, _ := fs.Sub(static.Static, "static")
	fileServer := http.FileServer(http.FS(fsys))
	handler := WrapStaticHandler(fileServer)
	router.GET("/", handler)
	router.GET("/favicon.ico", handler)
	router.GET("/config.js", handler)
	// 所有/assets/**开头的都是静态资源文件
	router.GET("/assets/*file", handler)

	// 设置静态资源
	if staticConfs := serverConfig.Static; staticConfs != nil {
		for _, scs := range *staticConfs {
			router.StaticFS(scs.RelativePath, http.Dir(scs.Root))
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
		common_router.InitIndexRouter(api)
		common_router.InitCommonRouter(api)

		sys_router.InitCaptchaRouter(api)
		sys_router.InitAccountRouter(api) // 注册account路由
		sys_router.InitResourceRouter(api)
		sys_router.InitRoleRouter(api)
		sys_router.InitSystemRouter(api)
		sys_router.InitSyslogRouter(api)
		sys_router.InitSysConfigRouter(api)

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
