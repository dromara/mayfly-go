package entity

import (
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/timex"
)

// DbBinlog 数据库备份任务
type DbBinlog struct {
	model.Model

	LastStatus   TaskStatus     // 最近一次执行状态
	LastResult   string         // 最近一次执行结果
	LastTime     timex.NullTime // 最近一次执行时间
	DbInstanceId uint64         `json:"dbInstanceId"` // 数据库实例ID
}

func NewDbBinlog(history *DbBackupHistory) *DbBinlog {
	binlogTask := &DbBinlog{}
	binlogTask.Id = history.DbInstanceId
	binlogTask.DbInstanceId = history.DbInstanceId
	return binlogTask
}
