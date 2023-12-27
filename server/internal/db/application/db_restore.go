package application

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/domain/service"
	serviceImpl "mayfly-go/internal/db/infrastructure/service"
	"mayfly-go/pkg/model"
)

func newDbRestoreApp(repositories *repository.Repositories) (*DbRestoreApp, error) {
	dbRestoreSvc, err := serviceImpl.NewDbRestoreSvc(repositories)
	if err != nil {
		return nil, err
	}
	app := &DbRestoreApp{
		repo:         repositories.Restore,
		dbRestoreSvc: dbRestoreSvc,
	}
	return app, nil
}

type DbRestoreApp struct {
	repo         repository.DbRestore
	dbRestoreSvc service.DbRestoreSvc
}

func (app *DbRestoreApp) Create(ctx context.Context, tasks ...*entity.DbRestore) error {
	return app.dbRestoreSvc.AddTask(ctx, tasks...)
}

func (app *DbRestoreApp) Save(ctx context.Context, task *entity.DbRestore) error {
	return app.dbRestoreSvc.UpdateTask(ctx, task)
}

func (app *DbRestoreApp) Delete(ctx context.Context, taskId uint64) error {
	// todo: 删除数据库恢复历史文件
	return app.dbRestoreSvc.DeleteTask(ctx, taskId)
}

func (app *DbRestoreApp) Enable(ctx context.Context, taskId uint64) error {
	return app.dbRestoreSvc.EnableTask(ctx, taskId)
}

func (app *DbRestoreApp) Disable(ctx context.Context, taskId uint64) error {
	return app.dbRestoreSvc.DisableTask(ctx, taskId)
}

// GetPageList 分页获取数据库恢复任务
func (app *DbRestoreApp) GetPageList(condition *entity.DbRestoreQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.repo.GetDbRestoreList(condition, pageParam, toEntity, orderBy...)
}

// GetDbNamesWithoutRestore 获取未配置定时恢复的数据库名称
func (app *DbRestoreApp) GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error) {
	return app.repo.GetDbNamesWithoutRestore(instanceId, dbNames)
}
