package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/server/sys/apis"
	"mayfly-go/server/sys/application"

	"github.com/gin-gonic/gin"
)

func InitRoleRouter(router *gin.RouterGroup) {
	r := &apis.Role{
		RoleApp:     application.Role,
		ResourceApp: application.Resource,
	}
	db := router.Group("sys/roles")
	{

		db.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.Roles)
		})

		saveRole := ctx.NewLogInfo("保存角色")
		db.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveRole).Handle(r.SaveRole)
		})

		delRole := ctx.NewLogInfo("删除角色")
		db.DELETE(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(delRole).Handle(r.DelRole)
		})

		db.GET(":id/resourceIds", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.RoleResourceIds)
		})

		db.GET(":id/resources", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.RoleResource)
		})

		saveResource := ctx.NewLogInfo("保存角色资源")
		db.POST(":id/resources", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveResource).Handle(r.SaveResource)
		})
	}
}
