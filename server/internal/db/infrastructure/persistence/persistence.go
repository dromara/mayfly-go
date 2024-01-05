package persistence

import "mayfly-go/internal/db/domain/repository"

var (
	instanceRepo         repository.Instance     = newInstanceRepo()
	dbRepo               repository.Db           = newDbRepo()
	dbSqlRepo            repository.DbSql        = newDbSqlRepo()
	dbSqlExecRepo        repository.DbSqlExec    = newDbSqlExecRepo()
	dbBackupHistoryRepo                          = NewDbBackupHistoryRepo()
	dbRestoreHistoryRepo                         = NewDbRestoreHistoryRepo()
	dbDataSyncTaskRepo   repository.DataSyncTask = newDataSyncTaskRepo()
	dbDataSyncLogRepo    repository.DataSyncLog  = newDataSyncLogRepo()
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

func GetDbBackupHistoryRepo() repository.DbBackupHistory {
	return dbBackupHistoryRepo
}

func GetDbRestoreHistoryRepo() repository.DbRestoreHistory {
	return dbRestoreHistoryRepo
}

func GetDataSyncLogRepo() repository.DataSyncLog {
	return dbDataSyncLogRepo
}

func GetDataSyncTaskRepo() repository.DataSyncTask {
	return dbDataSyncTaskRepo
}
