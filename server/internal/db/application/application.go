package application

import (
	"mayfly-go/pkg/ioc"
	"sync"
)

func InitIoc() {
	ioc.Register(new(instanceAppImpl), ioc.WithComponentName("DbInstanceApp"))
	ioc.Register(new(dbAppImpl), ioc.WithComponentName("DbApp"))
	ioc.Register(new(dbSqlExecAppImpl), ioc.WithComponentName("DbSqlExecApp"))
	ioc.Register(new(dbSqlAppImpl), ioc.WithComponentName("DbSqlApp"))
	ioc.Register(new(dataSyncAppImpl), ioc.WithComponentName("DbDataSyncTaskApp"))
	ioc.Register(new(dbTransferAppImpl), ioc.WithComponentName("DbTransferTaskApp"))
	ioc.Register(new(dbTransferFileAppImpl), ioc.WithComponentName("DbTransferFileApp"))
}

func Init() {
	sync.OnceFunc(func() {
		GetDataSyncTaskApp().InitCronJob()
		GetDbTransferTaskApp().InitCronJob()
		GetDbTransferTaskApp().TimerDeleteTransferFile()
		InitDbFlowHandler()
	})()
}

func GetDbSqlExecApp() DbSqlExec {
	return ioc.Get[DbSqlExec]("DbSqlExecApp")
}

func GetDataSyncTaskApp() DataSyncTask {
	return ioc.Get[DataSyncTask]("DbDataSyncTaskApp")
}

func GetDbTransferTaskApp() DbTransferTask {
	return ioc.Get[DbTransferTask]("DbTransferTaskApp")
}
