package application

import (
	"context"
	"math"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/timex"
	"sync"
	"time"
)

type DbBinlogApp struct {
	scheduler         *dbScheduler               `inject:"DbScheduler"`
	binlogRepo        repository.DbBinlog        `inject:"DbBinlogRepo"`
	binlogHistoryRepo repository.DbBinlogHistory `inject:"DbBinlogHistoryRepo"`
	backupRepo        repository.DbBackup        `inject:"DbBackupRepo"`
	backupHistoryRepo repository.DbBackupHistory `inject:"DbBackupHistoryRepo"`
	instanceRepo      repository.Instance        `inject:"DbInstanceRepo"`
	dbApp             Db                         `inject:"DbApp"`

	context   context.Context
	cancel    context.CancelFunc
	waitGroup sync.WaitGroup
}

func newDbBinlogApp() *DbBinlogApp {
	ctx, cancel := context.WithCancel(context.Background())
	svc := &DbBinlogApp{
		context: ctx,
		cancel:  cancel,
	}
	return svc
}

func (app *DbBinlogApp) Init() error {
	app.context, app.cancel = context.WithCancel(context.Background())
	app.waitGroup.Add(1)
	go app.run()
	return nil
}

func (app *DbBinlogApp) run() {
	defer app.waitGroup.Done()

	for app.context.Err() == nil {
		if err := app.fetchBinlog(app.context); err != nil {
			timex.SleepWithContext(app.context, time.Minute)
			continue
		}
		if err := app.pruneBinlog(app.context); err != nil {
			timex.SleepWithContext(app.context, time.Minute)
			continue
		}
		timex.SleepWithContext(app.context, entity.BinlogDownloadInterval)
	}
}

func (app *DbBinlogApp) fetchBinlog(ctx context.Context) error {
	jobs, err := app.loadJobs(ctx)
	if err != nil {
		logx.Errorf("DbBinlogApp: 加载 BINLOG 同步任务失败: %s", err.Error())
		timex.SleepWithContext(app.context, time.Minute)
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := app.scheduler.AddJob(app.context, jobs); err != nil {
		logx.Error("DbBinlogApp: 添加 BINLOG 同步任务失败: ", err.Error())
		return err
	}
	return nil
}

func (app *DbBinlogApp) pruneBinlog(ctx context.Context) error {
	var jobs []*entity.DbBinlog
	if err := app.binlogRepo.ListByCond(map[string]any{}, &jobs); err != nil {
		logx.Error("DbBinlogApp: 获取 BINLOG 同步任务失败: ", err.Error())
		return err
	}
	for _, instance := range jobs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var histories []*entity.DbBinlogHistory
		backupHistory, backupHistoryExists, err := app.backupHistoryRepo.GetEarliestHistoryForBinlog(instance.Id)
		if err != nil {
			logx.Errorf("DbBinlogApp: 获取数据库备份历史失败: %s", err.Error())
			return err
		}
		var binlogSeq int64 = math.MaxInt64
		if backupHistoryExists {
			binlogSeq = backupHistory.BinlogSequence
		}
		if err := app.binlogHistoryRepo.GetHistoriesBeforeSequence(ctx, instance.Id, binlogSeq, &histories); err != nil {
			logx.Errorf("DbBinlogApp: 获取数据库 BINLOG 历史失败: %s", err.Error())
			return err
		}
		conn, err := app.dbApp.GetDbConnByInstanceId(instance.Id)
		if err != nil {
			logx.Errorf("DbBinlogApp: 创建数据库连接失败: %s", err.Error())
			return err
		}
		dbProgram, err := conn.GetDialect().GetDbProgram()
		if err != nil {
			logx.Errorf("DbBinlogApp: 获取数据库备份与恢复程序失败: %s", err.Error())
			return err
		}
		for i, history := range histories {
			// todo: 在避免并发访问的前提下删除本地最新的 BINLOG 文件
			if !backupHistoryExists && i == len(histories)-1 {
				// 暂不删除本地最新的 BINLOG 文件
				break
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err := dbProgram.PruneBinlog(history); err != nil {
				logx.Errorf("清理 BINLOG 文件失败: %v", err)
				continue
			}
			if err := app.binlogHistoryRepo.DeleteById(ctx, history.Id); err != nil {
				logx.Errorf("删除 BINLOG 历史失败: %v", err)
				continue
			}
		}
	}
	return nil
}

func (app *DbBinlogApp) loadJobs(ctx context.Context) ([]*entity.DbBinlog, error) {
	var instanceIds []uint64
	if err := app.backupRepo.ListDbInstances(true, true, &instanceIds); err != nil {
		return nil, err
	}
	jobs := make([]*entity.DbBinlog, 0, len(instanceIds))
	for _, id := range instanceIds {
		if ctx.Err() != nil {
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
	cancel := app.cancel
	if cancel == nil {
		return
	}
	app.cancel = nil
	cancel()
	app.waitGroup.Wait()
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
