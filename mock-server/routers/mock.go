package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/mock-server/controllers"

	"github.com/gin-gonic/gin"
)

func InitMockRouter(router *gin.RouterGroup) {
	mock := router.Group("mock-datas")
	{
		// 获取mock数据
		mock.GET(":method", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithNeedToken(false).WithLog(ctx.NewLogInfo("获取mock数据"))
			rc.Handle(controllers.GetMockData)
		})

		mock.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithNeedToken(false).Handle(controllers.GetAllData)
		})

		mock.POST("", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithNeedToken(false).WithLog(ctx.NewLogInfo("保存新增mock数据"))
			rc.Handle(controllers.CreateMockData)
		})

		mock.PUT("", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithNeedToken(false).WithLog(ctx.NewLogInfo("修改mock数据"))
			rc.Handle(controllers.UpdateMockData)
		})

		mock.DELETE(":method", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithNeedToken(false).WithLog(ctx.NewLogInfo("删除mock数据"))
			rc.Handle(controllers.DeleteMockData)
		})
	}
}
