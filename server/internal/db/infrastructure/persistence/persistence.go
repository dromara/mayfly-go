package persistence

import "mayfly-go/internal/db/domain/repository"

var (
	instanceRepo  repository.Instance  = newInstanceRepo()
	dbRepo        repository.Db        = newDbRepo()
	dbSqlRepo     repository.DbSql     = newDbSqlRepo()
	dbSqlExecRepo repository.DbSqlExec = newDbSqlExecRepo()
)

func GetInstanceRepo() repository.Instance {
	return instanceRepo
}

func GetDbRepo() repository.Db {
	return dbRepo
}

func GetDbSqlRepo() repository.DbSql {
	return dbSqlRepo
}

func GetDbSqlExecRepo() repository.DbSqlExec {
	return dbSqlExecRepo
}
