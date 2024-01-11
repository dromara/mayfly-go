package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

type DbBinlog interface {
	base.Repo[*entity.DbBinlog]

	AddJobIfNotExists(ctx context.Context, job *entity.DbBinlog) error
}
