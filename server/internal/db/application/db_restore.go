package application

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/model"
)

type DbRestoreApp struct {
	scheduler          *dbScheduler                `inject:"DbScheduler"`
	restoreRepo        repository.DbRestore        `inject:"DbRestoreRepo"`
	restoreHistoryRepo repository.DbRestoreHistory `inject:"DbRestoreHistoryRepo"`
}

func (app *DbRestoreApp) Init() error {
	var jobs []*entity.DbRestore
	if err := app.restoreRepo.ListToDo(&jobs); err != nil {
		return err
	}
	if err := app.scheduler.AddJob(context.Background(), false, entity.DbJobTypeRestore, jobs); err != nil {
		return err
	}
	return nil
}

func (app *DbRestoreApp) Close() {
	app.scheduler.Close()
}

func (app *DbRestoreApp) Create(ctx context.Context, job *entity.DbRestore) error {
	return app.scheduler.AddJob(ctx, true /* 保存到数据库 */, entity.DbJobTypeRestore, job)
}

func (app *DbRestoreApp) Update(ctx context.Context, job *entity.DbRestore) error {
	return app.scheduler.UpdateJob(ctx, job)
}

func (app *DbRestoreApp) Delete(ctx context.Context, jobId uint64) error {
	// todo: 删除数据库恢复历史文件
	return app.scheduler.RemoveJob(ctx, entity.DbJobTypeRestore, jobId)
}

func (app *DbRestoreApp) Enable(ctx context.Context, jobId uint64) error {
	return app.scheduler.EnableJob(ctx, entity.DbJobTypeRestore, jobId)
}

func (app *DbRestoreApp) Disable(ctx context.Context, jobId uint64) error {
	return app.scheduler.DisableJob(ctx, entity.DbJobTypeRestore, jobId)
}

// GetPageList 分页获取数据库恢复任务
func (app *DbRestoreApp) GetPageList(condition *entity.DbJobQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.restoreRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// GetDbNamesWithoutRestore 获取未配置定时恢复的数据库名称
func (app *DbRestoreApp) GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error) {
	return app.restoreRepo.GetDbNamesWithoutRestore(instanceId, dbNames)
}

// GetHistoryPageList 分页获取数据库备份历史
func (app *DbRestoreApp) GetHistoryPageList(condition *entity.DbRestoreHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.restoreHistoryRepo.GetDbRestoreHistories(condition, pageParam, toEntity, orderBy...)
}
