package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitResourceRouter(router *gin.RouterGroup) {
	rg := router.Group("sys/resources")
	r := new(api.Resource)
	biz.ErrIsNil(ioc.Inject(r))

	reqs := [...]*req.Conf{
		req.NewGet("", r.GetAllResourceTree),

		req.NewGet(":id", r.GetById),

		req.NewGet(":id/roles", r.GetResourceRoles),

		req.NewPost("", r.SaveResource).Log(req.NewLogSaveI(imsg.LogResourceSave)).RequiredPermissionCode("resource:add"),

		req.NewPut(":id/:status", r.ChangeStatus).Log(req.NewLogSaveI(imsg.LogChangeResourceStatus)).RequiredPermissionCode("resource:changeStatus"),

		req.NewPost("sort", r.Sort).Log(req.NewLogSaveI(imsg.LogSortResource)),

		req.NewDelete(":id", r.DelResource).Log(req.NewLogSaveI(imsg.LogResourceDelete)).RequiredPermissionCode("resource:delete"),
	}

	req.BatchSetGroup(rg, reqs[:])
}
