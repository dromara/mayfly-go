package application

import (
	"mayfly-go/internal/db/infrastructure/persistence"
)

var (
	instanceApp  Instance  = newInstanceApp(persistence.GetInstanceRepo())
	dbApp        Db        = newDbApp(persistence.GetDbRepo(), persistence.GetDbSqlRepo(), instanceApp)
	dbSqlExecApp DbSqlExec = newDbSqlExecApp(persistence.GetDbSqlExecRepo())
)

func GetInstanceApp() Instance {
	return instanceApp
}

func GetDbApp() Db {
	return dbApp
}

func GetDbSqlExecApp() DbSqlExec {
	return dbSqlExecApp
}
