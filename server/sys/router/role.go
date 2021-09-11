package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/server/sys/api"
	"mayfly-go/server/sys/application"

	"github.com/gin-gonic/gin"
)

func InitRoleRouter(router *gin.RouterGroup) {
	r := &api.Role{
		RoleApp:     application.RoleApp,
		ResourceApp: application.ResourceApp,
	}
	db := router.Group("sys/roles")
	{

		db.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.Roles)
		})

		saveRole := ctx.NewLogInfo("保存角色")
		sPermission := ctx.NewPermission("role:add")
		db.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveRole).
				WithRequiredPermission(sPermission).
				Handle(r.SaveRole)
		})

		delRole := ctx.NewLogInfo("删除角色")
		drPermission := ctx.NewPermission("role:del")
		db.DELETE(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(delRole).
				WithRequiredPermission(drPermission).
				Handle(r.DelRole)
		})

		db.GET(":id/resourceIds", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.RoleResourceIds)
		})

		db.GET(":id/resources", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.RoleResource)
		})

		saveResource := ctx.NewLogInfo("保存角色资源")
		srPermission := ctx.NewPermission("role:saveResources")
		db.POST(":id/resources", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveResource).
				WithRequiredPermission(srPermission).
				Handle(r.SaveResource)
		})
	}
}
