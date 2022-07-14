package router

import (
	"mayfly-go/internal/devops/api"
	"mayfly-go/internal/devops/application"
	sysApplication "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitDbRouter(router *gin.RouterGroup) {
	db := router.Group("dbs")
	{
		d := &api.Db{
			DbApp:        application.DbApp,
			DbSqlExecApp: application.DbSqlExecApp,
			MsgApp:       sysApplication.MsgApp,
			ProjectApp:   application.ProjectApp,
		}
		// 获取所有数据库列表
		db.GET("", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c)
			rc.Handle(d.Dbs)
		})

		saveDb := ctx.NewLogInfo("保存数据库信息").WithSave(true)
		db.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(saveDb).
				Handle(d.Save)
		})

		deleteDb := ctx.NewLogInfo("删除数据库信息").WithSave(true)
		db.DELETE(":dbId", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(deleteDb).
				Handle(d.DeleteDb)
		})

		db.GET(":dbId/t-infos", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(d.TableInfos)
		})

		db.GET(":dbId/t-index", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(d.TableIndex)
		})

		db.GET(":dbId/t-create-ddl", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(d.GetCreateTableDdl)
		})

		execSqlLog := ctx.NewLogInfo("执行Sql语句")
		db.POST(":dbId/exec-sql", func(g *gin.Context) {
			rc := ctx.NewReqCtxWithGin(g).WithLog(execSqlLog)
			rc.Handle(d.ExecSql)
		})

		execSqlFileLog := ctx.NewLogInfo("执行Sql文件").WithSave(true)
		db.POST(":dbId/exec-sql-file", func(g *gin.Context) {
			ctx.NewReqCtxWithGin(g).
				WithLog(execSqlFileLog).
				Handle(d.ExecSqlFile)
		})

		dumpLog := ctx.NewLogInfo("导出sql文件").WithSave(true)
		db.GET(":dbId/dump", func(g *gin.Context) {
			ctx.NewReqCtxWithGin(g).
				WithLog(dumpLog).
				Handle(d.DumpSql)
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

		/**  db sql相关接口  */

		db.POST(":dbId/sql", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c)
			rc.Handle(d.SaveSql)
		})

		db.GET(":dbId/sql", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(d.GetSql)
		})

		db.DELETE(":dbId/sql", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(d.DeleteSql)
		})

		db.GET(":dbId/sql-names", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(d.GetSqlNames)
		})
	}
}
