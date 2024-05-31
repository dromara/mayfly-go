package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitDbRestoreRouter(router *gin.RouterGroup) {
	dbs := router.Group("/dbs")

	d := &api.DbRestore{}
	biz.ErrIsNil(ioc.Inject(d))

	reqs := []*req.Conf{
		// 获取数据库备份任务
		req.NewGet(":dbId/restores", d.GetPageList),
		// 创建数据库备份任务
		req.NewPost(":dbId/restores", d.Create).Log(req.NewLogSave("db-创建数据库恢复任务")),
		// 保存数据库备份任务
		req.NewPut(":dbId/restores/:restoreId", d.Update).Log(req.NewLogSave("db-保存数据库恢复任务")),
		// 启用数据库备份任务
		req.NewPut(":dbId/restores/:restoreId/enable", d.Enable).Log(req.NewLogSave("db-启用数据库恢复任务")),
		// 禁用数据库备份任务
		req.NewPut(":dbId/restores/:restoreId/disable", d.Disable).Log(req.NewLogSave("db-禁用数据库恢复任务")),
		// 删除数据库备份任务
		req.NewDelete(":dbId/restores/:restoreId", d.Delete),
		// 获取未配置定时恢复的数据库名称
		req.NewGet(":dbId/db-names-without-restore", d.GetDbNamesWithoutRestore),

		// 获取数据库备份历史
		req.NewGet(":dbId/restores/:restoreId/histories", d.GetHistoryPageList),
	}

	req.BatchSetGroup(dbs, reqs)
}
