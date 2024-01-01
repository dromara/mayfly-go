package persistence

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type dbTaskBase[T model.ModelI] struct {
	base.RepoImpl[T]
}

func (d *dbTaskBase[T]) UpdateEnabled(_ context.Context, taskId uint64, enabled bool) error {
	cond := map[string]any{
		"id": taskId,
	}
	return d.Updates(cond, map[string]any{
		"enabled": enabled,
	})
}

func (d *dbTaskBase[T]) UpdateTaskStatus(ctx context.Context, task T) error {
	return d.UpdateById(ctx, task, "last_status", "last_result", "last_time")
}

func (d *dbTaskBase[T]) ListToDo() ([]T, error) {
	var tasks []T
	db := global.Db.Model(d.GetModel())
	err := db.Where("enabled = ?", true).
		Where(db.Where("repeated = ?", true).Or("last_status <> ?", entity.TaskSuccess)).
		Scopes(gormx.UndeleteScope).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (d *dbTaskBase[T]) ListRepeating() ([]T, error) {
	cond := map[string]any{
		"enabled":  true,
		"repeated": true,
	}
	var tasks []T
	if err := d.ListByCond(cond, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}
