package router

import (
	"mayfly-go/internal/devops/api"
	"mayfly-go/internal/devops/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitMongoRouter(router *gin.RouterGroup) {
	m := router.Group("mongos")
	{
		ma := &api.Mongo{
			MongoApp: application.MongoApp,
		}

		// 获取所有mongo列表
		m.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				Handle(ma.Mongos)
		})

		saveMongo := ctx.NewLogInfo("保存mongo信息")
		m.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(saveMongo).
				Handle(ma.Save)
		})

		deleteMongo := ctx.NewLogInfo("删除mongo信息")
		m.DELETE(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(deleteMongo).
				Handle(ma.DeleteMongo)
		})

		// 获取mongo下的所有数据库
		m.GET(":id/databases", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				Handle(ma.Databases)
		})

		// 获取mongo指定库的所有集合
		m.GET(":id/collections", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				Handle(ma.Collections)
		})

		// 获取mongo runCommand
		m.POST(":id/run-command", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				Handle(ma.RunCommand)
		})

		// 执行mongo find命令
		m.POST(":id/command/find", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				Handle(ma.FindCommand)
		})

		// 执行mongo update by id命令
		updateDocById := ctx.NewLogInfo("mongo-更新文档")
		m.POST(":id/command/update-by-id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(updateDocById).
				Handle(ma.UpdateByIdCommand)
		})

		// 执行mongo delete by id命令
		deleteDoc := ctx.NewLogInfo("mongo-删除文档")
		m.POST(":id/command/delete-by-id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(deleteDoc).
				Handle(ma.DeleteByIdCommand)
		})

		// 执行mongo insert 命令
		insertDoc := ctx.NewLogInfo("mongo-新增文档")
		m.POST(":id/command/insert", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(insertDoc).
				Handle(ma.InsertOneCommand)
		})
	}
}
