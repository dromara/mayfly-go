package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
)

type dbSqlRepoImpl struct {
	base.RepoImpl[*entity.DbSql]
}

var _ repository.DbSql = (*dbSqlRepoImpl)(nil)

func newDbSqlRepo() repository.DbSql {
	return &dbSqlRepoImpl{}
}
