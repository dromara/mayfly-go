package persistence

import (
	"context"
	"mayfly-go/pkg/base"
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
	//t := &entity.DbBinlog{
	//	Model: model.Model{
	//		DeletedModel: model.DeletedModel{
	//			Id: task.Id,
	//		},
	//	},
	//	Finished:   task.Finished,
	//	LastStatus: task.LastStatus,
	//	LastResult: task.LastResult,
	//	LastTime:   task.LastTime,
	//}
	return d.UpdateById(ctx, task, "finished", "last_status", "last_result", "last_time")
}
