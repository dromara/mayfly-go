package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

type DbBinlog interface {
	base.Repo[*entity.DbBinlog]

	AddTaskIfNotExists(ctx context.Context, task *entity.DbBinlog) error
	UpdateTaskStatus(ctx context.Context, task *entity.DbBinlog) error
	UpdateEnabled(ctx context.Context, taskId uint64, enabled bool) error
}
