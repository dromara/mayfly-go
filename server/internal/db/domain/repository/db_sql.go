package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

type DbSQL interface {
	base.Repo[*entity.DbSQL]
}
