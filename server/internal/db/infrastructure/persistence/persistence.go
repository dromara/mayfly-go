package persistence

import (
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newInstanceRepo(), ioc.WithComponentName("DbInstanceRepo"))
	ioc.Register(newDbRepo(), ioc.WithComponentName("DbRepo"))
	ioc.Register(newDbSqlRepo(), ioc.WithComponentName("DbSqlRepo"))
	ioc.Register(newDbSqlExecRepo(), ioc.WithComponentName("DbSqlExecRepo"))
	ioc.Register(newDataSyncTaskRepo(), ioc.WithComponentName("DbDataSyncTaskRepo"))
	ioc.Register(newDataSyncLogRepo(), ioc.WithComponentName("DbDataSyncLogRepo"))

	ioc.Register(NewDbBackupRepo(), ioc.WithComponentName("DbBackupRepo"))
	ioc.Register(NewDbBackupHistoryRepo(), ioc.WithComponentName("DbBackupHistoryRepo"))
	ioc.Register(NewDbRestoreRepo(), ioc.WithComponentName("DbRestoreRepo"))
	ioc.Register(NewDbRestoreHistoryRepo(), ioc.WithComponentName("DbRestoreHistoryRepo"))
	ioc.Register(NewDbBinlogRepo(), ioc.WithComponentName("DbBinlogRepo"))
	ioc.Register(NewDbBinlogHistoryRepo(), ioc.WithComponentName("DbBinlogHistoryRepo"))
}

func GetInstanceRepo() repository.Instance {
	return ioc.Get[repository.Instance]("DbInstanceRepo")
}

func GetDbRepo() repository.Db {
	return ioc.Get[repository.Db]("DbRepo")
}

func GetDbSqlRepo() repository.DbSql {
	return ioc.Get[repository.DbSql]("DbSqlRepo")
}

func GetDbSqlExecRepo() repository.DbSqlExec {
	return ioc.Get[repository.DbSqlExec]("DbSqlExecRepo")
}

func GetDbBackupHistoryRepo() repository.DbBackupHistory {
	return ioc.Get[repository.DbBackupHistory]("DbBackupHistoryRepo")
}

func GetDbRestoreHistoryRepo() repository.DbRestoreHistory {
	return ioc.Get[repository.DbRestoreHistory]("DbRestoreHistoryRepo")
}

func GetDataSyncLogRepo() repository.DataSyncLog {
	return ioc.Get[repository.DataSyncLog]("DataSyncLogRepo")
}

func GetDataSyncTaskRepo() repository.DataSyncTask {
	return ioc.Get[repository.DataSyncTask]("DataSyncTaskRepo")
}
