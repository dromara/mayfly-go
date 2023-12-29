package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
)

type DbBinlog interface {
	DbTask[*entity.DbBinlog]

	AddTaskIfNotExists(ctx context.Context, task *entity.DbBinlog) error
}
