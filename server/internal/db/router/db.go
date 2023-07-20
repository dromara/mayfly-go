package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	msgapp "mayfly-go/internal/msg/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitDbRouter(router *gin.RouterGroup) {
	db := router.Group("dbs")

	d := &api.Db{
		DbApp:        application.GetDbApp(),
		DbSqlExecApp: application.GetDbSqlExecApp(),
		MsgApp:       msgapp.GetMsgApp(),
		TagApp:       tagapp.GetTagTreeApp(),
	}

	reqs := [...]*req.Conf{
		// 获取数据库列表
		req.NewGet("", d.Dbs),

		req.NewPost("", d.Save).Log(req.NewLogSave("db-保存数据库信息")),

		// 获取数据库实例的所有数据库名
		req.NewPost("/databases", d.GetDatabaseNames),

		req.NewGet(":dbId/pwd", d.GetDbPwd),

		req.NewDelete(":dbId", d.DeleteDb).Log(req.NewLogSave("db-删除数据库信息")),

		req.NewGet(":dbId/t-infos", d.TableInfos),

		req.NewGet(":dbId/t-index", d.TableIndex),

		req.NewGet(":dbId/t-create-ddl", d.GetCreateTableDdl),

		req.NewPost(":dbId/exec-sql", d.ExecSql).Log(req.NewLog("db-执行Sql")),

		req.NewPost(":dbId/exec-sql-file", d.ExecSqlFile).Log(req.NewLogSave("db-执行Sql文件")),

		req.NewGet(":dbId/dump", d.DumpSql).Log(req.NewLogSave("db-导出sql文件")).NoRes(),

		req.NewGet(":dbId/t-metadata", d.TableMA),

		req.NewGet(":dbId/c-metadata", d.ColumnMA),

		req.NewGet(":dbId/hint-tables", d.HintTables),

		// 用户sql相关
		req.NewPost(":dbId/sql", d.SaveSql),

		req.NewGet(":dbId/sql", d.GetSql),

		req.NewDelete(":dbId/sql", d.DeleteSql),

		req.NewGet(":dbId/sql-names", d.GetSqlNames),
	}

	req.BatchSetGroup(db, reqs[:])

}
