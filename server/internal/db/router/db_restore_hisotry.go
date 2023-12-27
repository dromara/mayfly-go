package router

import (
	"github.com/gin-gonic/gin"
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	"mayfly-go/pkg/req"
)

func InitDbRestoreHistoryRouter(router *gin.RouterGroup) {
	dbs := router.Group("/dbs")

	d := &api.DbRestoreHistory{
		DbRestoreHistoryApp: application.GetDbRestoreHistoryApp(),
	}

	reqs := []*req.Conf{
		// 获取数据库备份历史
		req.NewGet(":dbId/restores/:restoreId/histories", d.GetPageList),
		// 删除数据库备份历史
		req.NewDelete(":dbId/restores/:restoreId/histories/:historyId", d.Delete),
	}

	req.BatchSetGroup(dbs, reqs)
}
