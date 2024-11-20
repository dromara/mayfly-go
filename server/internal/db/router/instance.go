package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitInstanceRouter(router *gin.RouterGroup) {
	instances := router.Group("/instances")

	d := new(api.Instance)
	biz.ErrIsNil(ioc.Inject(d))

	reqs := [...]*req.Conf{
		// 获取数据库列表
		req.NewGet("", d.Instances),

		req.NewPost("/test-conn", d.TestConn),

		req.NewPost("", d.SaveInstance).Log(req.NewLogSaveI(imsg.LogDbInstSave)),

		req.NewGet(":instanceId", d.GetInstance),

		// 获取数据库实例的所有数据库名
		req.NewPost("/databases", d.GetDatabaseNames),

		// 根据授权凭证名获取其所有库名
		req.NewGet("/databases/:ac", d.GetDatabaseNamesByAc),

		req.NewGet(":instanceId/server-info", d.GetDbServer),

		req.NewDelete(":instanceId", d.DeleteInstance).Log(req.NewLogSaveI(imsg.LogDbInstDelete)),
	}

	req.BatchSetGroup(instances, reqs[:])
}
