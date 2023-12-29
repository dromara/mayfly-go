package application

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/domain/service"
	service2 "mayfly-go/internal/db/infrastructure/service"
	"mayfly-go/pkg/model"
)

func newDbBackupApp(repositories *repository.Repositories) (*DbBackupApp, error) {
	binlogSvc, err := service2.NewDbBinlogSvc(repositories)
	if err != nil {
		return nil, err
	}
	dbBackupSvc, err := service2.NewDbBackupSvc(repositories, binlogSvc)
	if err != nil {
		return nil, err
	}

	app := &DbBackupApp{
		repo:        repositories.Backup,
		dbBackupSvc: dbBackupSvc,
	}
	return app, nil
}

type DbBackupApp struct {
	repo        repository.DbBackup
	dbBackupSvc service.DbBackupSvc
}

func (app *DbBackupApp) Create(ctx context.Context, tasks ...*entity.DbBackup) error {
	return app.dbBackupSvc.AddTask(ctx, tasks...)
}

func (app *DbBackupApp) Save(ctx context.Context, task *entity.DbBackup) error {
	return app.dbBackupSvc.UpdateTask(ctx, task)
}

func (app *DbBackupApp) Delete(ctx context.Context, taskId uint64) error {
	// todo: 删除数据库备份历史文件
	return app.dbBackupSvc.DeleteTask(ctx, taskId)
}

func (app *DbBackupApp) Enable(ctx context.Context, taskId uint64) error {
	return app.dbBackupSvc.EnableTask(ctx, taskId)
}

func (app *DbBackupApp) Disable(ctx context.Context, taskId uint64) error {
	return app.dbBackupSvc.DisableTask(ctx, taskId)
}

// GetPageList 分页获取数据库备份任务
func (app *DbBackupApp) GetPageList(condition *entity.DbBackupQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.repo.GetDbBackupList(condition, pageParam, toEntity, orderBy...)
}

// GetDbNamesWithoutBackup 获取未配置定时备份的数据库名称
func (app *DbBackupApp) GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error) {
	return app.repo.GetDbNamesWithoutBackup(instanceId, dbNames)
}
