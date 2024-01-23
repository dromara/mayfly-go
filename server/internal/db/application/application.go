package application

import (
	"fmt"
	"mayfly-go/internal/db/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
	"sync"
)

func InitIoc() {
	persistence.Init()

	ioc.Register(new(instanceAppImpl), ioc.WithComponentName("DbInstanceApp"))
	ioc.Register(new(dbAppImpl), ioc.WithComponentName("DbApp"))
	ioc.Register(new(dbSqlExecAppImpl), ioc.WithComponentName("DbSqlExecApp"))
	ioc.Register(new(dbSqlAppImpl), ioc.WithComponentName("DbSqlApp"))
	ioc.Register(new(dataSyncAppImpl), ioc.WithComponentName("DbDataSyncTaskApp"))

	ioc.Register(newDbScheduler(), ioc.WithComponentName("DbScheduler"))
	ioc.Register(new(DbBackupApp), ioc.WithComponentName("DbBackupApp"))
	ioc.Register(new(DbRestoreApp), ioc.WithComponentName("DbRestoreApp"))
	ioc.Register(newDbBinlogApp(), ioc.WithComponentName("DbBinlogApp"))
}

func Init() {
	sync.OnceFunc(func() {
		if err := GetDbBackupApp().Init(); err != nil {
			panic(fmt.Sprintf("初始化 dbBackupApp 失败: %v", err))
		}
		if err := GetDbRestoreApp().Init(); err != nil {
			panic(fmt.Sprintf("初始化 dbRestoreApp 失败: %v", err))
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
	return ioc.Get[*DbBackupApp]("DbBackupApp")
}

func GetDbRestoreApp() *DbRestoreApp {
	return ioc.Get[*DbRestoreApp]("DbRestoreApp")
}

func GetDbBinlogApp() *DbBinlogApp {
	return ioc.Get[*DbBinlogApp]("DbBinlogApp")
}

func GetDataSyncTaskApp() DataSyncTask {
	return ioc.Get[DataSyncTask]("DbDataSyncTaskApp")
}
