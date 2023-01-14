package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitRoleRouter(router *gin.RouterGroup) {
	r := &api.Role{
		RoleApp:     application.GetRoleApp(),
		ResourceApp: application.GetResourceApp(),
	}
	db := router.Group("sys/roles")
	{

		db.GET("", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(r.Roles)
		})

		saveRole := req.NewLogInfo("保存角色").WithSave(true)
		sPermission := req.NewPermission("role:add")
		db.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(saveRole).
				WithRequiredPermission(sPermission).
				Handle(r.SaveRole)
		})

		delRole := req.NewLogInfo("删除角色").WithSave(true)
		drPermission := req.NewPermission("role:del")
		db.DELETE(":id", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(delRole).
				WithRequiredPermission(drPermission).
				Handle(r.DelRole)
		})

		db.GET(":id/resourceIds", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(r.RoleResourceIds)
		})

		db.GET(":id/resources", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(r.RoleResource)
		})

		saveResource := req.NewLogInfo("保存角色资源").WithSave(true)
		srPermission := req.NewPermission("role:saveResources")
		db.POST(":id/resources", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(saveResource).
				WithRequiredPermission(srPermission).
				Handle(r.SaveResource)
		})
	}
}
