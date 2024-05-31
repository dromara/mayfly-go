package init

import "mayfly-go/internal/db/application"

// 终止进程时的处理函数
func Terminate() {
	closeDbTasks()
}

func closeDbTasks() {
	restoreApp := application.GetDbRestoreApp()
	if restoreApp != nil {
		restoreApp.Close()
	}
	binlogApp := application.GetDbBinlogApp()
	if binlogApp != nil {
		binlogApp.Close()
	}
	backupApp := application.GetDbBackupApp()
	if backupApp != nil {
		backupApp.Close()
	}
}
