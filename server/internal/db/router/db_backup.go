package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitDbBackupRouter(router *gin.RouterGroup) {
	dbs := router.Group("/dbs")

	d := &api.DbBackup{
		DbBackupApp: application.GetDbBackupApp(),
		DbApp:       application.GetDbApp(),
	}

	reqs := []*req.Conf{
		// 获取数据库备份任务
		req.NewGet(":dbId/backups", d.GetPageList),
		// 创建数据库备份任务
		req.NewPost(":dbId/backups", d.Create).Log(req.NewLogSave("db-创建数据库备份任务")),
		// 保存数据库备份任务
		req.NewPut(":dbId/backups/:backupId", d.Update).Log(req.NewLogSave("db-保存数据库备份任务")),
		// 启用数据库备份任务
		req.NewPut(":dbId/backups/:backupId/enable", d.Enable).Log(req.NewLogSave("db-启用数据库备份任务")),
		// 禁用数据库备份任务
		req.NewPut(":dbId/backups/:backupId/disable", d.Disable).Log(req.NewLogSave("db-禁用数据库备份任务")),
		// 立即启动数据库备份任务
		req.NewPut(":dbId/backups/:backupId/start", d.Start).Log(req.NewLogSave("db-立即启动数据库备份任务")),
		// 删除数据库备份任务
		req.NewDelete(":dbId/backups/:backupId", d.Delete),
		// 获取未配置定时备份的数据库名称
		req.NewGet(":dbId/db-names-without-backup", d.GetDbNamesWithoutBackup),

		// 获取数据库备份历史
		req.NewGet(":dbId/backup-histories/", d.GetHistoryPageList),
	}

	req.BatchSetGroup(dbs, reqs)
}
