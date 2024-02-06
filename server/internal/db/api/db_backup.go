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
	"mayfly-go/pkg/utils/timex"
	"strconv"
	"strings"
	"time"
)

type DbBackup struct {
	backupApp  *application.DbBackupApp  `inject:"DbBackupApp"`
	dbApp      application.Db            `inject:"DbApp"`
	restoreApp *application.DbRestoreApp `inject:"DbRestoreApp"`
}

// todo: 鉴权，避免未经授权进行数据库备份和恢复

// GetPageList 获取数据库备份任务
// @router /api/dbs/:dbId/backups [GET]
func (d *DbBackup) GetPageList(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.dbApp.GetById(new(entity.Db), dbId, "db_instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	queryCond, page := ginx.BindQueryAndPage[*entity.DbBackupQuery](rc.GinCtx, new(entity.DbBackupQuery))
	queryCond.DbInstanceId = db.InstanceId
	queryCond.InDbNames = strings.Fields(db.Database)
	res, err := d.backupApp.GetPageList(queryCond, page, new([]vo.DbBackup))
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
	db, err := d.dbApp.GetById(new(entity.Db), dbId, "instanceId")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")
	jobs := make([]*entity.DbBackup, 0, len(dbNames))
	for _, dbName := range dbNames {
		job := &entity.DbBackup{
			DbInstanceId: db.InstanceId,
			DbName:       dbName,
			Enabled:      true,
			Repeated:     backupForm.Repeated,
			StartTime:    backupForm.StartTime,
			Interval:     backupForm.Interval,
			Name:         backupForm.Name,
		}
		jobs = append(jobs, job)
	}
	biz.ErrIsNilAppendErr(d.backupApp.Create(rc.MetaCtx, jobs), "添加数据库备份任务失败: %v")
}

// Update 保存数据库备份任务
// @router /api/dbs/:dbId/backups/:backupId [PUT]
func (d *DbBackup) Update(rc *req.Ctx) {
	backupForm := &form.DbBackupForm{}
	ginx.BindJsonAndValid(rc.GinCtx, backupForm)
	rc.ReqParam = backupForm

	job := &entity.DbBackup{}
	job.Id = backupForm.Id
	job.Name = backupForm.Name
	job.StartTime = backupForm.StartTime
	job.Interval = backupForm.Interval
	job.MaxSaveDays = backupForm.MaxSaveDays
	biz.ErrIsNilAppendErr(d.backupApp.Update(rc.MetaCtx, job), "保存数据库备份任务失败: %v")
}

func (d *DbBackup) walk(rc *req.Ctx, paramName string, fn func(ctx context.Context, id uint64) error) error {
	idsStr := ginx.PathParam(rc.GinCtx, paramName)
	biz.NotEmpty(idsStr, paramName+" 为空")
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
	err := d.walk(rc, "backupId", d.backupApp.Delete)
	biz.ErrIsNilAppendErr(err, "删除数据库备份任务失败: %v")
}

// Enable 启用数据库备份任务
// @router /api/dbs/:dbId/backups/:backupId/enable [PUT]
func (d *DbBackup) Enable(rc *req.Ctx) {
	err := d.walk(rc, "backupId", d.backupApp.Enable)
	biz.ErrIsNilAppendErr(err, "启用数据库备份任务失败: %v")
}

// Disable 禁用数据库备份任务
// @router /api/dbs/:dbId/backups/:backupId/disable [PUT]
func (d *DbBackup) Disable(rc *req.Ctx) {
	err := d.walk(rc, "backupId", d.backupApp.Disable)
	biz.ErrIsNilAppendErr(err, "禁用数据库备份任务失败: %v")
}

// Start 禁用数据库备份任务
// @router /api/dbs/:dbId/backups/:backupId/start [PUT]
func (d *DbBackup) Start(rc *req.Ctx) {
	err := d.walk(rc, "backupId", d.backupApp.StartNow)
	biz.ErrIsNilAppendErr(err, "运行数据库备份任务失败: %v")
}

// GetDbNamesWithoutBackup 获取未配置定时备份的数据库名称
// @router /api/dbs/:dbId/db-names-without-backup [GET]
func (d *DbBackup) GetDbNamesWithoutBackup(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	db, err := d.dbApp.GetById(new(entity.Db), dbId, "instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")
	dbNames := strings.Fields(db.Database)
	dbNamesWithoutBackup, err := d.backupApp.GetDbNamesWithoutBackup(db.InstanceId, dbNames)
	biz.ErrIsNilAppendErr(err, "获取未配置定时备份的数据库名称失败: %v")
	rc.ResData = dbNamesWithoutBackup
}

// GetHistoryPageList 获取数据库备份历史
// @router /api/dbs/:dbId/backups/:backupId/histories [GET]
func (d *DbBackup) GetHistoryPageList(rc *req.Ctx) {
	dbId := uint64(ginx.PathParamInt(rc.GinCtx, "dbId"))
	biz.IsTrue(dbId > 0, "无效的 dbId: %v", dbId)
	db, err := d.dbApp.GetById(new(entity.Db), dbId, "db_instance_id", "database")
	biz.ErrIsNilAppendErr(err, "获取数据库信息失败: %v")

	backupHistoryCond, page := ginx.BindQueryAndPage[*entity.DbBackupHistoryQuery](rc.GinCtx, new(entity.DbBackupHistoryQuery))
	backupHistoryCond.DbInstanceId = db.InstanceId
	backupHistoryCond.InDbNames = strings.Fields(db.Database)
	backupHistories := make([]*vo.DbBackupHistory, 0, page.PageSize)
	res, err := d.backupApp.GetHistoryPageList(backupHistoryCond, page, &backupHistories)
	biz.ErrIsNilAppendErr(err, "获取数据库备份历史失败: %v")
	historyIds := make([]uint64, 0, len(backupHistories))
	for _, history := range backupHistories {
		historyIds = append(historyIds, history.Id)
	}
	restores := make([]*entity.DbRestore, 0, page.PageSize)
	if err := d.restoreApp.GetRestoresEnabled(&restores, historyIds...); err != nil {
		biz.ErrIsNilAppendErr(err, "获取数据库备份恢复记录失败")
	}
	for _, history := range backupHistories {
		for _, restore := range restores {
			if restore.DbBackupHistoryId == history.Id {
				history.LastStatus = restore.LastStatus
				history.LastResult = restore.LastResult
				history.LastTime = restore.LastTime
				break
			}
		}
	}
	rc.ResData = res
}

// RestoreHistories 从数据库备份历史中恢复数据库
// @router /api/dbs/:dbId/backup-histories/:backupHistoryId/restore [POST]
func (d *DbBackup) RestoreHistories(rc *req.Ctx) {
	pm := ginx.PathParam(rc.GinCtx, "backupHistoryId")
	biz.NotEmpty(pm, "backupHistoryId 为空")
	idsStr := strings.Fields(pm)
	ids := make([]uint64, 0, len(idsStr))
	for _, s := range idsStr {
		id, err := strconv.ParseUint(s, 10, 64)
		biz.ErrIsNilAppendErr(err, "从数据库备份历史恢复数据库失败: %v")
		ids = append(ids, id)
	}
	histories := make([]*entity.DbBackupHistory, 0, len(ids))
	err := d.backupApp.GetHistories(ids, &histories)
	biz.ErrIsNilAppendErr(err, "添加数据库恢复任务失败: %v")
	restores := make([]*entity.DbRestore, 0, len(histories))
	now := time.Now()
	for _, history := range histories {
		job := &entity.DbRestore{
			DbInstanceId:        history.DbInstanceId,
			DbName:              history.DbName,
			Enabled:             true,
			Repeated:            false,
			StartTime:           now,
			Interval:            0,
			PointInTime:         timex.NewNullTime(time.Time{}),
			DbBackupId:          history.DbBackupId,
			DbBackupHistoryId:   history.Id,
			DbBackupHistoryName: history.Name,
		}
		restores = append(restores, job)
	}
	biz.ErrIsNilAppendErr(d.restoreApp.Create(rc.MetaCtx, restores), "添加数据库恢复任务失败: %v")
}

// DeleteHistories 删除数据库备份历史
// @router /api/dbs/:dbId/backup-histories/:backupHistoryId [DELETE]
func (d *DbBackup) DeleteHistories(rc *req.Ctx) {
	err := d.walk(rc, "backupHistoryId", d.backupApp.DeleteHistory)
	biz.ErrIsNilAppendErr(err, "删除数据库备份历史失败: %v")
}
