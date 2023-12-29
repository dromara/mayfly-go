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
	DbRestoreApp *application.DbRestoreApp
	DbApp        application.Db
}

// GetPageList 获取数据库恢复任务
// @router /api/dbs/:dbId/restores [GET]
func (d *DbRestore) GetPageList(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.DbApp.GetById(new(entity.Db), dbId, "db_instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	var restores []vo.DbRestore
	queryCond, page := ginx.BindQueryAndPage[*entity.DbRestoreQuery](rc.GinCtx, new(entity.DbRestoreQuery))
	queryCond.DbInstanceId = db.InstanceId
	queryCond.InDbNames = strings.Fields(db.Database)
	res, err := d.DbRestoreApp.GetPageList(queryCond, page, &restores)
	biz.ErrIsNilAppendErr(err, "获取数据库恢复任务失败: %v")
	rc.ResData = res
}

// Create 保存数据库恢复任务
// @router /api/dbs/:dbId/restores [POST]
func (d *DbRestore) Create(rc *req.Ctx) {
	form := &form.DbRestoreForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)
	rc.ReqParam = form

	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.DbApp.GetById(new(entity.Db), dbId, "instanceId")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	task := &entity.DbRestore{
		DbName:              form.DbName,
		StartTime:           form.StartTime,
		Interval:            form.Interval,
		Enabled:             true,
		Repeated:            form.Repeated,
		DbInstanceId:        db.InstanceId,
		PointInTime:         form.PointInTime,
		DbBackupId:          form.DbBackupId,
		DbBackupHistoryId:   form.DbBackupHistoryId,
		DbBackupHistoryName: form.DbBackupHistoryName,
	}
	biz.ErrIsNilAppendErr(d.DbRestoreApp.Create(rc.MetaCtx, task), "添加数据库恢复任务失败: %v")
}

// Save 保存数据库恢复任务
// @router /api/dbs/:dbId/restores/:restoreId [PUT]
func (d *DbRestore) Save(rc *req.Ctx) {
	form := &form.DbRestoreForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)
	rc.ReqParam = form

	task := &entity.DbRestore{
		StartTime: form.StartTime,
		Interval:  form.Interval,
	}
	task.Id = form.Id
	biz.ErrIsNilAppendErr(d.DbRestoreApp.Save(rc.MetaCtx, task), "保存数据库恢复任务失败: %v")
}

func (d *DbRestore) walk(rc *req.Ctx, fn func(ctx context.Context, taskId uint64) error) error {
	idsStr := ginx.PathParam(rc.GinCtx, "restoreId")
	biz.NotEmpty(idsStr, "restoreId 为空")
	rc.ReqParam = idsStr
	ids := strings.Fields(idsStr)
	for _, v := range ids {
		value, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		taskId := uint64(value)
		err = fn(rc.MetaCtx, taskId)
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete 删除数据库恢复任务
// @router /api/dbs/:dbId/restores/:taskId [DELETE]
func (d *DbRestore) Delete(rc *req.Ctx) {
	err := d.walk(rc, d.DbRestoreApp.Delete)
	biz.ErrIsNilAppendErr(err, "删除数据库恢复任务失败: %v")
}

// Enable 删除数据库恢复任务
// @router /api/dbs/:dbId/restores/:taskId/enable [PUT]
func (d *DbRestore) Enable(rc *req.Ctx) {
	err := d.walk(rc, d.DbRestoreApp.Enable)
	biz.ErrIsNilAppendErr(err, "启用数据库恢复任务失败: %v")
}

// Disable 删除数据库恢复任务
// @router /api/dbs/:dbId/restores/:taskId/disable [PUT]
func (d *DbRestore) Disable(rc *req.Ctx) {
	err := d.walk(rc, d.DbRestoreApp.Disable)
	biz.ErrIsNilAppendErr(err, "禁用数据库恢复任务失败: %v")
}

// GetDbNamesWithoutRestore 获取未配置定时恢复的数据库名称
// @router /api/dbs/:dbId/db-names-without-backup [GET]
func (d *DbRestore) GetDbNamesWithoutRestore(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	db, err := d.DbApp.GetById(new(entity.Db), dbId, "instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")
	dbNames := strings.Fields(db.Database)
	dbNamesWithoutRestore, err := d.DbRestoreApp.GetDbNamesWithoutRestore(db.InstanceId, dbNames)
	biz.ErrIsNilAppendErr(err, "获取未配置定时备份的数据库名称失败: %v")
	rc.ResData = dbNamesWithoutRestore
}
