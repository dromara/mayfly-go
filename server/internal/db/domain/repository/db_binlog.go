package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
)

type DbBinlog interface {
	DbJob[*entity.DbBinlog]

	AddJobIfNotExists(ctx context.Context, job *entity.DbBinlog) error
}
