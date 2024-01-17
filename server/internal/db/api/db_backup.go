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

	queryCond, page := ginx.BindQueryAndPage[*entity.DbJobQuery](rc.GinCtx, new(entity.DbJobQuery))
	queryCond.DbInstanceId = db.InstanceId
	queryCond.InDbNames = strings.Fields(db.Database)
	res, err := d.DbBackupApp.GetPageList(queryCond, page, new([]vo.DbBackup))
	biz.ErrIsNilAppendErr(err, "获取数据库备份任务失败: %v")
	rc.ResData = res
}

// Create 保存数据库备份任务
// @router /api/dbs/:dbId/backups [POST]
func (d *DbBackup) Create(rc *req.Ctx) {
	backupForm := &form.DbBackupForm{}
	ginx.BindJsonAndValid(rc.GinCtx, backupForm)
	rc.ReqParam = backupForm

	dbNames := strings.Fields(backupForm.DbNames)
	biz.IsTrue(len(dbNames) > 0, "解析数据库备份任务失败：数据库名称未定义")

	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.DbApp.GetById(new(entity.Db), dbId, "instanceId")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	jobs := make([]*entity.DbBackup, 0, len(dbNames))
	for _, dbName := range dbNames {
		job := &entity.DbBackup{
			DbJobBaseImpl: entity.NewDbBJobBase(db.InstanceId, entity.DbJobTypeBackup),
			Enabled:       true,
			Repeated:      backupForm.Repeated,
			StartTime:     backupForm.StartTime,
			Interval:      backupForm.Interval,
			Name:          backupForm.Name,
		}
		job.DbName = dbName
		jobs = append(jobs, job)
	}
	biz.ErrIsNilAppendErr(d.DbBackupApp.Create(rc.MetaCtx, jobs), "添加数据库备份任务失败: %v")
}

// Update 保存数据库备份任务
// @router /api/dbs/:dbId/backups/:backupId [PUT]
func (d *DbBackup) Update(rc *req.Ctx) {
	backupForm := &form.DbBackupForm{}
	ginx.BindJsonAndValid(rc.GinCtx, backupForm)
	rc.ReqParam = backupForm

	job := entity.NewDbJob(entity.DbJobTypeBackup).(*entity.DbBackup)
	job.Id = backupForm.Id
	job.Name = backupForm.Name
	job.StartTime = backupForm.StartTime
	job.Interval = backupForm.Interval
	biz.ErrIsNilAppendErr(d.DbBackupApp.Update(rc.MetaCtx, job), "保存数据库备份任务失败: %v")
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
		backupId := uint64(value)
		err = fn(rc.MetaCtx, backupId)
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete 删除数据库备份任务
// @router /api/dbs/:dbId/backups/:backupId [DELETE]
func (d *DbBackup) Delete(rc *req.Ctx) {
	err := d.walk(rc, d.DbBackupApp.Delete)
	biz.ErrIsNilAppendErr(err, "删除数据库备份任务失败: %v")
}

// Enable 启用数据库备份任务
// @router /api/dbs/:dbId/backups/:backupId/enable [PUT]
func (d *DbBackup) Enable(rc *req.Ctx) {
	err := d.walk(rc, d.DbBackupApp.Enable)
	biz.ErrIsNilAppendErr(err, "启用数据库备份任务失败: %v")
}

// Disable 禁用数据库备份任务
// @router /api/dbs/:dbId/backups/:backupId/disable [PUT]
func (d *DbBackup) Disable(rc *req.Ctx) {
	err := d.walk(rc, d.DbBackupApp.Disable)
	biz.ErrIsNilAppendErr(err, "禁用数据库备份任务失败: %v")
}

// Start 禁用数据库备份任务
// @router /api/dbs/:dbId/backups/:backupId/start [PUT]
func (d *DbBackup) Start(rc *req.Ctx) {
	err := d.walk(rc, d.DbBackupApp.Start)
	biz.ErrIsNilAppendErr(err, "运行数据库备份任务失败: %v")
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

// GetHistoryPageList 获取数据库备份历史
// @router /api/dbs/:dbId/backups/:backupId/histories [GET]
func (d *DbBackup) GetHistoryPageList(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.DbApp.GetById(new(entity.Db), dbId, "db_instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	queryCond, page := ginx.BindQueryAndPage[*entity.DbBackupHistoryQuery](rc.GinCtx, new(entity.DbBackupHistoryQuery))
	queryCond.DbInstanceId = db.InstanceId
	queryCond.InDbNames = strings.Fields(db.Database)
	res, err := d.DbBackupApp.GetHistoryPageList(queryCond, page, new([]vo.DbBackupHistory))
	biz.ErrIsNilAppendErr(err, "获取数据库备份历史失败: %v")
	rc.ResData = res
}
