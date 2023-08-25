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

	ma := &api.Mongo{
		MongoApp: application.GetMongoApp(),
		TagApp:   tagapp.GetTagTreeApp(),
	}

	saveDataPerm := req.NewPermission("mongo:data:save")

	reqs := [...]*req.Conf{
		// 获取所有mongo列表
		req.NewGet("", ma.Mongos),

		req.NewGet("/tags", ma.MongoTags),

		req.NewPost("", ma.Save).Log(req.NewLogSave("mongo-保存信息")),

		req.NewDelete(":id", ma.DeleteMongo).Log(req.NewLogSave("mongo-删除信息")),

		// 获取mongo下的所有数据库
		req.NewGet(":id/databases", ma.Databases),

		// 获取mongo指定库的所有集合
		req.NewGet(":id/collections", ma.Collections),

		// mongo runCommand
		req.NewPost(":id/run-command", ma.RunCommand).Log(req.NewLogSave("mongo-runCommand")),

		// 执行mongo find命令
		req.NewPost(":id/command/find", ma.FindCommand),

		req.NewPost(":id/command/update-by-id", ma.UpdateByIdCommand).RequiredPermission(saveDataPerm).Log(req.NewLogSave("mongo-更新文档")),

		// 执行mongo delete by id命令
		req.NewPost(":id/command/delete-by-id", ma.DeleteByIdCommand).RequiredPermission(req.NewPermission("mongo:data:del")).Log(req.NewLogSave("mongo-删除文档")),

		// 执行mongo insert 命令
		req.NewPost(":id/command/insert", ma.InsertOneCommand).RequiredPermission(saveDataPerm).Log(req.NewLogSave("mogno-插入文档")),
	}

	req.BatchSetGroup(m, reqs[:])

}
