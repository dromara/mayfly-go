package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	msgapp "mayfly-go/internal/msg/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitInstanceRouter(router *gin.RouterGroup) {
	instances := router.Group("/instances")

	d := &api.Instance{
		InstanceApp: application.GetInstanceApp(),
		MsgApp:      msgapp.GetMsgApp(),
	}

	reqs := [...]*req.Conf{
		// 获取数据库列表
		req.NewGet("", d.Instances),

		req.NewPost("", d.SaveInstance).Log(req.NewLogSave("db-保存数据库实例信息")),

		req.NewGet(":dbId/pwd", d.GetInstancePwd),

		req.NewDelete(":dbId", d.DeleteInstance).Log(req.NewLogSave("db-删除数据库实例")),
	}

	req.BatchSetGroup(instances, reqs[:])
}
