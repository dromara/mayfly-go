package service

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/domain/service"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/timex"
	"time"
)

var _ service.DbBinlogSvc = (*DbBinlogSvcImpl)(nil)

type DbBinlogSvcImpl struct {
	repo         repository.DbBinlog
	instanceRepo repository.Instance
	scheduler    *Scheduler[*entity.DbBinlog]
}

func withDownloadBinlog(repositories *repository.Repositories) SchedulerOption[*entity.DbBinlog] {
	return func(scheduler *Scheduler[*entity.DbBinlog]) {
		scheduler.RunTask = func(ctx context.Context, task *entity.DbBinlog) error {
			instance := new(entity.DbInstance)
			if err := repositories.Instance.GetById(instance, task.DbInstanceId); err != nil {
				return err
			}
			if err := instance.PwdDecrypt(); err != nil {
				return err
			}
			svc := NewDbInstanceSvc(instance, repositories)
			err := svc.FetchBinlogs(ctx, false)
			if err != nil {
				return err
			}
			return nil
		}
	}
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

func withUpdateBinlogStatus(repositories *repository.Repositories) SchedulerOption[*entity.DbBinlog] {
	return func(scheduler *Scheduler[*entity.DbBinlog]) {
		scheduler.UpdateTaskStatus = func(ctx context.Context, status entity.TaskStatus, lastErr error, task *entity.DbBinlog) error {
			task.LastStatus = status
			var result = backupResult[status]
			if lastErr != nil {
				result = fmt.Sprintf("%v: %v", binlogResult[status], lastErr)
			}
			task.LastResult = stringx.TruncateStr(result, entity.LastResultSize)
			task.LastTime = timex.NewNullTime(time.Now())
			return repositories.Binlog.UpdateTaskStatus(ctx, task)
		}
	}
}

func NewDbBinlogSvc(repositories *repository.Repositories) (service.DbBinlogSvc, error) {
	scheduler, err := NewScheduler[*entity.DbBinlog](withDownloadBinlog(repositories), withUpdateBinlogStatus(repositories))
	if err != nil {
		return nil, err
	}
	svc := &DbBinlogSvcImpl{
		repo:         repositories.Binlog,
		instanceRepo: repositories.Instance,
		scheduler:    scheduler,
	}
	err = svc.loadTasks(context.Background())
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func (svc *DbBinlogSvcImpl) loadTasks(ctx context.Context) error {
	tasks := make([]*entity.DbBinlog, 0, 64)
	cond := map[string]any{
		"Enabled": true,
	}
	if err := svc.repo.ListByCond(cond, &tasks); err != nil {
		return err
	}
	for _, task := range tasks {
		svc.scheduler.PushTask(ctx, task)
	}
	return nil
}

func (svc *DbBinlogSvcImpl) AddTaskIfNotExists(ctx context.Context, task *entity.DbBinlog) error {
	if err := svc.repo.AddTaskIfNotExists(ctx, task); err != nil {
		return err
	}
	if task.GetId() == 0 {
		return nil
	}
	svc.scheduler.PushTask(ctx, task)
	return nil
}

func (svc *DbBinlogSvcImpl) UpdateTask(ctx context.Context, task *entity.DbBinlog) error {
	if err := svc.repo.UpdateById(ctx, task); err != nil {
		return err
	}
	svc.scheduler.UpdateTask(ctx, task)
	return nil
}

func (svc *DbBinlogSvcImpl) DeleteTask(ctx context.Context, taskId uint64) error {
	// todo: 删除 Binlog 历史文件
	if err := svc.repo.DeleteById(ctx, taskId); err != nil {
		return err
	}
	svc.scheduler.RemoveTask(taskId)
	return nil
}

func (svc *DbBinlogSvcImpl) EnableTask(ctx context.Context, taskId uint64) error {
	if err := svc.repo.UpdateEnabled(ctx, taskId, true); err != nil {
		return err
	}
	task := new(entity.DbBinlog)
	if err := svc.repo.GetById(task, taskId); err != nil {
		return err
	}
	svc.scheduler.UpdateTask(ctx, task)
	return nil
}

func (svc *DbBinlogSvcImpl) DisableTask(ctx context.Context, taskId uint64) error {
	if err := svc.repo.UpdateEnabled(ctx, taskId, false); err != nil {
		return err
	}
	svc.scheduler.RemoveTask(taskId)
	return nil
}
