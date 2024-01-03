package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/timex"
	"sync"
	"time"
)

const (
	binlogDownloadInterval = time.Minute * 15
)

type DbBinlogApp struct {
	binlogRepo        repository.DbBinlog
	binlogHistoryRepo repository.DbBinlogHistory
	backupRepo        repository.DbBackup
	backupHistoryRepo repository.DbBackupHistory
	dbApp             Db
	context           context.Context
	cancel            context.CancelFunc
	waitGroup         sync.WaitGroup
}

var (
	binlogResult = map[entity.TaskStatus]string{
		entity.TaskDelay:    "等待备份BINLOG",
		entity.TaskReady:    "准备备份BINLOG",
		entity.TaskReserved: "BINLOG备份中",
		entity.TaskSuccess:  "BINLOG备份成功",
		entity.TaskFailed:   "BINLOG备份失败",
	}
)

func newDbBinlogApp(repositories *repository.Repositories, dbApp Db) (*DbBinlogApp, error) {
	ctx, cancel := context.WithCancel(context.Background())
	svc := &DbBinlogApp{
		binlogRepo:        repositories.Binlog,
		binlogHistoryRepo: repositories.BinlogHistory,
		backupRepo:        repositories.Backup,
		backupHistoryRepo: repositories.BackupHistory,
		dbApp:             dbApp,
		context:           ctx,
		cancel:            cancel,
	}
	svc.waitGroup.Add(1)
	go svc.run()
	return svc, nil
}

func (app *DbBinlogApp) runTask(ctx context.Context, backup *entity.DbBackup) error {
	if err := app.AddTaskIfNotExists(ctx, entity.NewDbBinlog(backup.DbInstanceId)); err != nil {
		return err
	}
	latestBinlogSequence, earliestBackupSequence := int64(-1), int64(-1)
	binlogHistory, ok, err := app.binlogHistoryRepo.GetLatestHistory(backup.DbInstanceId)
	if err != nil {
		return err
	}
	if ok {
		latestBinlogSequence = binlogHistory.Sequence
	} else {
		backupHistory, err := app.backupHistoryRepo.GetEarliestHistory(backup.DbInstanceId)
		if err != nil {
			return err
		}
		earliestBackupSequence = backupHistory.BinlogSequence
	}
	conn, err := app.dbApp.GetDbConnByInstanceId(backup.DbInstanceId)
	if err != nil {
		return err
	}
	dbProgram := conn.GetDialect().GetDbProgram()
	binlogFiles, err := dbProgram.FetchBinlogs(ctx, false, earliestBackupSequence, latestBinlogSequence)
	if err == nil {
		err = app.binlogHistoryRepo.InsertWithBinlogFiles(ctx, backup.DbInstanceId, binlogFiles)
	}
	taskStatus := entity.TaskSuccess
	if err != nil {
		taskStatus = entity.TaskFailed
	}
	task := &entity.DbBinlog{}
	task.Id = backup.DbInstanceId
	return app.updateCurTask(ctx, taskStatus, err, task)
}

func (app *DbBinlogApp) run() {
	defer app.waitGroup.Done()

	for !app.closed() {
		app.fetchFromAllInstances()
		timex.SleepWithContext(app.context, binlogDownloadInterval)
	}
}

func (app *DbBinlogApp) fetchFromAllInstances() {
	tasks, err := app.backupRepo.ListRepeating()
	if err != nil {
		logx.Errorf("DbBinlogApp: 获取数据库备份任务失败: %s", err.Error())
		return
	}
	for _, task := range tasks {
		if app.closed() {
			break
		}
		if err := app.runTask(app.context, task); err != nil {
			logx.Errorf("DbBinlogApp: 下载 binlog 文件失败: %s", err.Error())
			return
		}
	}
}

func (app *DbBinlogApp) Close() {
	app.cancel()
	app.waitGroup.Wait()
}

func (app *DbBinlogApp) closed() bool {
	return app.context.Err() != nil
}

func (app *DbBinlogApp) AddTaskIfNotExists(ctx context.Context, task *entity.DbBinlog) error {
	if err := app.binlogRepo.AddTaskIfNotExists(ctx, task); err != nil {
		return err
	}
	if task.Id == 0 {
		return nil
	}
	return nil
}

func (app *DbBinlogApp) DeleteTask(ctx context.Context, taskId uint64) error {
	// todo: 删除 Binlog 历史文件
	if err := app.binlogRepo.DeleteById(ctx, taskId); err != nil {
		return err
	}
	return nil
}

func (app *DbBinlogApp) updateCurTask(ctx context.Context, status entity.TaskStatus, lastErr error, task *entity.DbBinlog) error {
	task.LastStatus = status
	var result = binlogResult[status]
	if lastErr != nil {
		result = fmt.Sprintf("%v: %v", binlogResult[status], lastErr)
	}
	task.LastResult = stringx.TruncateStr(result, entity.LastResultSize)
	task.LastTime = timex.NewNullTime(time.Now())
	return app.binlogRepo.UpdateById(ctx, task, "last_status", "last_result", "last_time")
}
