package router

import (
	"mayfly-go/internal/redis/api"
	"mayfly-go/internal/redis/imsg"
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

	reqs := [...]*req.Conf{
		req.NewGet("dashbord", dashbord.Dashbord),

		// 获取redis list
		req.NewGet("", rs.RedisList),

		req.NewPost("/test-conn", rs.TestConn),

		req.NewPost("", rs.Save).Log(req.NewLogSaveI(imsg.LogRedisSave)),

		req.NewDelete(":id", rs.DeleteRedis).Log(req.NewLogSaveI(imsg.LogRedisDelete)),

		req.NewGet("/:id/info", rs.RedisInfo),

		req.NewGet(":id/cluster-info", rs.ClusterInfo),

		req.NewPost(":id/:db/run-cmd", rs.RunCmd).Log(req.NewLogSaveI(imsg.LogRedisRunCmd)),

		// 获取指定redis keys
		req.NewPost(":id/:db/scan", rs.ScanKeys),

		req.NewGet(":id/:db/key-info", rs.KeyInfo),

		req.NewGet(":id/:db/key-ttl", rs.TtlKey),

		req.NewGet(":id/:db/key-memuse", rs.MemoryUsage),
	}

	req.BatchSetGroup(redis, reqs[:])
}
