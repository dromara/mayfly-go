package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
)

type DbBinlog interface {
	DbJob

	AddJobIfNotExists(ctx context.Context, job *entity.DbBinlog) error
}
