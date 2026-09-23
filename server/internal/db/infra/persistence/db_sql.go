package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
)

type dbSQLRepoImpl struct {
	base.RepoImpl[*entity.DbSQL]
}

var _ repository.DbSQL = (*dbSQLRepoImpl)(nil)

func newDbSQLRepo() repository.DbSQL {
	return &dbSQLRepoImpl{}
}
