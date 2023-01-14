package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitResourceRouter(router *gin.RouterGroup) {
	r := &api.Resource{ResourceApp: application.GetResourceApp()}
	db := router.Group("sys/resources")
	{
		db.GET("", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(r.GetAllResourceTree)
		})

		db.GET(":id", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(r.GetById)
		})

		saveResource := req.NewLogInfo("保存资源").WithSave(true)
		srPermission := req.NewPermission("resource:add")
		db.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(saveResource).
				WithRequiredPermission(srPermission).
				Handle(r.SaveResource)
		})

		changeStatus := req.NewLogInfo("修改资源状态").WithSave(true)
		csPermission := req.NewPermission("resource:changeStatus")
		db.PUT(":id/:status", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(changeStatus).
				WithRequiredPermission(csPermission).
				Handle(r.ChangeStatus)
		})

		delResource := req.NewLogInfo("删除资源").WithSave(true)
		dePermission := req.NewPermission("resource:delete")
		db.DELETE(":id", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(delResource).
				WithRequiredPermission(dePermission).
				Handle(r.DelResource)
		})
	}
}
