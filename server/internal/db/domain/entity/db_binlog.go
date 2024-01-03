package entity

import (
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/timex"
	"time"
)

// DbBinlog 数据库备份任务
type DbBinlog struct {
	model.Model

	LastStatus   TaskStatus     // 最近一次执行状态
	LastResult   string         // 最近一次执行结果
	LastTime     timex.NullTime // 最近一次执行时间
	DbInstanceId uint64         `json:"dbInstanceId"` // 数据库实例ID
}

func NewDbBinlog(instanceId uint64) *DbBinlog {
	binlogTask := &DbBinlog{}
	binlogTask.Id = instanceId
	binlogTask.DbInstanceId = instanceId
	return binlogTask
}

// BinlogFile is the metadata of the MySQL binlog file.
type BinlogFile struct {
	Name string
	Size int64

	// Sequence is parsed from Name and is for the sorting purpose.
	Sequence       int64
	FirstEventTime time.Time
	Downloaded     bool
}
