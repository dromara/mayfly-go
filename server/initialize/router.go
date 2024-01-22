package initialize

import (
	"fmt"
	"io/fs"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/middleware"
	"mayfly-go/static"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 初始化路由函数
type InitRouterFunc func(router *gin.RouterGroup)

var (
	initRouterFuncs = make([]InitRouterFunc, 0)
)

// 添加初始化路由函数，由各个默认自行添加
func AddInitRouterFunc(initRouterFunc InitRouterFunc) {
	initRouterFuncs = append(initRouterFuncs, initRouterFunc)
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

	// 设置静态资源
	setStatic(serverConfig.ContextPath, router)

	// 是否允许跨域
	if serverConfig.Cors {
		router.Use(middleware.Cors())
	}

	// 设置路由组
	api := router.Group(serverConfig.ContextPath + "/api")
	// 调用所有模块注册的初始化路由函数
	for _, initRouterFunc := range initRouterFuncs {
		initRouterFunc(api)
	}
	initRouterFuncs = nil

	return router
}

func setStatic(contextPath string, router *gin.Engine) {
	// 使用embed打包静态资源至二进制文件中
	fsys, _ := fs.Sub(static.Static, "static")
	fileServer := http.FileServer(http.FS(fsys))
	handler := WrapStaticHandler(http.StripPrefix(contextPath, fileServer))

	router.GET(contextPath+"/", handler)
	router.GET(contextPath+"/favicon.ico", handler)
	router.GET(contextPath+"/config.js", handler)
	// 所有/assets/**开头的都是静态资源文件
	router.GET(contextPath+"/assets/*file", handler)

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
