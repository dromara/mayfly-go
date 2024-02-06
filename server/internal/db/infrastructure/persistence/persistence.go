package persistence

import (
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(NewInstanceRepo(), ioc.WithComponentName("DbInstanceRepo"))
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
