package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

type DbSql interface {
	base.Repo[*entity.DbSql]
}
