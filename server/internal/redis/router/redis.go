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

		redis.GET(":id/:db/key-ttl", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.TtlKey)
		})

		// 保存数据权限
		saveDataP := req.NewPermission("redis:data:save")
		// 删除数据权限
		deleteDataP := req.NewPermission("redis:data:del")

		// 删除key
		deleteKeyL := req.NewLogInfo("redis-删除key").WithSave(true)
		redis.DELETE(":id/:db/key", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(deleteKeyL).
				WithRequiredPermission(deleteDataP).
				Handle(rs.DeleteKey)
		})

		renameKeyL := req.NewLogInfo("redis-重命名key").WithSave(true)
		redis.POST(":id/:db/rename-key", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(renameKeyL).
				WithRequiredPermission(saveDataP).
				Handle(rs.RenameKey)
		})

		expireKeyL := req.NewLogInfo("redis-设置key过期时间").WithSave(true)
		redis.POST(":id/:db/expire-key", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(expireKeyL).
				WithRequiredPermission(saveDataP).
				Handle(rs.ExpireKey)
		})

		persistKeyL := req.NewLogInfo("redis-移除key过期时间").WithSave(true)
		redis.DELETE(":id/:db/persist-key", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(persistKeyL).
				WithRequiredPermission(saveDataP).
				Handle(rs.PersistKey)
		})

		// 获取string类型值
		redis.GET(":id/:db/string-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.GetStringValue)
		})

		// 设置string类型值
		setStringL := req.NewLogInfo("redis-setString").WithSave(true)
		redis.POST(":id/:db/string-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(setStringL).
				WithRequiredPermission(saveDataP).
				Handle(rs.SetStringValue)
		})

		// hscan
		redis.GET(":id/:db/hscan", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.Hscan)
		})

		redis.GET(":id/:db/hget", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.Hget)
		})

		hsetL := req.NewLogInfo("redis-hset").WithSave(true)
		redis.POST(":id/:db/hset", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(hsetL).
				WithRequiredPermission(saveDataP).
				Handle(rs.Hset)
		})

		hdelL := req.NewLogInfo("redis-hdel").WithSave(true)
		redis.DELETE(":id/:db/hdel", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(hdelL).
				WithRequiredPermission(deleteDataP).
				Handle(rs.Hdel)
		})

		// 设置hash类型值
		setHashValueL := req.NewLogInfo("redis-setHashValue").WithSave(true)
		redis.POST(":id/:db/hash-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(setHashValueL).
				WithRequiredPermission(saveDataP).
				Handle(rs.SetHashValue)
		})

		// set操作
		redis.GET(":id/:db/set-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.GetSetValue)
		})

		redis.POST(":id/:db/set-value", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithRequiredPermission(saveDataP).
				Handle(rs.SetSetValue)
		})

		redis.GET(":id/:db/scard", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.Scard)
		})

		redis.POST(":id/:db/sscan", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.Sscan)
		})

		redis.POST(":id/:db/sadd", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithRequiredPermission(saveDataP).
				Handle(rs.Sadd)
		})

		redis.POST(":id/:db/srem", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithRequiredPermission(deleteDataP).
				Handle(rs.Srem)
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

		redis.POST(":id/:db/lrem", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithRequiredPermission(deleteDataP).
				Handle(rs.Lrem)
		})

		// zset操作
		redis.GET(":id/:db/zcard", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.ZCard)
		})

		redis.GET(":id/:db/zscan", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.ZScan)
		})

		redis.GET(":id/:db/zrevrange", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(rs.ZRevRange)
		})

		redis.POST(":id/:db/zrem", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithRequiredPermission(deleteDataP).
				Handle(rs.ZRem)
		})

		redis.POST(":id/:db/zadd", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithRequiredPermission(saveDataP).
				Handle(rs.ZAdd)
		})
	}
}
