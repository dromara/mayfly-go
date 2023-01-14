package router

import (
	"mayfly-go/internal/mongo/api"
	"mayfly-go/internal/mongo/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMongoRouter(router *gin.RouterGroup) {
	m := router.Group("mongos")
	{
		ma := &api.Mongo{
			MongoApp: application.GetMongoApp(),
			TagApp:   tagapp.GetTagTreeApp(),
		}

		// 获取所有mongo列表
		m.GET("", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				Handle(ma.Mongos)
		})

		saveMongo := req.NewLogInfo("mongo-保存信息")
		m.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(saveMongo).
				Handle(ma.Save)
		})

		deleteMongo := req.NewLogInfo("mongo-删除信息")
		m.DELETE(":id", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(deleteMongo).
				Handle(ma.DeleteMongo)
		})

		// 获取mongo下的所有数据库
		m.GET(":id/databases", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				Handle(ma.Databases)
		})

		// 获取mongo指定库的所有集合
		m.GET(":id/collections", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				Handle(ma.Collections)
		})

		// 获取mongo runCommand
		m.POST(":id/run-command", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				Handle(ma.RunCommand)
		})

		// 执行mongo find命令
		m.POST(":id/command/find", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				Handle(ma.FindCommand)
		})

		// 执行mongo update by id命令
		updateDocById := req.NewLogInfo("mongo-更新文档").WithSave(true)
		m.POST(":id/command/update-by-id", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(updateDocById).
				Handle(ma.UpdateByIdCommand)
		})

		// 执行mongo delete by id命令
		deleteDoc := req.NewLogInfo("mongo-删除文档").WithSave(true)
		m.POST(":id/command/delete-by-id", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(deleteDoc).
				Handle(ma.DeleteByIdCommand)
		})

		// 执行mongo insert 命令
		insertDoc := req.NewLogInfo("mongo-新增文档").WithSave(true)
		m.POST(":id/command/insert", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(insertDoc).
				Handle(ma.InsertOneCommand)
		})
	}
}
