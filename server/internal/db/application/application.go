package application

import (
	"fmt"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/infrastructure/persistence"
	tagapp "mayfly-go/internal/tag/application"
	"sync"
)

var (
	instanceApp         Instance
	dbApp               Db
	dbSqlExecApp        DbSqlExec
	dbSqlApp            DbSql
	dbBackupApp         *DbBackupApp
	dbBackupHistoryApp  *DbBackupHistoryApp
	dbRestoreApp        *DbRestoreApp
	dbRestoreHistoryApp *DbRestoreHistoryApp
	dbBinlogApp         *DbBinlogApp
	dataSyncApp         DataSyncTask
	dataSyncLogApp      DataSyncLog
)

var repositories *repository.Repositories

func Init() {
	sync.OnceFunc(func() {
		repositories = &repository.Repositories{
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
		dataSyncApp = newDataSyncApp(persistence.GetDataSyncTaskRepo())
		dataSyncLogApp = newDataSyncLogApp(persistence.GetDataSyncLogRepo())

		dbBackupApp, err = newDbBackupApp(repositories, dbApp)
		if err != nil {
			panic(fmt.Sprintf("初始化 dbBackupApp 失败: %v", err))
		}
		dbRestoreApp, err = newDbRestoreApp(repositories, dbApp)
		if err != nil {
			panic(fmt.Sprintf("初始化 dbRestoreApp 失败: %v", err))
		}
		dbBackupHistoryApp, err = newDbBackupHistoryApp(repositories)
		if err != nil {
			panic(fmt.Sprintf("初始化 dbBackupHistoryApp 失败: %v", err))
		}
		dbRestoreHistoryApp, err = newDbRestoreHistoryApp(repositories)
		if err != nil {
			panic(fmt.Sprintf("初始化 dbRestoreHistoryApp 失败: %v", err))
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

func GetDbBackupHistoryApp() *DbBackupHistoryApp {
	return dbBackupHistoryApp
}

func GetDbRestoreApp() *DbRestoreApp {
	return dbRestoreApp
}

func GetDbRestoreHistoryApp() *DbRestoreHistoryApp {
	return dbRestoreHistoryApp
}

func GetDbBinlogApp() *DbBinlogApp {
	return dbBinlogApp
}

func GetDataSyncTaskApp() DataSyncTask {
	return dataSyncApp
}

func GetDataSyncLogApp() DataSyncLog {
	return dataSyncLogApp
}
