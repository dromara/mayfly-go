package entity

import (
	"mayfly-go/pkg/utils/timex"
)

var _ DbTask = (*DbRestore)(nil)

// DbRestore 数据库恢复任务
type DbRestore struct {
	*DbTaskBase

	DbName              string         `json:"dbName"`              // 数据库名
	PointInTime         timex.NullTime `json:"pointInTime"`         // 指定数据库恢复的时间点
	DbBackupId          uint64         `json:"dbBackupId"`          // 用于恢复的数据库备份任务ID
	DbBackupHistoryId   uint64         `json:"dbBackupHistoryId"`   // 用于恢复的数据库备份历史ID
	DbBackupHistoryName string         `json:"dbBackupHistoryName"` // 数据库备份历史名称
	DbInstanceId        uint64         `json:"dbInstanceId"`        // 数据库实例ID
}
