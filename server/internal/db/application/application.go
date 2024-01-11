package application

import (
	"fmt"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/infrastructure/persistence"
	tagapp "mayfly-go/internal/tag/application"
	"sync"
)

var (
	instanceApp  Instance
	dbApp        Db
	dbSqlExecApp DbSqlExec
	dbSqlApp     DbSql
	dbBackupApp  *DbBackupApp
	dbRestoreApp *DbRestoreApp
	dbBinlogApp  *DbBinlogApp
	dataSyncApp  DataSyncTask
)

//var repositories *repository.Repositories
//var scheduler *dbScheduler[*entity.DbBackup]
//var scheduler1 *dbScheduler[*entity.DbRestore]

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
		instanceRepo := persistence.GetInstanceRepo()
		instanceApp = newInstanceApp(instanceRepo)
		dbApp = newDbApp(persistence.GetDbRepo(), persistence.GetDbSqlRepo(), instanceApp, tagapp.GetTagTreeApp())
		dbSqlExecApp = newDbSqlExecApp(persistence.GetDbSqlExecRepo())
		dbSqlApp = newDbSqlApp(persistence.GetDbSqlRepo())
		dataSyncApp = newDataSyncApp(persistence.GetDataSyncTaskRepo(), persistence.GetDataSyncLogRepo())

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
		dbBinlogApp, err = newDbBinlogApp(repositories, dbApp)
		if err != nil {
			panic(fmt.Sprintf("初始化 dbBinlogApp 失败: %v", err))
		}

		dataSyncApp.InitCronJob()
	})()
}

func GetInstanceApp() Instance {
	return instanceApp
}

func GetDbApp() Db {
	return dbApp
}

func GetDbSqlApp() DbSql {
	return dbSqlApp
}

func GetDbSqlExecApp() DbSqlExec {
	return dbSqlExecApp
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
	return dataSyncApp
}
