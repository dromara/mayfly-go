package api

import (
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
)

type DbRestoreHistory struct {
	InstanceApp         application.Instance
	DbRestoreHistoryApp *application.DbRestoreHistoryApp
}

// GetPageList 获取数据库备份历史
// @router /api/dbs/:dbId/restores/:restoreId/histories [GET]
func (d *DbRestoreHistory) GetPageList(rc *req.Ctx) {
	queryCond := &entity.DbRestoreHistoryQuery{
		DbRestoreId: uint64(ginx.PathParamInt(rc.GinCtx, "restoreId")),
	}
	res, err := d.DbRestoreHistoryApp.GetPageList(queryCond, ginx.GetPageParam(rc.GinCtx), new([]vo.DbRestoreHistory))
	biz.ErrIsNilAppendErr(err, "获取数据库备份历史失败: %v")
	rc.ResData = res
}

// Delete 删除数据库备份历史
// @router /api/dbs/:dbId/restores/:restoreId/histories/:historyId [DELETE]
func (d *DbRestoreHistory) Delete(rc *req.Ctx) {
	// todo delete restore histories
	panic("implement me")
}
