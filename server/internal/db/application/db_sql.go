package application

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
)

type DbSql interface {
	base.App[*entity.DbSql]
}

type dbSqlAppImpl struct {
	base.AppImpl[*entity.DbSql, repository.DbSql]
}

// 注入DbSqlRepo
func (d *dbSqlAppImpl) InjectDbSqlRepo(repo repository.DbSql) {
	d.Repo = repo
}
