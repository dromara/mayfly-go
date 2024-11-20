package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/imsg"
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

		req.NewPost("", d.Save).Log(req.NewLogSaveI(imsg.LogDbSave)),

		req.NewDelete(":dbId", d.DeleteDb).Log(req.NewLogSaveI(imsg.LogDbDelete)),

		req.NewGet(":dbId/t-create-ddl", d.GetTableDDL),

		req.NewGet(":dbId/version", d.GetVersion),

		req.NewGet(":dbId/pg/schemas", d.GetSchemas),

		req.NewPost(":dbId/exec-sql", d.ExecSql).Log(req.NewLogI(imsg.LogDbRunSql)),

		req.NewPost(":dbId/exec-sql-file", d.ExecSqlFile).Log(req.NewLogSaveI(imsg.LogDbRunSqlFile)).RequiredPermissionCode("db:sqlscript:run"),

		req.NewGet(":dbId/dump", d.DumpSql).Log(req.NewLogSaveI(imsg.LogDbDump)).NoRes(),

		req.NewGet(":dbId/t-infos", d.TableInfos),

		req.NewGet(":dbId/t-index", d.TableIndex),

		req.NewGet(":dbId/c-metadata", d.ColumnMA),

		req.NewGet(":dbId/hint-tables", d.HintTables),

		req.NewPost(":dbId/copy-table", d.CopyTable),
	}

	req.BatchSetGroup(db, reqs[:])

}
