package application

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
)

type DbSQL interface {
	base.App[*entity.DbSQL]
}

type dbSQLAppImpl struct {
	base.AppImpl[*entity.DbSQL, repository.DbSQL]
}

var _ DbSQL = (*dbSQLAppImpl)(nil)
