package router

import (
	"mayfly-go/internal/common/api"
	devops_app "mayfly-go/internal/devops/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitIndexRouter(router *gin.RouterGroup) {
	index := router.Group("common/index")
	i := &api.Index{
		ProjectApp: devops_app.ProjectApp,
		MachineApp: devops_app.MachineApp,
		DbApp:      devops_app.DbApp,
		RedisApp:   devops_app.RedisApp,
	}
	{
		// 首页基本信息统计
		index.GET("count", func(g *gin.Context) {
			ctx.NewReqCtxWithGin(g).
				Handle(i.Count)
		})
	}
}
