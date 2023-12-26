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

func newDbSqlApp(dbSqlRepo repository.DbSql) DbSql {
	app := new(dbSqlAppImpl)
	app.Repo = dbSqlRepo
	return app
}
