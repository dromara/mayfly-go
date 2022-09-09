package application

import "mayfly-go/internal/db/infrastructure/persistence"

var (
	dbApp        Db        = newDbApp(persistence.GetDbRepo(), persistence.GetDbSqlRepo())
	dbSqlExecApp DbSqlExec = newDbSqlExecApp(persistence.GetDbSqlExecRepo())
)

func GetDbApp() Db {
	return dbApp
}

func GetDbSqlExecApp() DbSqlExec {
	return dbSqlExecApp
}
