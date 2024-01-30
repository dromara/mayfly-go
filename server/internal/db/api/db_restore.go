package api

import (
	"context"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"strconv"
	"strings"
)

type DbRestore struct {
	restoreApp *application.DbRestoreApp `inject:"DbRestoreApp"`
	dbApp      application.Db            `inject:"DbApp"`
}

// GetPageList 获取数据库恢复任务
// @router /api/dbs/:dbId/restores [GET]
func (d *DbRestore) GetPageList(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.dbApp.GetById(new(entity.Db), dbId, "db_instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	var restores []vo.DbRestore
	queryCond, page := ginx.BindQueryAndPage[*entity.DbRestoreQuery](rc.GinCtx, new(entity.DbRestoreQuery))
	queryCond.DbInstanceId = db.InstanceId
	queryCond.InDbNames = strings.Fields(db.Database)
	res, err := d.restoreApp.GetPageList(queryCond, page, &restores)
	biz.ErrIsNilAppendErr(err, "获取数据库恢复任务失败: %v")
	rc.ResData = res
}

// Create 保存数据库恢复任务
// @router /api/dbs/:dbId/restores [POST]
func (d *DbRestore) Create(rc *req.Ctx) {
	restoreForm := &form.DbRestoreForm{}
	ginx.BindJsonAndValid(rc.GinCtx, restoreForm)
	rc.ReqParam = restoreForm

	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.dbApp.GetById(new(entity.Db), dbId, "instanceId")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	job := &entity.DbRestore{
		DbInstanceId:        db.InstanceId,
		DbName:              restoreForm.DbName,
		Enabled:             true,
		Repeated:            restoreForm.Repeated,
		StartTime:           restoreForm.StartTime,
		Interval:            restoreForm.Interval,
		PointInTime:         restoreForm.PointInTime,
		DbBackupId:          restoreForm.DbBackupId,
		DbBackupHistoryId:   restoreForm.DbBackupHistoryId,
		DbBackupHistoryName: restoreForm.DbBackupHistoryName,
	}
	biz.ErrIsNilAppendErr(d.restoreApp.Create(rc.MetaCtx, job), "添加数据库恢复任务失败: %v")
}

func (d *DbRestore) createWithBackupHistory(backupHistoryIds string) {

}

// Update 保存数据库恢复任务
// @router /api/dbs/:dbId/restores/:restoreId [PUT]
func (d *DbRestore) Update(rc *req.Ctx) {
	restoreForm := &form.DbRestoreForm{}
	ginx.BindJsonAndValid(rc.GinCtx, restoreForm)
	rc.ReqParam = restoreForm

	job := &entity.DbRestore{}
	job.Id = restoreForm.Id
	job.StartTime = restoreForm.StartTime
	job.Interval = restoreForm.Interval
	biz.ErrIsNilAppendErr(d.restoreApp.Update(rc.MetaCtx, job), "保存数据库恢复任务失败: %v")
}

func (d *DbRestore) walk(rc *req.Ctx, fn func(ctx context.Context, restoreId uint64) error) error {
	idsStr := ginx.PathParam(rc.GinCtx, "restoreId")
	biz.NotEmpty(idsStr, "restoreId 为空")
	rc.ReqParam = idsStr
	ids := strings.Fields(idsStr)
	for _, v := range ids {
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		restoreId := uint64(value)
		err = fn(rc.MetaCtx, restoreId)
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete 删除数据库恢复任务
// @router /api/dbs/:dbId/restores/:restoreId [DELETE]
func (d *DbRestore) Delete(rc *req.Ctx) {
	err := d.walk(rc, d.restoreApp.Delete)
	biz.ErrIsNilAppendErr(err, "删除数据库恢复任务失败: %v")
}

// Enable 启用数据库恢复任务
// @router /api/dbs/:dbId/restores/:restoreId/enable [PUT]
func (d *DbRestore) Enable(rc *req.Ctx) {
	err := d.walk(rc, d.restoreApp.Enable)
	biz.ErrIsNilAppendErr(err, "启用数据库恢复任务失败: %v")
}

// Disable 禁用数据库恢复任务
// @router /api/dbs/:dbId/restores/:restoreId/disable [PUT]
func (d *DbRestore) Disable(rc *req.Ctx) {
	err := d.walk(rc, d.restoreApp.Disable)
	biz.ErrIsNilAppendErr(err, "禁用数据库恢复任务失败: %v")
}

// GetDbNamesWithoutRestore 获取未配置定时恢复的数据库名称
// @router /api/dbs/:dbId/db-names-without-backup [GET]
func (d *DbRestore) GetDbNamesWithoutRestore(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	db, err := d.dbApp.GetById(new(entity.Db), dbId, "instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")
	dbNames := strings.Fields(db.Database)
	dbNamesWithoutRestore, err := d.restoreApp.GetDbNamesWithoutRestore(db.InstanceId, dbNames)
	biz.ErrIsNilAppendErr(err, "获取未配置定时备份的数据库名称失败: %v")
	rc.ResData = dbNamesWithoutRestore
}

// GetHistoryPageList 获取数据库备份历史
// @router /api/dbs/:dbId/restores/:restoreId/histories [GET]
func (d *DbRestore) GetHistoryPageList(rc *req.Ctx) {
	queryCond := &entity.DbRestoreHistoryQuery{
		DbRestoreId: uint64(ginx.PathParamInt(rc.GinCtx, "restoreId")),
	}
	res, err := d.restoreApp.GetHistoryPageList(queryCond, ginx.GetPageParam(rc.GinCtx), new([]vo.DbRestoreHistory))
	biz.ErrIsNilAppendErr(err, "获取数据库备份历史失败: %v")
	rc.ResData = res
}
