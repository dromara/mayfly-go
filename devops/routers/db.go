package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/devops/apis"
	"mayfly-go/devops/application"

	"github.com/gin-gonic/gin"
)

func InitDbRouter(router *gin.RouterGroup) {
	db := router.Group("dbs")
	{
		d := &apis.Db{DbApp: application.Db}
		// 获取所有数据库列表
		db.GET("", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c)
			rc.Handle(d.Dbs)
		})
		// db.GET(":dbId/select", controllers.SelectData)
		db.GET(":dbId/select", func(g *gin.Context) {
			rc := ctx.NewReqCtxWithGin(g).WithLog(ctx.NewLogInfo("执行数据库查询语句"))
			rc.Handle(d.SelectData)
		})

		db.GET(":dbId/t-metadata", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(d.TableMA)
		})

		db.GET(":dbId/c-metadata", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(d.ColumnMA)
		})
		db.GET(":dbId/hint-tables", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(d.HintTables)
		})

		db.POST(":dbId/sql", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(ctx.NewLogInfo("保存sql内容"))
			rc.Handle(d.SaveSql)
		})

		db.GET(":dbId/sql", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(d.GetSql)
		})
	}
}
