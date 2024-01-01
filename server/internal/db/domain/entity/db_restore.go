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
	DbBackupId          uint64         `json:"dbBackupId"`          // 用于恢复的数据库恢复任务ID
	DbBackupHistoryId   uint64         `json:"dbBackupHistoryId"`   // 用于恢复的数据库恢复历史ID
	DbBackupHistoryName string         `json:"dbBackupHistoryName"` // 数据库恢复历史名称
	DbInstanceId        uint64         `json:"dbInstanceId"`        // 数据库实例ID
}

func (*DbRestore) TaskResult(status TaskStatus) string {
	var result string
	switch status {
	case TaskDelay:
		result = "等待恢复数据库"
	case TaskReady:
		result = "准备恢复数据库"
	case TaskReserved:
		result = "数据库恢复中"
	case TaskSuccess:
		result = "数据库恢复成功"
	case TaskFailed:
		result = "数据库恢复失败"
	}
	return result
}
