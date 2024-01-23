package application

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/model"
)

type DbRestoreApp struct {
	DbApp              Db                          `inject:"DbApp"`
	Scheduler          *dbScheduler                `inject:"DbScheduler"`
	InstanceRepo       repository.Instance         `inject:"DbInstanceRepo"`
	BackupHistoryRepo  repository.DbBackupHistory  `inject:"DbBackupHistoryRepo"`
	RestoreRepo        repository.DbRestore        `inject:"DbRestoreRepo"`
	RestoreHistoryRepo repository.DbRestoreHistory `inject:"DbRestoreHistoryRepo"`
	BinlogHistoryRepo  repository.DbBinlogHistory  `inject:"DbBinlogHistoryRepo"`
}

func (app *DbRestoreApp) Init() error {
	var jobs []*entity.DbRestore
	if err := app.RestoreRepo.ListToDo(&jobs); err != nil {
		return err
	}
	if err := app.Scheduler.AddJob(context.Background(), false, entity.DbJobTypeRestore, jobs); err != nil {
		return err
	}
	return nil
}

func (app *DbRestoreApp) Close() {
	app.Scheduler.Close()
}

func (app *DbRestoreApp) Create(ctx context.Context, job *entity.DbRestore) error {
	return app.Scheduler.AddJob(ctx, true /* 保存到数据库 */, entity.DbJobTypeRestore, job)
}

func (app *DbRestoreApp) Update(ctx context.Context, job *entity.DbRestore) error {
	return app.Scheduler.UpdateJob(ctx, job)
}

func (app *DbRestoreApp) Delete(ctx context.Context, jobId uint64) error {
	// todo: 删除数据库恢复历史文件
	return app.Scheduler.RemoveJob(ctx, entity.DbJobTypeRestore, jobId)
}

func (app *DbRestoreApp) Enable(ctx context.Context, jobId uint64) error {
	return app.Scheduler.EnableJob(ctx, entity.DbJobTypeRestore, jobId)
}

func (app *DbRestoreApp) Disable(ctx context.Context, jobId uint64) error {
	return app.Scheduler.DisableJob(ctx, entity.DbJobTypeRestore, jobId)
}

// GetPageList 分页获取数据库恢复任务
func (app *DbRestoreApp) GetPageList(condition *entity.DbJobQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.RestoreRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// GetDbNamesWithoutRestore 获取未配置定时恢复的数据库名称
func (app *DbRestoreApp) GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error) {
	return app.RestoreRepo.GetDbNamesWithoutRestore(instanceId, dbNames)
}

// GetHistoryPageList 分页获取数据库备份历史
func (app *DbRestoreApp) GetHistoryPageList(condition *entity.DbRestoreHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.RestoreHistoryRepo.GetDbRestoreHistories(condition, pageParam, toEntity, orderBy...)
}
