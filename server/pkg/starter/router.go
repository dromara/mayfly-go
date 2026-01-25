package starter

import (
	"fmt"
	"io/fs"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"
	"net/http"

	"github.com/gin-gonic/gin"
)

// StaticRouter 静态资源路由配置
type StaticRouter struct {
	Fs    fs.FS    // 静态资源文件系统
	Paths []string // 静态资源访问路径，如 /assets/*file
}

func initRouter(router *gin.Engine, conf req.RouterConfig) *gin.Engine {
	// 没有路由即 404返回
	router.NoRoute(func(g *gin.Context) {
		g.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": fmt.Sprintf("not found '%s:%s'", g.Request.Method, g.Request.URL.Path)})
	})

	// 设置路由组
	api := router.Group(conf.ContextPath + "/api")

	// 获取所有实现了RouterApi接口的实例，并注册对应路由
	ras := ioc.GetBeansByType[req.RouterApi]()
	for _, ra := range ras {
		confs := ra.ReqConfs()
		if group := confs.Group; group != "" {
			req.BatchSetGroup(api.Group(group), confs.Confs)
		} else {
			req.BatchSetGroup(api, confs.Confs)
		}
	}

	return router
}
