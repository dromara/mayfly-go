package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitResourceRouter(router *gin.RouterGroup) {
	r := &api.Resource{ResourceApp: application.GetResourceApp()}
	rg := router.Group("sys/resources")

	reqs := [...]*req.Conf{
		req.NewGet("", r.GetAllResourceTree),

		req.NewGet(":id", r.GetById),

		req.NewPost("", r.SaveResource).Log(req.NewLogSave("保存资源")).RequiredPermissionCode("resource:add"),

		req.NewPut(":id/:status", r.ChangeStatus).Log(req.NewLogSave("修改资源状态")).RequiredPermissionCode("resource:changeStatus"),

		req.NewPost("sort", r.Sort).Log(req.NewLogSave("资源排序")),

		req.NewDelete(":id", r.DelResource).Log(req.NewLogSave("删除资源")).RequiredPermissionCode("resource:delete"),
	}

	req.BatchSetGroup(rg, reqs[:])
}
