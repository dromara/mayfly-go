package application

import (
	"fmt"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
	"sync"
)

var (
	dbBackupApp  *DbBackupApp
	dbRestoreApp *DbRestoreApp
	dbBinlogApp  *DbBinlogApp
)

func InitIoc() {
	persistence.Init()

	ioc.Register(new(instanceAppImpl), ioc.WithComponentName("DbInstanceApp"))
	ioc.Register(new(dbAppImpl), ioc.WithComponentName("DbApp"))
	ioc.Register(new(dbSqlExecAppImpl), ioc.WithComponentName("DbSqlExecApp"))
	ioc.Register(new(dbSqlAppImpl), ioc.WithComponentName("DbSqlApp"))
	ioc.Register(new(dataSyncAppImpl), ioc.WithComponentName("DbDataSyncTaskApp"))
}

func Init() {
	sync.OnceFunc(func() {
		repositories := &repository.Repositories{
			Instance:       persistence.GetInstanceRepo(),
			Backup:         persistence.NewDbBackupRepo(),
			BackupHistory:  persistence.NewDbBackupHistoryRepo(),
			Restore:        persistence.NewDbRestoreRepo(),
			RestoreHistory: persistence.NewDbRestoreHistoryRepo(),
			Binlog:         persistence.NewDbBinlogRepo(),
			BinlogHistory:  persistence.NewDbBinlogHistoryRepo(),
		}
		var err error
		dbApp := GetDbApp()
		scheduler, err := newDbScheduler(repositories)
		if err != nil {
			panic(fmt.Sprintf("初始化 dbScheduler 失败: %v", err))
		}
		dbBackupApp, err = newDbBackupApp(repositories, dbApp, scheduler)
		if err != nil {
			panic(fmt.Sprintf("初始化 dbBackupApp 失败: %v", err))
		}
		dbRestoreApp, err = newDbRestoreApp(repositories, dbApp, scheduler)
		if err != nil {
			panic(fmt.Sprintf("初始化 dbRestoreApp 失败: %v", err))
		}
		dbBinlogApp, err = newDbBinlogApp(repositories, dbApp, scheduler)
		if err != nil {
			panic(fmt.Sprintf("初始化 dbBinlogApp 失败: %v", err))
		}

		GetDataSyncTaskApp().InitCronJob()
	})()
}

func GetInstanceApp() Instance {
	return ioc.Get[Instance]("DbInstance")
}

func GetDbApp() Db {
	return ioc.Get[Db]("DbApp")
}

func GetDbSqlApp() DbSql {
	return ioc.Get[DbSql]("DbSqlApp")
}

func GetDbSqlExecApp() DbSqlExec {
	return ioc.Get[DbSqlExec]("DbSqlExecApp")
}

func GetDbBackupApp() *DbBackupApp {
	return dbBackupApp
}

func GetDbRestoreApp() *DbRestoreApp {
	return dbRestoreApp
}

func GetDbBinlogApp() *DbBinlogApp {
	return dbBinlogApp
}

func GetDataSyncTaskApp() DataSyncTask {
	return ioc.Get[DataSyncTask]("DbDataSyncTaskApp")
}
