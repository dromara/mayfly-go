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

type DbBackup struct {
	DbBackupApp *application.DbBackupApp
	DbApp       application.Db
}

// todo: 鉴权，避免未经授权进行数据库备份和恢复

// GetPageList 获取数据库备份任务
// @router /api/dbs/:dbId/backups [GET]
func (d *DbBackup) GetPageList(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.DbApp.GetById(new(entity.Db), dbId, "db_instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	queryCond, page := ginx.BindQueryAndPage[*entity.DbBackupQuery](rc.GinCtx, new(entity.DbBackupQuery))
	queryCond.DbInstanceId = db.InstanceId
	queryCond.InDbNames = strings.Fields(db.Database)
	res, err := d.DbBackupApp.GetPageList(queryCond, page, new([]vo.DbBackup))
	biz.ErrIsNilAppendErr(err, "获取数据库备份任务失败: %v")
	rc.ResData = res
}

// Create 保存数据库备份任务
// @router /api/dbs/:dbId/backups [POST]
func (d *DbBackup) Create(rc *req.Ctx) {
	form := &form.DbBackupForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)
	rc.ReqParam = form

	dbNames := strings.Fields(form.DbNames)
	biz.IsTrue(len(dbNames) > 0, "解析数据库备份任务失败：数据库名称未定义")

	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.DbApp.GetById(new(entity.Db), dbId, "instanceId")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	tasks := make([]*entity.DbBackup, 0, len(dbNames))
	for _, dbName := range dbNames {
		task := &entity.DbBackup{
			DbName:       dbName,
			Name:         form.Name,
			StartTime:    form.StartTime,
			Interval:     form.Interval,
			Enabled:      true,
			Repeated:     form.Repeated,
			DbInstanceId: db.InstanceId,
			LastTime:     form.StartTime,
		}
		tasks = append(tasks, task)
	}
	biz.ErrIsNilAppendErr(d.DbBackupApp.Create(rc.MetaCtx, tasks...), "添加数据库备份任务失败: %v")
}

// Save 保存数据库备份任务
// @router /api/dbs/:dbId/backups/:backupId [PUT]
func (d *DbBackup) Save(rc *req.Ctx) {
	form := &form.DbBackupForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)
	rc.ReqParam = form

	task := &entity.DbBackup{
		Name:      form.Name,
		StartTime: form.StartTime,
		Interval:  form.Interval,
		LastTime:  form.StartTime,
	}
	task.Id = form.Id
	biz.ErrIsNilAppendErr(d.DbBackupApp.Save(rc.MetaCtx, task), "保存数据库备份任务失败: %v")
}

func (d *DbBackup) walk(rc *req.Ctx, fn func(ctx context.Context, backupId uint64) error) error {
	idsStr := ginx.PathParam(rc.GinCtx, "backupId")
	biz.NotEmpty(idsStr, "backupId 为空")
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

// Delete 删除数据库备份任务
// @router /api/dbs/:dbId/backups/:taskId [DELETE]
func (d *DbBackup) Delete(rc *req.Ctx) {
	err := d.walk(rc, d.DbBackupApp.Delete)
	biz.ErrIsNilAppendErr(err, "删除数据库备份任务失败: %v")
}

// Enable 启用数据库备份任务
// @router /api/dbs/:dbId/backups/:taskId/enable [PUT]
func (d *DbBackup) Enable(rc *req.Ctx) {
	err := d.walk(rc, d.DbBackupApp.Enable)
	biz.ErrIsNilAppendErr(err, "启用数据库备份任务失败: %v")
}

// Disable 禁用数据库备份任务
// @router /api/dbs/:dbId/backups/:taskId/disable [PUT]
func (d *DbBackup) Disable(rc *req.Ctx) {
	err := d.walk(rc, d.DbBackupApp.Disable)
	biz.ErrIsNilAppendErr(err, "禁用数据库备份任务失败: %v")
}

// GetDbNamesWithoutBackup 获取未配置定时备份的数据库名称
// @router /api/dbs/:dbId/db-names-without-backup [GET]
func (d *DbBackup) GetDbNamesWithoutBackup(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	db, err := d.DbApp.GetById(new(entity.Db), dbId, "instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")
	dbNames := strings.Fields(db.Database)
	dbNamesWithoutBackup, err := d.DbBackupApp.GetDbNamesWithoutBackup(db.InstanceId, dbNames)
	biz.ErrIsNilAppendErr(err, "获取未配置定时备份的数据库名称失败: %v")
	rc.ResData = dbNamesWithoutBackup
}
