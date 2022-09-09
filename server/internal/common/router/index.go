package router

import (
	"mayfly-go/internal/common/api"
	dbapp "mayfly-go/internal/db/application"
	machineapp "mayfly-go/internal/machine/application"
	projectapp "mayfly-go/internal/project/application"
	redisapp "mayfly-go/internal/redis/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitIndexRouter(router *gin.RouterGroup) {
	index := router.Group("common/index")
	i := &api.Index{
		ProjectApp: projectapp.GetProjectApp(),
		MachineApp: machineapp.GetMachineApp(),
		DbApp:      dbapp.GetDbApp(),
		RedisApp:   redisapp.GetRedisApp(),
	}
	{
		// 首页基本信息统计
		index.GET("count", func(g *gin.Context) {
			ctx.NewReqCtxWithGin(g).
				Handle(i.Count)
		})
	}
}
