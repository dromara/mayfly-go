package service

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
)

type DbBackupSvc interface {
	AddTask(ctx context.Context, tasks ...*entity.DbBackup) error
	UpdateTask(ctx context.Context, task *entity.DbBackup) error
	DeleteTask(ctx context.Context, taskId uint64) error
	EnableTask(ctx context.Context, taskId uint64) error
	DisableTask(ctx context.Context, taskId uint64) error
}
