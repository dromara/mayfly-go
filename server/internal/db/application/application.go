package application

import (
	"mayfly-go/internal/db/application/mask"
	dbschema "mayfly-go/internal/db/application/schema"
	"mayfly-go/internal/db/application/sqlexec"
	dbsync "mayfly-go/internal/db/application/sync"
	transfer "mayfly-go/internal/db/application/transfer"
	"mayfly-go/pkg/ioc"
	"sync"
)

func InitIoc() {
	ioc.Register(new(instanceAppImpl))
	ioc.Register(new(dbAppImpl))
	ioc.Register(new(sqlexec.DbSQLExecAppImpl))
	ioc.Register(new(dbDataImportAppImpl))
	ioc.Register(new(sqlexec.DbSQLAppImpl))
	ioc.Register(new(dbsync.DataSyncAppImpl))
	ioc.Register(new(transfer.DbTransferAppImpl))
	ioc.Register(new(transfer.DbTransferFileAppImpl))
	ioc.Register(new(mask.MaskAppImpl))
	ioc.Register(new(dbschema.SchemaDiffAppImpl))
	ioc.Register(new(dbschema.SchemaBaselineAppImpl))
}

func Init() {
	sync.OnceFunc(func() {
		GetDataSyncTaskApp().InitCronJob()
		GetDbTransferTaskApp().InitCronJob()
		GetDbTransferTaskApp().TimerDeleteTransferFile()
		sqlexec.InitDbFlowHandler()
		// 表结构漂移巡检：基线只有持续重比才有意义，不巡检就是存了一堆没人看的哈希
		dbschema.InitDriftCron()
	})()
}

func GetDbApp() Db {
	return ioc.Get[Db]()
}

func GetDbInstanceApp() Instance {
	return ioc.Get[Instance]()
}

func GetDbSQLExecApp() sqlexec.DbSQLExec {
	return sqlexec.GetDbSQLExecApp()
}

// GetDataSyncTaskApp 获取数据同步任务应用门面
func GetDataSyncTaskApp() dbsync.DataSyncTask {
	return dbsync.GetDataSyncTaskApp()
}

func GetDbTransferTaskApp() transfer.DbTransferTask {
	return transfer.GetDbTransferTaskApp()
}

func GetMaskApp() mask.MaskApp {
	return mask.GetMaskApp()
}

// GetSchemaDiffApp 结构对比（库间表结构漂移检测）
func GetSchemaDiffApp() dbschema.SchemaDiff {
	return dbschema.GetSchemaDiffApp()
}

// GetMaskEngine 获取通用脱敏引擎（供任意模块使用）
func GetMaskEngine() mask.MaskEngine {
	return mask.GetMaskEngine()
}
