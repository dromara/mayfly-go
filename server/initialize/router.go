package initialize

import (
	"fmt"
	"io/fs"
	auth_router "mayfly-go/internal/auth/router"
	common_router "mayfly-go/internal/common/router"
	db_router "mayfly-go/internal/db/router"
	machine_router "mayfly-go/internal/machine/router"
	mongo_router "mayfly-go/internal/mongo/router"
	msg_router "mayfly-go/internal/msg/router"
	redis_router "mayfly-go/internal/redis/router"
	sys_router "mayfly-go/internal/sys/router"
	tag_router "mayfly-go/internal/tag/router"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/middleware"
	"mayfly-go/static"
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
	setStatic(router)

	// 是否允许跨域
	if serverConfig.Cors {
		router.Use(middleware.Cors())
	}

	// 设置路由组
	api := router.Group("/api")
	{
		common_router.Init(api)

		auth_router.Init(api)

		sys_router.Init(api)
		msg_router.Init(api)

		tag_router.Init(api)
		machine_router.Init(api)
		db_router.Init(api)
		redis_router.Init(api)
		mongo_router.Init(api)
	}

	return router
}

func setStatic(router *gin.Engine) {
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
	if staticConfs := config.Conf.Server.Static; staticConfs != nil {
		for _, scs := range *staticConfs {
			router.StaticFS(scs.RelativePath, http.Dir(scs.Root))
		}
	}
	// 设置静态文件
	if staticFileConfs := config.Conf.Server.StaticFile; staticFileConfs != nil {
		for _, sfs := range *staticFileConfs {
			router.StaticFile(sfs.RelativePath, sfs.Filepath)
		}
	}
}

func WrapStaticHandler(h http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", `public, max-age=31536000`)
		h.ServeHTTP(c.Writer, c.Request)
	}
}
