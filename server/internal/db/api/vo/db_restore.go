package vo

import (
	"encoding/json"
	"mayfly-go/pkg/utils/timex"
	"time"
)

// DbRestore 数据库备份任务
type DbRestore struct {
	Id                  uint64         `json:"id"`
	DbName              string         `json:"dbName"`               // 数据库名
	StartTime           time.Time      `json:"startTime"`            // 开始时间
	Interval            time.Duration  `json:"-"`                    // 间隔时间
	IntervalDay         uint64         `json:"intervalDay" gorm:"-"` // 间隔天数
	Enabled             bool           `json:"enabled"`              // 是否启用
	EnabledDesc         string         `json:"enabledDesc"`          // 启用状态描述
	LastTime            timex.NullTime `json:"lastTime"`             // 最近一次执行时间
	LastStatus          string         `json:"lastStatus"`           // 最近一次执行状态
	LastResult          string         `json:"lastResult"`           // 最近一次执行结果
	PointInTime         timex.NullTime `json:"pointInTime"`          // 指定数据库恢复的时间点
	DbBackupId          uint64         `json:"dbBackupId"`           // 数据库备份任务ID
	DbBackupHistoryId   uint64         `json:"dbBackupHistoryId"`    // 数据库备份历史ID
	DbBackupHistoryName string         `json:"dbBackupHistoryName"`  // 数据库备份历史名称
	DbInstanceId        uint64         `json:"dbInstanceId"`         // 数据库实例ID
}

func (restore *DbRestore) MarshalJSON() ([]byte, error) {
	type dbBackup DbRestore
	restore.IntervalDay = uint64(restore.Interval / time.Hour / 24)
	if len(restore.EnabledDesc) == 0 {
		if restore.Enabled {
			restore.EnabledDesc = "已启用"
		} else {
			restore.EnabledDesc = "已禁用"
		}
	}
	return json.Marshal((*dbBackup)(restore))
}

// DbRestoreHistory 数据库备份历史
type DbRestoreHistory struct {
	Id          uint64 `json:"id"`
	DbRestoreId uint64 `json:"dbRestoreId"`
}
