package router

import (
	"mayfly-go/internal/redis/api"
	"mayfly-go/internal/redis/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"

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
			req.NewCtxWithGin(c).Handle(rs.RedisList)
		})

		save := req.NewLogInfo("redis-保存信息").WithSave(true)
		redis.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(save).Handle(rs.Save)
		})

		redis.GET(":id/pwd", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.GetRedisPwd)
		})

		delRedis := req.NewLogInfo("redis-删除信息").WithSave(true)
		redis.DELETE(":id", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(delRedis).Handle(rs.DeleteRedis)
		})

		redis.GET(":id/info", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.RedisInfo)
		})

		redis.GET(":id/cluster-info", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.ClusterInfo)
		})

		// 获取指定redis keys
		redis.POST(":id/:db/scan", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.Scan)
		})

		// 删除key
		deleteKeyL := req.NewLogInfo("redis-删除key").WithSave(true)
		deleteKeyP := req.NewPermission("redis:data:del")
		redis.DELETE(":id/:db/key", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(deleteKeyL).WithRequiredPermission(deleteKeyP).Handle(rs.DeleteKey)
		})

		// 保存数据权限
		saveDataP := req.NewPermission("redis:data:save")

		// 获取string类型值
		redis.GET(":id/:db/string-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.GetStringValue)
		})

		// 设置string类型值
		setStringL := req.NewLogInfo("redis-setString").WithSave(true)
		redis.POST(":id/:db/string-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(setStringL).WithRequiredPermission(saveDataP).Handle(rs.SetStringValue)
		})

		// hscan
		redis.GET(":id/:db/hscan", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.Hscan)
		})

		redis.GET(":id/:db/hget", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.Hget)
		})

		hdelL := req.NewLogInfo("redis-hdel").WithSave(true)
		redis.DELETE(":id/:db/hdel", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(hdelL).WithRequiredPermission(deleteKeyP).Handle(rs.Hdel)
		})

		// 设置hash类型值
		setHashValueL := req.NewLogInfo("redis-setHashValue").WithSave(true)
		redis.POST(":id/:db/hash-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(setHashValueL).WithRequiredPermission(saveDataP).Handle(rs.SetHashValue)
		})

		// 获取set类型值
		redis.GET(":id/:db/set-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.GetSetValue)
		})

		// 设置set类型值
		redis.POST(":id/:db/set-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithRequiredPermission(saveDataP).Handle(rs.SetSetValue)
		})

		// 获取list类型值
		redis.GET(":id/:db/list-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.GetListValue)
		})

		redis.POST(":id/:db/list-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.SaveListValue)
		})

		redis.POST(":id/:db/list-value/lset", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.SetListValue)
		})
	}
}
