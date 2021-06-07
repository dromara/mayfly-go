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
		db.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveResource).Handle(r.SaveResource)
		})

		changeStatus := ctx.NewLogInfo("修改资源状态")
		db.PUT(":id/:status", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(changeStatus).Handle(r.ChangeStatus)
		})

		delResource := ctx.NewLogInfo("删除资源")
		db.DELETE(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(delResource).Handle(r.DelResource)
		})
	}
}
