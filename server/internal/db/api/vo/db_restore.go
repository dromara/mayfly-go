package vo

import (
	"encoding/json"
	"time"
)

// DbRestore 数据库备份任务
type DbRestore struct {
	Id                  uint64        `json:"id"`
	DbName              string        `json:"dbName"`               // 数据库名
	StartTime           time.Time     `json:"startTime"`            // 开始时间: 2023-11-08 02:00:00
	Interval            time.Duration `json:"-"`                    // 间隔时间: 为零表示单次执行，为正表示反复执行
	IntervalDay         uint64        `json:"intervalDay" gorm:"-"` // 间隔天数: 为零表示单次执行，为正表示反复执行
	Enabled             bool          `json:"enabled"`              // 是否启用
	LastTime            time.Time     `json:"lastTime"`             // 最近一次执行时间: 2023-11-08 02:00:00
	LastStatus          string        `json:"lastStatus"`           // 最近一次执行状态
	LastResult          string        `json:"lastResult"`           // 最近一次执行结果
	PointInTime         time.Time     `json:"pointInTime"`          // 指定数据库恢复的时间点
	DbBackupId          uint64        `json:"dbBackupId"`           // 数据库备份任务ID
	DbBackupHistoryId   uint64        `json:"dbBackupHistoryId"`    // 数据库备份历史ID
	DbBackupHistoryName string        `json:"dbBackupHistoryName"`  // 数据库备份历史名称
	DbInstanceId        uint64        `json:"dbInstanceId"`         // 数据库实例ID
}

func (restore *DbRestore) MarshalJSON() ([]byte, error) {
	type dbBackup DbRestore
	restore.IntervalDay = uint64(restore.Interval / time.Hour / 24)
	return json.Marshal((*dbBackup)(restore))
}
