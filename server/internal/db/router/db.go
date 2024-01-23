package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitDbRouter(router *gin.RouterGroup) {
	db := router.Group("dbs")

	d := new(api.Db)
	biz.ErrIsNil(ioc.Inject(d))

	dashbord := new(api.Dashbord)
	biz.ErrIsNil(ioc.Inject(dashbord))

	reqs := [...]*req.Conf{
		req.NewGet("dashbord", dashbord.Dashbord),

		// 获取数据库列表
		req.NewGet("", d.Dbs),

		req.NewPost("", d.Save).Log(req.NewLogSave("db-保存数据库信息")),

		req.NewDelete(":dbId", d.DeleteDb).Log(req.NewLogSave("db-删除数据库信息")),

		req.NewGet(":dbId/t-create-ddl", d.GetTableDDL),

		req.NewGet(":dbId/pg/schemas", d.GetSchemas),

		req.NewPost(":dbId/exec-sql", d.ExecSql).Log(req.NewLog("db-执行Sql")),

		req.NewPost(":dbId/exec-sql-file", d.ExecSqlFile).Log(req.NewLogSave("db-执行Sql文件")),

		req.NewGet(":dbId/dump", d.DumpSql).Log(req.NewLogSave("db-导出sql文件")).NoRes(),

		req.NewGet(":dbId/t-infos", d.TableInfos),

		req.NewGet(":dbId/t-index", d.TableIndex),

		req.NewGet(":dbId/c-metadata", d.ColumnMA),

		req.NewGet(":dbId/hint-tables", d.HintTables),

		req.NewGet(":dbId/restore-task", d.GetRestoreTask),

		req.NewPost(":dbId/restore-task", d.SaveRestoreTask).
			Log(req.NewLogSave("db-保存数据库恢复任务")),

		req.NewGet(":dbId/restore-histories", d.GetRestoreHistories),

		req.NewPost(":dbId/copy-table", d.CopyTable),
	}

	req.BatchSetGroup(db, reqs[:])

}
