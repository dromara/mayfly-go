package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitResourceRouter(router *gin.RouterGroup) {
	r := &api.Resource{ResourceApp: application.GetResourceApp()}
	db := router.Group("sys/resources")
	{
		db.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.GetAllResourceTree)
		})

		db.GET(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.GetById)
		})

		saveResource := ctx.NewLogInfo("保存资源").WithSave(true)
		srPermission := ctx.NewPermission("resource:add")
		db.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(saveResource).
				WithRequiredPermission(srPermission).
				Handle(r.SaveResource)
		})

		changeStatus := ctx.NewLogInfo("修改资源状态").WithSave(true)
		csPermission := ctx.NewPermission("resource:changeStatus")
		db.PUT(":id/:status", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(changeStatus).
				WithRequiredPermission(csPermission).
				Handle(r.ChangeStatus)
		})

		delResource := ctx.NewLogInfo("删除资源").WithSave(true)
		dePermission := ctx.NewPermission("resource:delete")
		db.DELETE(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(delResource).
				WithRequiredPermission(dePermission).
				Handle(r.DelResource)
		})
	}
}
