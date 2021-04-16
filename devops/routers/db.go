package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/devops/controllers"

	"github.com/gin-gonic/gin"
)

func InitDbRouter(router *gin.RouterGroup) {
	db := router.Group("dbs")
	{
		// 获取所有数据库列表
		db.GET("", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c)
			rc.Handle(controllers.Dbs)
		})
		// db.GET(":dbId/select", controllers.SelectData)
		db.GET(":dbId/select", func(g *gin.Context) {
			rc := ctx.NewReqCtxWithGin(g).WithLog(ctx.NewLogInfo("执行数据库查询语句"))
			rc.Handle(controllers.SelectData)
		})

		db.GET(":dbId/t-metadata", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(controllers.TableMA)
		})

		db.GET(":dbId/c-metadata", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(controllers.ColumnMA)
		})
		db.GET(":dbId/hint-tables", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(controllers.HintTables)
		})

		db.POST(":dbId/sql", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(ctx.NewLogInfo("保存sql内容"))
			rc.Handle(controllers.SaveSql)
		})

		db.GET(":dbId/sql", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(controllers.GetSql)
		})
	}
}
