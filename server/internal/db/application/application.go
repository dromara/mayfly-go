package application

import (
	"mayfly-go/internal/db/infrastructure/persistence"
)

var (
	instanceApp  Instance  = newInstanceApp(persistence.GetInstanceRepo())
	dbApp        Db        = newDbApp(persistence.GetDbRepo(), persistence.GetDbSqlRepo(), instanceApp)
	dbSqlExecApp DbSqlExec = newDbSqlExecApp(persistence.GetDbSqlExecRepo())
	dbSqlApp     DbSql     = newDbSqlApp(persistence.GetDbSqlRepo())
)

func GetInstanceApp() Instance {
	return instanceApp
}

func GetDbApp() Db {
	return dbApp
}

func GetDbSqlApp() DbSql {
	return dbSqlApp
}

func GetDbSqlExecApp() DbSqlExec {
	return dbSqlExecApp
}
