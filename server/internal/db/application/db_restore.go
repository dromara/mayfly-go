package application

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	service "mayfly-go/internal/db/infrastructure/service"
	"mayfly-go/pkg/model"
	"time"
)

func newDbRestoreApp(repositories *repository.Repositories) (*DbRestoreApp, error) {
	scheduler, err := newDbScheduler[*entity.DbRestore](
		repositories.Restore,
		withRunRestoreTask(repositories))
	if err != nil {
		return nil, err
	}
	app := &DbRestoreApp{
		repo:         repositories.Restore,
		instanceRepo: repositories.Instance,
		scheduler:    scheduler,
	}
	return app, nil
}

type DbRestoreApp struct {
	repo         repository.DbRestore
	instanceRepo repository.Instance
	scheduler    *dbScheduler[*entity.DbRestore]
	//scheduler service.DbRestoreSvc
}

func (app *DbRestoreApp) Close() {
	app.Close()
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
	return app.repo.GetDbRestoreList(condition, pageParam, toEntity, orderBy...)
}

// GetDbNamesWithoutRestore 获取未配置定时恢复的数据库名称
func (app *DbRestoreApp) GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error) {
	return app.repo.GetDbNamesWithoutRestore(instanceId, dbNames)
}

func withRunRestoreTask(repositories *repository.Repositories) dbSchedulerOption[*entity.DbRestore] {
	return func(scheduler *dbScheduler[*entity.DbRestore]) {
		scheduler.RunTask = func(ctx context.Context, task *entity.DbRestore) error {
			instance := new(entity.DbInstance)
			if err := repositories.Instance.GetById(instance, task.DbInstanceId); err != nil {
				return err
			}
			if err := instance.PwdDecrypt(); err != nil {
				return err
			}
			if err := service.NewDbInstanceSvc(instance, repositories).Restore(ctx, task); err != nil {
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

func newDbRestoreHistoryApp(repositories *repository.Repositories) (*DbRestoreHistoryApp, error) {
	app := &DbRestoreHistoryApp{
		repo: repositories.RestoreHistory,
	}
	return app, nil
}

type DbRestoreHistoryApp struct {
	repo repository.DbRestoreHistory
}

// GetPageList 分页获取数据库备份历史
func (app *DbRestoreHistoryApp) GetPageList(condition *entity.DbRestoreHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.repo.GetDbRestoreHistories(condition, pageParam, toEntity, orderBy...)
}
