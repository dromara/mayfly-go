package application

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/timex"
	"sync"
	"time"
)

type DbBinlogApp struct {
	DbApp             Db                         `inject:"DbApp"`
	Scheduler         *dbScheduler               `inject:"DbScheduler"`
	BinlogRepo        repository.DbBinlog        `inject:"DbBinlogRepo"`
	BinlogHistoryRepo repository.DbBinlogHistory `inject:"DbBinlogHistoryRepo"`
	BackupRepo        repository.DbBackup        `inject:"DbBackupRepo"`
	BackupHistoryRepo repository.DbBackupHistory `inject:"DbBackupHistoryRepo"`
	context           context.Context
	cancel            context.CancelFunc
	waitGroup         sync.WaitGroup
}

func newDbBinlogApp() *DbBinlogApp {
	ctx, cancel := context.WithCancel(context.Background())
	svc := &DbBinlogApp{
		context: ctx,
		cancel:  cancel,
	}
	svc.waitGroup.Add(1)
	go svc.run()
	return svc
}

func (app *DbBinlogApp) run() {
	defer app.waitGroup.Done()

	// todo: 实现 binlog 并发下载
	timex.SleepWithContext(app.context, time.Minute)
	for !app.closed() {
		jobs, err := app.loadJobs()
		if err != nil {
			logx.Errorf("DbBinlogApp: 加载 BINLOG 同步任务失败: %s", err.Error())
			timex.SleepWithContext(app.context, time.Minute)
			continue
		}
		if app.closed() {
			break
		}
		if err := app.Scheduler.AddJob(app.context, false, entity.DbJobTypeBinlog, jobs); err != nil {
			logx.Error("DbBinlogApp: 添加 BINLOG 同步任务失败: ", err.Error())
		}
		timex.SleepWithContext(app.context, entity.BinlogDownloadInterval)
	}
}

func (app *DbBinlogApp) loadJobs() ([]*entity.DbBinlog, error) {
	var instanceIds []uint64
	if err := app.BackupRepo.ListDbInstances(true, true, &instanceIds); err != nil {
		return nil, err
	}
	jobs := make([]*entity.DbBinlog, 0, len(instanceIds))
	for _, id := range instanceIds {
		if app.closed() {
			break
		}
		binlog := entity.NewDbBinlog(id)
		if err := app.AddJobIfNotExists(app.context, binlog); err != nil {
			return nil, err
		}
		jobs = append(jobs, binlog)
	}
	return jobs, nil
}

func (app *DbBinlogApp) Close() {
	app.cancel()
	app.waitGroup.Wait()
}

func (app *DbBinlogApp) closed() bool {
	return app.context.Err() != nil
}

func (app *DbBinlogApp) AddJobIfNotExists(ctx context.Context, job *entity.DbBinlog) error {
	if err := app.BinlogRepo.AddJobIfNotExists(ctx, job); err != nil {
		return err
	}
	if job.Id == 0 {
		return nil
	}
	return nil
}

func (app *DbBinlogApp) Delete(ctx context.Context, jobId uint64) error {
	// todo: 删除 Binlog 历史文件
	if err := app.BinlogRepo.DeleteById(ctx, jobId); err != nil {
		return err
	}
	return nil
}
