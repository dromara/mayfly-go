package application

import (
	"context"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/model"
	"time"
)

func newDbRestoreApp(repositories *repository.Repositories, dbApp Db) (*DbRestoreApp, error) {
	app := &DbRestoreApp{
		restoreRepo:        repositories.Restore,
		instanceRepo:       repositories.Instance,
		backupHistoryRepo:  repositories.BackupHistory,
		restoreHistoryRepo: repositories.RestoreHistory,
		binlogHistoryRepo:  repositories.BinlogHistory,
		dbApp:              dbApp,
	}
	scheduler, err := newDbScheduler[*entity.DbRestore](
		repositories.Restore,
		withRunRestoreTask(app))
	if err != nil {
		return nil, err
	}
	app.scheduler = scheduler
	return app, nil
}

type DbRestoreApp struct {
	restoreRepo        repository.DbRestore
	instanceRepo       repository.Instance
	backupHistoryRepo  repository.DbBackupHistory
	restoreHistoryRepo repository.DbRestoreHistory
	binlogHistoryRepo  repository.DbBinlogHistory
	dbApp              Db
	scheduler          *dbScheduler[*entity.DbRestore]
}

func (app *DbRestoreApp) Close() {
	app.scheduler.Close()
}

func (app *DbRestoreApp) Create(ctx context.Context, tasks ...*entity.DbRestore) error {
	return app.scheduler.AddTask(ctx, tasks...)
}

func (app *DbRestoreApp) Save(ctx context.Context, task *entity.DbRestore) error {
	return app.scheduler.UpdateTask(ctx, task)
}

func (app *DbRestoreApp) Delete(ctx context.Context, taskId uint64) error {
	// todo: 删除数据库恢复历史文件
	return app.scheduler.DeleteTask(ctx, taskId)
}

func (app *DbRestoreApp) Enable(ctx context.Context, taskId uint64) error {
	return app.scheduler.EnableTask(ctx, taskId)
}

func (app *DbRestoreApp) Disable(ctx context.Context, taskId uint64) error {
	return app.scheduler.DisableTask(ctx, taskId)
}

// GetPageList 分页获取数据库恢复任务
func (app *DbRestoreApp) GetPageList(condition *entity.DbRestoreQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.restoreRepo.GetDbRestoreList(condition, pageParam, toEntity, orderBy...)
}

// GetDbNamesWithoutRestore 获取未配置定时恢复的数据库名称
func (app *DbRestoreApp) GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error) {
	return app.restoreRepo.GetDbNamesWithoutRestore(instanceId, dbNames)
}

// 分页获取数据库备份历史
func (app *DbRestoreApp) GetHistoryPageList(condition *entity.DbRestoreHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.restoreHistoryRepo.GetDbRestoreHistories(condition, pageParam, toEntity, orderBy...)
}

func (app *DbRestoreApp) runTask(ctx context.Context, task *entity.DbRestore) error {
	conn, err := app.dbApp.GetDbConnByInstanceId(task.DbInstanceId)
	if err != nil {
		return err
	}
	dbProgram := conn.GetDialect().GetDbProgram()
	if task.PointInTime.Valid {
		latestBinlogSequence, earliestBackupSequence := int64(-1), int64(-1)
		binlogHistory, ok, err := app.binlogHistoryRepo.GetLatestHistory(task.DbInstanceId)
		if err != nil {
			return err
		}
		if ok {
			latestBinlogSequence = binlogHistory.Sequence
		} else {
			backupHistory, err := app.backupHistoryRepo.GetEarliestHistory(task.DbInstanceId)
			if err != nil {
				return err
			}
			earliestBackupSequence = backupHistory.BinlogSequence
		}
		binlogFiles, err := dbProgram.FetchBinlogs(ctx, true, earliestBackupSequence, latestBinlogSequence)
		if err != nil {
			return err
		}
		if err := app.binlogHistoryRepo.InsertWithBinlogFiles(ctx, task.DbInstanceId, binlogFiles); err != nil {
			return err
		}
		if err := app.restorePointInTime(ctx, dbProgram, task); err != nil {
			return err
		}
	} else {
		if err := app.restoreBackupHistory(ctx, dbProgram, task); err != nil {
			return err
		}
	}

	history := &entity.DbRestoreHistory{
		CreateTime:  time.Now(),
		DbRestoreId: task.Id,
	}
	if err := app.restoreHistoryRepo.Insert(ctx, history); err != nil {
		return err
	}
	return nil
}

func (app *DbRestoreApp) restorePointInTime(ctx context.Context, program dbm.DbProgram, task *entity.DbRestore) error {
	binlogHistory, err := app.binlogHistoryRepo.GetHistoryByTime(task.DbInstanceId, task.PointInTime.Time)
	if err != nil {
		return err
	}
	position, err := program.GetBinlogEventPositionAtOrAfterTime(ctx, binlogHistory.FileName, task.PointInTime.Time)
	if err != nil {
		return err
	}
	target := &entity.BinlogInfo{
		FileName: binlogHistory.FileName,
		Sequence: binlogHistory.Sequence,
		Position: position,
	}
	backupHistory, err := app.backupHistoryRepo.GetLatestHistory(task.DbInstanceId, task.DbName, target)
	if err != nil {
		return err
	}
	start := &entity.BinlogInfo{
		FileName: backupHistory.BinlogFileName,
		Sequence: backupHistory.BinlogSequence,
		Position: backupHistory.BinlogPosition,
	}
	binlogHistories, err := app.binlogHistoryRepo.GetHistories(task.DbInstanceId, start, target)
	if err != nil {
		return err
	}
	restoreInfo := &dbm.RestoreInfo{
		BackupHistory:   backupHistory,
		BinlogHistories: binlogHistories,
		StartPosition:   backupHistory.BinlogPosition,
		TargetPosition:  target.Position,
		TargetTime:      task.PointInTime.Time,
	}
	if err := program.RestoreBackupHistory(ctx, backupHistory.DbName, backupHistory.DbBackupId, backupHistory.Uuid); err != nil {
		return err
	}
	return program.ReplayBinlog(ctx, task.DbName, task.DbName, restoreInfo)
}

func (app *DbRestoreApp) restoreBackupHistory(ctx context.Context, program dbm.DbProgram, task *entity.DbRestore) error {
	backupHistory := &entity.DbBackupHistory{}
	if err := app.backupHistoryRepo.GetById(backupHistory, task.DbBackupHistoryId); err != nil {
		return err
	}
	return program.RestoreBackupHistory(ctx, backupHistory.DbName, backupHistory.DbBackupId, backupHistory.Uuid)
}

func withRunRestoreTask(app *DbRestoreApp) dbSchedulerOption[*entity.DbRestore] {
	return func(scheduler *dbScheduler[*entity.DbRestore]) {
		scheduler.RunTask = app.runTask
	}
}
