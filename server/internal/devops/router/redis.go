package router

import (
	"mayfly-go/internal/devops/api"
	"mayfly-go/internal/devops/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitRedisRouter(router *gin.RouterGroup) {
	redis := router.Group("redis")
	{
		rs := &api.Redis{
			RedisApp:   application.RedisApp,
			ProjectApp: application.ProjectApp,
		}

		// 获取redis list
		redis.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.RedisList)
		})

		save := ctx.NewLogInfo("保存redis信息").WithSave(true)
		redis.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(save).Handle(rs.Save)
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
		redis.POST(":id/scan", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.Scan)
		})

		// 删除key
		deleteKeyL := ctx.NewLogInfo("redis删除key").WithSave(true)
		redis.DELETE(":id/key", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(deleteKeyL).Handle(rs.DeleteKey)
		})

		// 获取string类型值
		redis.GET(":id/string-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.GetStringValue)
		})

		// 设置string类型值
		redis.POST(":id/string-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.SetStringValue)
		})

		// 获取hash类型值
		redis.GET(":id/hash-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.GetHashValue)
		})

		// 设置hash类型值
		redis.POST(":id/hash-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.SetHashValue)
		})

		// 获取set类型值
		redis.GET(":id/set-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.GetSetValue)
		})

		// 设置set类型值
		redis.POST(":id/set-value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(rs.SetSetValue)
		})
	}
}
