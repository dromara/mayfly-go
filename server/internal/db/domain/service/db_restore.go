package service

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
)

type DbRestoreSvc interface {
	AddTask(ctx context.Context, tasks ...*entity.DbRestore) error
	UpdateTask(ctx context.Context, task *entity.DbRestore) error
	DeleteTask(ctx context.Context, taskId uint64) error
	EnableTask(ctx context.Context, taskId uint64) error
	DisableTask(ctx context.Context, taskId uint64) error
}
