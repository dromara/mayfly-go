package application

import (
	"mayfly-go/internal/db/application/mask"
	dbsync "mayfly-go/internal/db/application/sync"
	transfer "mayfly-go/internal/db/application/transfer"
	"mayfly-go/pkg/ioc"
	"sync"
)

func InitIoc() {
	ioc.Register(new(instanceAppImpl))
	ioc.Register(new(dbAppImpl))
	ioc.Register(new(dbSqlExecAppImpl))
	ioc.Register(new(dbSqlAppImpl))
	ioc.Register(new(dbsync.DataSyncAppImpl))
	ioc.Register(new(transfer.DbTransferAppImpl))
	ioc.Register(new(transfer.DbTransferFileAppImpl))
	ioc.Register(new(mask.MaskAppImpl))
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

func GetDbInstanceApp() Instance {
	return ioc.Get[Instance]()
}

func GetDbSqlExecApp() DbSqlExec {
	return ioc.Get[DbSqlExec]()
}

func GetDataSyncTaskApp() dbsync.DataSyncTask {
	return dbsync.GetDataSyncTaskApp()
}

func GetDbTransferTaskApp() transfer.DbTransferTask {
	return transfer.GetDbTransferTaskApp()
}

func GetMaskApp() mask.MaskApp {
	return mask.GetMaskApp()
}
