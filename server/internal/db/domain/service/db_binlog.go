package service

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
)

type DbBinlogSvc interface {
	AddTaskIfNotExists(ctx context.Context, task *entity.DbBinlog) error
	UpdateTask(ctx context.Context, task *entity.DbBinlog) error
	DeleteTask(ctx context.Context, taskId uint64) error
	EnableTask(ctx context.Context, taskId uint64) error
	DisableTask(ctx context.Context, taskId uint64) error
}
