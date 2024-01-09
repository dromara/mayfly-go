package initialize

import (
	dbApp "mayfly-go/internal/db/application"
)

// 终止服务后的一些操作
func Terminate() {
	closeDbTasks()
}

func closeDbTasks() {
	restoreApp := dbApp.GetDbRestoreApp()
	if restoreApp != nil {
		restoreApp.Close()
	}
	binlogApp := dbApp.GetDbBinlogApp()
	if binlogApp != nil {
		binlogApp.Close()
	}
	backupApp := dbApp.GetDbBackupApp()
	if backupApp != nil {
		backupApp.Close()
	}
}
