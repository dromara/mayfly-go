package api

import (
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"strings"
)

type DbBackupHistory struct {
	DbBackupHistoryApp *application.DbBackupHistoryApp
	DbApp              application.Db
}

// GetPageList 获取数据库备份历史
// @router /api/dbs/:dbId/backups/:backupId/histories [GET]
func (d *DbBackupHistory) GetPageList(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.DbApp.GetById(new(entity.Db), dbId, "db_instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	queryCond, page := ginx.BindQueryAndPage[*entity.DbBackupHistoryQuery](rc.GinCtx, new(entity.DbBackupHistoryQuery))
	queryCond.DbInstanceId = db.InstanceId
	queryCond.InDbNames = strings.Fields(db.Database)
	res, err := d.DbBackupHistoryApp.GetPageList(queryCond, page, new([]vo.DbBackupHistory))
	biz.ErrIsNilAppendErr(err, "获取数据库备份历史失败: %v")
	rc.ResData = res
}

// Delete 删除数据库备份历史
// @router /api/dbs/:dbId/backups/:backupId/histories/:historyId [DELETE]
func (d *DbBackupHistory) Delete(rc *req.Ctx) {
	// todo delete backup histories
	panic("implement me")
}
