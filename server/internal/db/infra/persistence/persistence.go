package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(NewInstanceRepo())
	ioc.Register(newDbRepo())
	ioc.Register(newDbSqlRepo())
	ioc.Register(newDbSqlExecRepo())
	ioc.Register(newDataSyncTaskRepo())
	ioc.Register(newDataSyncLogRepo())
	ioc.Register(newDbTransferTaskRepo())
	ioc.Register(newDbTransferFileRepo())
}
