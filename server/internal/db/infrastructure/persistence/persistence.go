package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(NewInstanceRepo(), ioc.WithComponentName("DbInstanceRepo"))
	ioc.Register(newDbRepo(), ioc.WithComponentName("DbRepo"))
	ioc.Register(newDbSqlRepo(), ioc.WithComponentName("DbSqlRepo"))
	ioc.Register(newDbSqlExecRepo(), ioc.WithComponentName("DbSqlExecRepo"))
	ioc.Register(newDataSyncTaskRepo(), ioc.WithComponentName("DbDataSyncTaskRepo"))
	ioc.Register(newDataSyncLogRepo(), ioc.WithComponentName("DbDataSyncLogRepo"))
	ioc.Register(newDbTransferTaskRepo(), ioc.WithComponentName("DbTransferTaskRepo"))
	ioc.Register(newDbTransferFileRepo(), ioc.WithComponentName("DbTransferFileRepo"))
}
