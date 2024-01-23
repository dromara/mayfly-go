package router

import (
	"mayfly-go/internal/redis/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitRedisRouter(router *gin.RouterGroup) {
	redis := router.Group("redis")

	rs := new(api.Redis)
	biz.ErrIsNil(ioc.Inject(rs))

	dashbord := new(api.Dashbord)
	biz.ErrIsNil(ioc.Inject(dashbord))

	// 保存数据权限
	saveDataP := req.NewPermission("redis:data:save")
	// 删除数据权限
	deleteDataP := req.NewPermission("redis:data:del")

	reqs := [...]*req.Conf{
		req.NewGet("dashbord", dashbord.Dashbord),

		// 获取redis list
		req.NewGet("", rs.RedisList),

		req.NewPost("/test-conn", rs.TestConn),

		req.NewPost("", rs.Save).Log(req.NewLogSave("redis-保存信息")),

		req.NewGet(":id/pwd", rs.GetRedisPwd),

		req.NewDelete(":id", rs.DeleteRedis).Log(req.NewLogSave("redis-删除信息")),

		req.NewGet("/:id/info", rs.RedisInfo),

		req.NewGet(":id/cluster-info", rs.ClusterInfo),

		// 获取指定redis keys
		req.NewPost(":id/:db/scan", rs.ScanKeys),

		req.NewGet(":id/:db/key-info", rs.KeyInfo),

		req.NewGet(":id/:db/key-ttl", rs.TtlKey),

		req.NewGet(":id/:db/key-memuse", rs.MemoryUsage),

		req.NewDelete(":id/:db/key", rs.DeleteKey).Log(req.NewLogSave("redis-删除key")).RequiredPermission(deleteDataP),

		req.NewPost(":id/:db/rename-key", rs.RenameKey).Log(req.NewLogSave("redis-重命名key")).RequiredPermission(saveDataP),

		req.NewPost(":id/:db/expire-key", rs.ExpireKey).Log(req.NewLogSave("redis-设置key过期时间")).RequiredPermission(saveDataP),

		req.NewDelete(":id/:db/persist-key", rs.PersistKey).Log(req.NewLogSave("redis-移除key过期时间")).RequiredPermission(saveDataP),

		req.NewDelete(":id/:db/flushdb", rs.FlushDb).Log(req.NewLogSave("redis-flushdb")).RequiredPermission(deleteDataP),

		// 获取string类型值
		req.NewGet(":id/:db/string-value", rs.GetStringValue),

		// 设置string类型值
		req.NewPost(":id/:db/string-value", rs.SaveStringValue).Log(req.NewLogSave("redis-setString")).RequiredPermission(saveDataP),

		// ———————————————— hash操作 ————————————————
		req.NewGet(":id/:db/hscan", rs.Hscan),

		req.NewGet(":id/:db/hget", rs.Hget),

		req.NewPost(":id/:db/hset", rs.Hset).Log(req.NewLogSave("redis-hset")).RequiredPermission(saveDataP),

		req.NewDelete(":id/:db/hdel", rs.Hdel).Log(req.NewLogSave("redis-hdel")).RequiredPermission(deleteDataP),

		// 设置hash类型值
		req.NewPost(":id/:db/hash-value", rs.SaveHashValue).Log(req.NewLogSave("redis-setHashValue")).RequiredPermission(saveDataP),

		// ---------------  set操作  ----------------
		req.NewGet(":id/:db/set-value", rs.GetSetValue),

		req.NewPost(":id/:db/set-value", rs.SaveSetValue).RequiredPermission(saveDataP),

		req.NewGet(":id/:db/scard", rs.Scard),

		req.NewPost(":id/:db/sscan", rs.Sscan),

		req.NewPost(":id/:db/sadd", rs.Sadd).RequiredPermission(saveDataP),

		req.NewPost(":id/:db/srem", rs.Srem).RequiredPermission(deleteDataP),

		// --------------- list操作  ----------------
		req.NewGet(":id/:db/list-value", rs.GetListValue),

		req.NewPost(":id/:db/list-value", rs.SaveListValue).RequiredPermission(saveDataP),

		req.NewPost(":id/:db/list-value/lset", rs.Lset).RequiredPermission(saveDataP),

		req.NewPost(":id/:db/lrem", rs.Lrem).RequiredPermission(deleteDataP),

		// --------------- zset操作  ----------------
		req.NewGet(":id/:db/zcard", rs.ZCard),

		req.NewGet(":id/:db/zscan", rs.ZScan),

		req.NewGet(":id/:db/zrevrange", rs.ZRevRange),

		req.NewPost(":id/:db/zrem", rs.ZRem).Log(req.NewLogSave("redis-zrem")).RequiredPermission(deleteDataP),

		req.NewPost(":id/:db/zadd", rs.ZAdd).RequiredPermission(saveDataP),
	}

	req.BatchSetGroup(redis, reqs[:])

}
