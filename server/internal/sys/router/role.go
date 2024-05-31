package router

import (
	"mayfly-go/internal/sys/api"
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

		req.NewPost("", r.SaveRole).Log(req.NewLogSave("保存角色")).RequiredPermissionCode("role:add"),

		req.NewDelete(":id", r.DelRole).Log(req.NewLogSave("删除角色")).RequiredPermissionCode("role:del"),

		req.NewGet(":id/resourceIds", r.RoleResourceIds),

		req.NewGet(":id/resources", r.RoleResource),

		req.NewPost(":id/resources", r.SaveResource).Log(req.NewLogSave("保存角色资源")).RequiredPermissionCode("role:saveResources"),

		req.NewGet(":id/accounts", r.RoleAccount),
	}

	req.BatchSetGroup(rg, reqs[:])
}
