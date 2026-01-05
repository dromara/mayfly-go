package initialize

import (
	"fmt"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RouterApi
// 该接口的实现类注册到ioc中，则会自动将请求配置注册到路由中
type RouterApi interface {
	// ReqConfs 获取请求配置信息
	ReqConfs() *req.Confs
}

type RouterConfig struct {
	ContextPath string // 请求路径上下文
}

func InitRouter(router *gin.Engine, conf RouterConfig) *gin.Engine {
	// 没有路由即 404返回
	router.NoRoute(func(g *gin.Context) {
		g.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": fmt.Sprintf("not found '%s:%s'", g.Request.Method, g.Request.URL.Path)})
	})

	// 设置路由组
	api := router.Group(conf.ContextPath + "/api")

	// 获取所有实现了RouterApi接口的实例，并注册对应路由
	ras := ioc.GetBeansByType[RouterApi]()
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
