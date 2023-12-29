package entity

import (
	"time"
)

const BinlogDownloadInterval = time.Minute * 15

var _ DbTask = (*DbBinlog)(nil)

// DbBinlog 数据库备份任务
type DbBinlog struct {
	*DbTaskBase
	DbInstanceId uint64 `json:"dbInstanceId"` // 数据库实例ID
}

func NewDbBinlog(history *DbBackupHistory) *DbBinlog {
	binlogTask := &DbBinlog{
		DbTaskBase:   NewDbBTaskBase(true, true, time.Now(), BinlogDownloadInterval),
		DbInstanceId: history.DbInstanceId,
		LastTime:     time.Now(),
	}
	binlogTask.Id = binlogTask.DbInstanceId
	return binlogTask
}
