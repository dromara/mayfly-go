package application

import (
	"context"
	"errors"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"sync"
)

type DbRestoreApp struct {
	scheduler          *dbScheduler                `inject:"DbScheduler"`
	restoreRepo        repository.DbRestore        `inject:"DbRestoreRepo"`
	restoreHistoryRepo repository.DbRestoreHistory `inject:"DbRestoreHistoryRepo"`
	mutex              sync.Mutex
}

func (app *DbRestoreApp) Init() error {
	var jobs []*entity.DbRestore
	if err := app.restoreRepo.ListToDo(&jobs); err != nil {
		return err
	}
	if err := app.scheduler.AddJob(context.Background(), jobs); err != nil {
		return err
	}
	return nil
}

func (app *DbRestoreApp) Close() {
	app.scheduler.Close()
}

func (app *DbRestoreApp) Create(ctx context.Context, jobs any) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if err := app.restoreRepo.AddJob(ctx, jobs); err != nil {
		return err
	}
	_ = app.scheduler.AddJob(ctx, jobs)
	return nil
}

func (app *DbRestoreApp) Update(ctx context.Context, job *entity.DbRestore) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if err := app.restoreRepo.UpdateById(ctx, job); err != nil {
		return err
	}
	_ = app.scheduler.UpdateJob(ctx, job)
	return nil
}

func (app *DbRestoreApp) Delete(ctx context.Context, jobId uint64) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if err := app.scheduler.RemoveJob(ctx, entity.DbJobTypeRestore, jobId); err != nil {
		return err
	}
	history := &entity.DbRestoreHistory{
		DbRestoreId: jobId,
	}
	if err := app.restoreHistoryRepo.DeleteByCond(ctx, history); err != nil {
		return err
	}
	if err := app.restoreRepo.DeleteById(ctx, jobId); err != nil {
		return err
	}
	return nil
}

func (app *DbRestoreApp) Enable(ctx context.Context, jobId uint64) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	repo := app.restoreRepo
	job := &entity.DbRestore{}
	if err := repo.GetById(job, jobId); err != nil {
		return err
	}
	if job.IsEnabled() {
		return nil
	}
	if job.IsExpired() {
		return errors.New("任务已过期")
	}
	_ = app.scheduler.EnableJob(ctx, job)
	if err := repo.UpdateEnabled(ctx, jobId, true); err != nil {
		logx.Errorf("数据库恢复任务已启用( jobId: %d )，任务状态保存失败: %v", jobId, err)
		return err
	}
	return nil
}

func (app *DbRestoreApp) Disable(ctx context.Context, jobId uint64) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	repo := app.restoreRepo
	job := &entity.DbRestore{}
	if err := repo.GetById(job, jobId); err != nil {
		return err
	}
	if !job.IsEnabled() {
		return nil
	}
	_ = app.scheduler.DisableJob(ctx, entity.DbJobTypeRestore, jobId)
	if err := repo.UpdateEnabled(ctx, jobId, false); err != nil {
		logx.Errorf("数据库恢复任务已禁用( jobId: %d )，任务状态保存失败: %v", jobId, err)
		return err
	}
	return nil
}

// GetPageList 分页获取数据库恢复任务
func (app *DbRestoreApp) GetPageList(condition *entity.DbRestoreQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.restoreRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// GetRestoresEnabled 获取数据库恢复任务
func (app *DbRestoreApp) GetRestoresEnabled(toEntity any, backupHistoryId ...uint64) error {
	return app.restoreRepo.GetEnabledRestores(toEntity, backupHistoryId...)
}

// GetDbNamesWithoutRestore 获取未配置定时恢复的数据库名称
func (app *DbRestoreApp) GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error) {
	return app.restoreRepo.GetDbNamesWithoutRestore(instanceId, dbNames)
}

// GetHistoryPageList 分页获取数据库备份历史
func (app *DbRestoreApp) GetHistoryPageList(condition *entity.DbRestoreHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.restoreHistoryRepo.GetDbRestoreHistories(condition, pageParam, toEntity, orderBy...)
}
