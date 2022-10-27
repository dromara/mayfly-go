package router

import (
	"mayfly-go/internal/common/api"
	dbapp "mayfly-go/internal/db/application"
	machineapp "mayfly-go/internal/machine/application"
	mongoapp "mayfly-go/internal/mongo/application"
	redisapp "mayfly-go/internal/redis/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitIndexRouter(router *gin.RouterGroup) {
	index := router.Group("common/index")
	i := &api.Index{
		TagApp:     tagapp.GetTagTreeApp(),
		MachineApp: machineapp.GetMachineApp(),
		DbApp:      dbapp.GetDbApp(),
		RedisApp:   redisapp.GetRedisApp(),
		MongoApp:   mongoapp.GetMongoApp(),
	}
	{
		// 首页基本信息统计
		index.GET("count", func(g *gin.Context) {
			ctx.NewReqCtxWithGin(g).
				Handle(i.Count)
		})
	}
}
