package router

import (
	"github.com/gin-gonic/gin"
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	"mayfly-go/pkg/req"
)

func InitDbBackupHistoryRouter(router *gin.RouterGroup) {
	dbs := router.Group("/dbs")

	d := &api.DbBackupHistory{
		DbBackupHistoryApp: application.GetDbBackupHistoryApp(),
		DbApp:              application.GetDbApp(),
	}

	reqs := []*req.Conf{
		// 获取数据库备份历史
		req.NewGet(":dbId/backup-histories/", d.GetPageList),
		// 删除数据库备份历史
		req.NewDelete(":dbId/backups/:backupId/histories/:historyId", d.Delete),
	}

	req.BatchSetGroup(dbs, reqs)
}
