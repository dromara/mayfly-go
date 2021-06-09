package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/server/sys/apis"
	"mayfly-go/server/sys/application"

	"github.com/gin-gonic/gin"
)

func InitResourceRouter(router *gin.RouterGroup) {
	r := &apis.Resource{ResourceApp: application.Resource}
	db := router.Group("sys/resources")
	{
		// db.GET("/account", func(c *gin.Context) {
		// 	ctx.NewReqCtxWithGin(c).Handle(r.ResourceTree)
		// })

		db.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.GetAllResourceTree)
		})

		db.GET(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.GetById)
		})

		saveResource := ctx.NewLogInfo("保存资源")
		srPermission := ctx.NewPermission("resource:add")
		db.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(saveResource).
				WithRequiredPermission(srPermission).
				Handle(r.SaveResource)
		})

		changeStatus := ctx.NewLogInfo("修改资源状态")
		csPermission := ctx.NewPermission("resource:changeStatus")
		db.PUT(":id/:status", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(changeStatus).
				WithRequiredPermission(csPermission).
				Handle(r.ChangeStatus)
		})

		delResource := ctx.NewLogInfo("删除资源")
		dePermission := ctx.NewPermission("resource:delete")
		db.DELETE(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(delResource).
				WithRequiredPermission(dePermission).
				Handle(r.DelResource)
		})
	}
}
