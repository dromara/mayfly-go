package persistence

import (
	"context"
	"gorm.io/gorm/clause"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
)

var _ repository.DbBinlog = (*dbBinlogRepoImpl)(nil)

type dbBinlogRepoImpl struct {
	base.RepoImpl[*entity.DbBinlog]
}

func NewDbBinlogRepo() repository.DbBinlog {
	return &dbBinlogRepoImpl{}
}

func (d *dbBinlogRepoImpl) UpdateEnabled(ctx context.Context, taskId uint64, enabled bool) error {
	cond := map[string]any{
		"id": taskId,
	}
	return d.Updates(cond, map[string]any{
		"enabled": enabled,
	})
}

func (d *dbBinlogRepoImpl) UpdateTaskStatus(ctx context.Context, task *entity.DbBinlog) error {
	task = &entity.DbBinlog{
		Model: model.Model{
			DeletedModel: model.DeletedModel{
				Id: task.Id,
			},
		},
		LastStatus: task.LastStatus,
		LastResult: task.LastResult,
		LastTime:   task.LastTime,
	}
	return d.UpdateById(ctx, task)
}

func (d *dbBinlogRepoImpl) AddTaskIfNotExists(ctx context.Context, task *entity.DbBinlog) error {
	return global.Db.Clauses(clause.OnConflict{DoNothing: true}).Create(task).Error
}
