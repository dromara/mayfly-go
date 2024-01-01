package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/infrastructure/service"
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
	backupRepo        repository.DbBackup
	backupHistoryRepo repository.DbBackupHistory
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

func newDbBinlogApp(repositories *repository.Repositories) (*DbBinlogApp, error) {
	context, cancel := context.WithCancel(context.Background())
	svc := &DbBinlogApp{
		binlogRepo:        repositories.Binlog,
		backupRepo:        repositories.Backup,
		backupHistoryRepo: repositories.BackupHistory,
		context:           context,
		cancel:            cancel,
	}
	svc.waitGroup.Add(1)
	go svc.run()
	return svc, nil
}

func (app *DbBinlogApp) runTask(ctx context.Context, backup *entity.DbBackup) error {
	backupHistory, ok, err := app.backupHistoryRepo.GetEarliestHistory(backup.DbInstanceId)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if err := app.AddTaskIfNotExists(ctx, entity.NewDbBinlog(backupHistory)); err != nil {
		return err
	}

	instance := new(entity.DbInstance)
	if err := repositories.Instance.GetById(instance, backup.DbInstanceId); err != nil {
		return err
	}
	if err := instance.PwdDecrypt(); err != nil {
		return err
	}

	// todo: 将 FetchBinlogs() 迁移到 DbBinlogApp, 避免 instanceSvc 访问 mayfly 数据库
	instSvc := service.NewDbInstanceSvc(instance, repositories)
	err = instSvc.FetchBinlogs(ctx, false)
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
