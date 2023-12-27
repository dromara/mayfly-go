package repository

type Repositories struct {
	Instance       Instance
	Backup         DbBackup
	BackupHistory  DbBackupHistory
	Restore        DbRestore
	RestoreHistory DbRestoreHistory
	Binlog         DbBinlog
	BinlogHistory  DbBinlogHistory
}
