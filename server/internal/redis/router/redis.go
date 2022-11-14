package router

import (
	"mayfly-go/internal/redis/api"
	"mayfly-go/internal/redis/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitRedisRouter(router *gin.RouterGroup) {
	redis := router.Group("redis")
	{
		rs := &api.Redis{
			RedisApp: application.GetRedisApp(),
			TagApp:   tagapp.GetTagTreeApp(),
		}

		// 获取redis list
		redis.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.RedisList)
		})

		save := ctx.NewLogInfo("保存redis信息").WithSave(true)
		redis.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(save).Handle(rs.Save)
		})

		redis.GET(":id/pwd", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.GetRedisPwd)
		})

		delRedis := ctx.NewLogInfo("删除redis信息").WithSave(true)
		redis.DELETE(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(delRedis).Handle(rs.DeleteRedis)
		})

		redis.GET(":id/info", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.RedisInfo)
		})

		redis.GET(":id/cluster-info", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.ClusterInfo)
		})

		// 获取指定redis keys
		redis.POST(":id/:db/scan", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.Scan)
		})

		// 删除key
		deleteKeyL := ctx.NewLogInfo("redis-删除key").WithSave(true)
		redis.DELETE(":id/:db/key", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(deleteKeyL).Handle(rs.DeleteKey)
		})

		// 获取string类型值
		redis.GET(":id/:db/string-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.GetStringValue)
		})

		// 设置string类型值
		setStringL := ctx.NewLogInfo("redis-setString").WithSave(true)
		redis.POST(":id/:db/string-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(setStringL).Handle(rs.SetStringValue)
		})

		// hscan
		redis.GET(":id/:db/hscan", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.Hscan)
		})

		redis.GET(":id/:db/hget", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.Hget)
		})

		hdelL := ctx.NewLogInfo("redis-hdel").WithSave(true)
		redis.DELETE(":id/:db/hdel", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(hdelL).Handle(rs.Hdel)
		})

		// 设置hash类型值
		setHashValueL := ctx.NewLogInfo("redis-setHashValue").WithSave(true)
		redis.POST(":id/:db/hash-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(setHashValueL).Handle(rs.SetHashValue)
		})

		// 获取set类型值
		redis.GET(":id/:db/set-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.GetSetValue)
		})

		// 设置set类型值
		redis.POST(":id/:db/set-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.SetSetValue)
		})

		// 获取list类型值
		redis.GET(":id/:db/list-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.GetListValue)
		})

		redis.POST(":id/:db/list-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.SaveListValue)
		})

		redis.POST(":id/:db/list-value/lset", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.SetListValue)
		})
	}
}
