package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitInstanceRouter(router *gin.RouterGroup) {
	instances := router.Group("/instances")

	d := &api.Instance{
		InstanceApp: application.GetInstanceApp(),
		DbApp:       application.GetDbApp(),
	}

	reqs := [...]*req.Conf{
		// 获取数据库列表
		req.NewGet("", d.Instances),

		req.NewPost("/test-conn", d.TestConn),

		req.NewPost("", d.SaveInstance).Log(req.NewLogSave("db-保存数据库实例信息")),

		req.NewGet(":instanceId", d.GetInstance),

		req.NewGet(":instanceId/pwd", d.GetInstancePwd),

		// 获取数据库实例的所有数据库名
		req.NewGet(":instanceId/databases", d.GetDatabaseNames),

		req.NewGet(":instanceId/server-info", d.GetDbServer),

		req.NewDelete(":instanceId", d.DeleteInstance).Log(req.NewLogSave("db-删除数据库实例")),
	}

	req.BatchSetGroup(instances, reqs[:])
}
