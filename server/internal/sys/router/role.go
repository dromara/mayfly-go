package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitRoleRouter(router *gin.RouterGroup) {
	rg := router.Group("sys/roles")
	r := new(api.Role)
	biz.ErrIsNil(ioc.Inject(r))

	reqs := [...]*req.Conf{
		req.NewGet("", r.Roles),

		req.NewPost("", r.SaveRole).Log(req.NewLogSaveI(imsg.LogRoleSave)).RequiredPermissionCode("role:add"),

		req.NewDelete(":id", r.DelRole).Log(req.NewLogSaveI(imsg.LogRoleDelete)).RequiredPermissionCode("role:del"),

		req.NewGet(":id/resourceIds", r.RoleResourceIds),

		req.NewGet(":id/resources", r.RoleResource),

		req.NewPost(":id/resources", r.SaveResource).Log(req.NewLogSaveI(imsg.LogAssignRoleResource)).RequiredPermissionCode("role:saveResources"),

		req.NewGet(":id/accounts", r.RoleAccount),
	}

	req.BatchSetGroup(rg, reqs[:])
}
