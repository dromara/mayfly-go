package application

import (
	"mayfly-go/pkg/ioc"
	"sync"
)

func InitIoc() {
	ioc.Register(new(instanceAppImpl))
	ioc.Register(new(dbAppImpl))
	ioc.Register(new(dbSqlExecAppImpl))
	ioc.Register(new(dbSqlAppImpl))
	ioc.Register(new(dataSyncAppImpl))
	ioc.Register(new(dbTransferAppImpl))
	ioc.Register(new(dbTransferFileAppImpl))
}

func Init() {
	sync.OnceFunc(func() {
		GetDataSyncTaskApp().InitCronJob()
		GetDbTransferTaskApp().InitCronJob()
		GetDbTransferTaskApp().TimerDeleteTransferFile()
		InitDbFlowHandler()
	})()
}

func GetDbApp() Db {
	return ioc.Get[Db]()
}

func GetDbSqlExecApp() DbSqlExec {
	return ioc.Get[DbSqlExec]()
}

func GetDataSyncTaskApp() DataSyncTask {
	return ioc.Get[DataSyncTask]()
}

func GetDbTransferTaskApp() DbTransferTask {
	return ioc.Get[DbTransferTask]()
}
