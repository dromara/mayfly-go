package service

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/domain/service"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
	"time"
)

var _ service.DbRestoreSvc = (*DbRestoreSvcImpl)(nil)

type DbRestoreSvcImpl struct {
	repo         repository.DbRestore
	instanceRepo repository.Instance
	scheduler    *Scheduler[*entity.DbRestore]
}

func withRunRestoreTask(repositories *repository.Repositories) SchedulerOption[*entity.DbRestore] {
	return func(scheduler *Scheduler[*entity.DbRestore]) {
		scheduler.RunTask = func(ctx context.Context, task *entity.DbRestore) error {
			instance := new(entity.DbInstance)
			if err := repositories.Instance.GetById(instance, task.DbInstanceId); err != nil {
				return err
			}
			if err := instance.PwdDecrypt(); err != nil {
				return err
			}
			if err := NewDbInstanceSvc(instance, repositories).Restore(ctx, task); err != nil {
				return err
			}

			history := &entity.DbRestoreHistory{
				CreateTime:  time.Now(),
				DbRestoreId: task.Id,
			}
			if err := repositories.RestoreHistory.Insert(ctx, history); err != nil {
				return err
			}

			return nil
		}
	}
}

var (
	restoreResult = map[entity.TaskStatus]string{
		entity.TaskDelay:    "等待恢复数据库",
		entity.TaskReady:    "准备恢复数据库",
		entity.TaskReserved: "数据库恢复中",
		entity.TaskSuccess:  "数据库恢复成功",
		entity.TaskFailed:   "数据库恢复失败",
	}
)

func withUpdateRestoreStatus(repositories *repository.Repositories) SchedulerOption[*entity.DbRestore] {
	return func(scheduler *Scheduler[*entity.DbRestore]) {
		scheduler.UpdateTaskStatus = func(ctx context.Context, status entity.TaskStatus, lastErr error, task *entity.DbRestore) error {
			task.Finished = !task.Repeated && status == entity.TaskSuccess
			task.LastStatus = status
			var result = restoreResult[status]
			if lastErr != nil {
				result = fmt.Sprintf("%v: %v", restoreResult[status], lastErr)
			}
			task.LastResult = stringx.TruncateStr(result, entity.LastResultSize)
			task.LastTime = time.Now()
			return repositories.Restore.UpdateTaskStatus(ctx, task)
		}
	}
}

func NewDbRestoreSvc(repositories *repository.Repositories) (service.DbRestoreSvc, error) {
	scheduler, err := NewScheduler[*entity.DbRestore](
		withRunRestoreTask(repositories),
		withUpdateRestoreStatus(repositories))
	if err != nil {
		return nil, err
	}
	svc := &DbRestoreSvcImpl{
		repo:         repositories.Restore,
		instanceRepo: repositories.Instance,
		scheduler:    scheduler,
	}
	if err := svc.loadTasks(context.Background()); err != nil {
		return nil, err
	}
	return svc, nil
}

func (svc *DbRestoreSvcImpl) loadTasks(ctx context.Context) error {
	tasks := make([]*entity.DbRestore, 0, 64)
	cond := map[string]any{
		"Enabled":  true,
		"Finished": false,
	}
	if err := svc.repo.ListByCond(cond, &tasks); err != nil {
		return err
	}
	for _, task := range tasks {
		svc.scheduler.PushTask(ctx, task)
	}
	return nil
}

func (svc *DbRestoreSvcImpl) AddTask(ctx context.Context, tasks ...*entity.DbRestore) error {
	for _, task := range tasks {
		if err := svc.repo.AddTask(ctx, task); err != nil {
			return err
		}
		svc.scheduler.PushTask(ctx, task)
	}
	return nil
}

func (svc *DbRestoreSvcImpl) UpdateTask(ctx context.Context, task *entity.DbRestore) error {
	if err := svc.repo.UpdateById(ctx, task); err != nil {
		return err
	}
	svc.scheduler.UpdateTask(ctx, task)
	return nil
}

func (svc *DbRestoreSvcImpl) DeleteTask(ctx context.Context, taskId uint64) error {
	// todo: 删除数据库恢复历史文件
	if err := svc.repo.DeleteById(ctx, taskId); err != nil {
		return err
	}
	svc.scheduler.RemoveTask(taskId)
	return nil
}

func (svc *DbRestoreSvcImpl) EnableTask(ctx context.Context, taskId uint64) error {
	if err := svc.repo.UpdateEnabled(ctx, taskId, true); err != nil {
		return err
	}
	task := new(entity.DbRestore)
	if err := svc.repo.GetById(task, taskId); err != nil {
		return err
	}
	svc.scheduler.UpdateTask(ctx, task)
	return nil
}

func (svc *DbRestoreSvcImpl) DisableTask(ctx context.Context, taskId uint64) error {
	if err := svc.repo.UpdateEnabled(ctx, taskId, false); err != nil {
		return err
	}
	svc.scheduler.RemoveTask(taskId)
	return nil
}

// GetPageList 分页获取数据库恢复任务
func (svc *DbRestoreSvcImpl) GetPageList(condition *entity.DbRestoreQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return svc.repo.GetDbRestoreList(condition, pageParam, toEntity, orderBy...)
}
