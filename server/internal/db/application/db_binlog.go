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
	binlogResult = map[entity.DbJobStatus]string{
		entity.DbJobDelay:   "等待备份BINLOG",
		entity.DbJobReady:   "准备备份BINLOG",
		entity.DbJobRunning: "BINLOG备份中",
		entity.DbJobSuccess: "BINLOG备份成功",
		entity.DbJobFailed:  "BINLOG备份失败",
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

func (app *DbBinlogApp) fetchBinlog(ctx context.Context, backup *entity.DbBackup) error {
	if err := app.AddJobIfNotExists(ctx, entity.NewDbBinlog(backup.DbInstanceId)); err != nil {
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
	jobStatus := entity.DbJobSuccess
	if err != nil {
		jobStatus = entity.DbJobFailed
	}
	job := &entity.DbBinlog{}
	job.Id = backup.DbInstanceId
	return app.updateCurJob(ctx, jobStatus, err, job)
}

func (app *DbBinlogApp) run() {
	defer app.waitGroup.Done()

	for !app.closed() {
		app.fetchFromAllInstances()
		timex.SleepWithContext(app.context, binlogDownloadInterval)
	}
}

func (app *DbBinlogApp) fetchFromAllInstances() {
	var backups []*entity.DbBackup
	if err := app.backupRepo.ListRepeating(&backups); err != nil {
		logx.Errorf("DbBinlogApp: 获取数据库备份任务失败: %s", err.Error())
		return
	}
	for _, backup := range backups {
		if app.closed() {
			break
		}
		if err := app.fetchBinlog(app.context, backup); err != nil {
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

func (app *DbBinlogApp) AddJobIfNotExists(ctx context.Context, job *entity.DbBinlog) error {
	if err := app.binlogRepo.AddJobIfNotExists(ctx, job); err != nil {
		return err
	}
	if job.Id == 0 {
		return nil
	}
	return nil
}

func (app *DbBinlogApp) Delete(ctx context.Context, jobId uint64) error {
	// todo: 删除 Binlog 历史文件
	if err := app.binlogRepo.DeleteById(ctx, jobId); err != nil {
		return err
	}
	return nil
}

func (app *DbBinlogApp) updateCurJob(ctx context.Context, status entity.DbJobStatus, lastErr error, job *entity.DbBinlog) error {
	job.LastStatus = status
	var result = binlogResult[status]
	if lastErr != nil {
		result = fmt.Sprintf("%v: %v", binlogResult[status], lastErr)
	}
	job.LastResult = stringx.TruncateStr(result, entity.LastResultSize)
	job.LastTime = timex.NewNullTime(time.Now())
	return app.binlogRepo.UpdateById(ctx, job, "last_status", "last_result", "last_time")
}
