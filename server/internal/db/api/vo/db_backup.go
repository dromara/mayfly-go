package vo

import (
	"encoding/json"
	"time"
)

// DbBackupHistory 数据库备份任务
type DbBackup struct {
	Id           uint64        `json:"id"`
	DbName       string        `json:"dbName"`               // 数据库名
	CreateTime   time.Time     `json:"createTime"`           // 创建时间: 2023-11-08 02:00:00
	StartTime    time.Time     `json:"startTime"`            // 开始时间: 2023-11-08 02:00:00
	Interval     time.Duration `json:"-"`                    // 间隔时间: 为零表示单次执行，为正表示反复执行
	IntervalDay  uint64        `json:"intervalDay" gorm:"-"` // 间隔天数: 为零表示单次执行，为正表示反复执行
	Enabled      bool          `json:"enabled"`              // 是否启用
	LastTime     time.Time     `json:"lastTime"`             // 最近一次执行时间: 2023-11-08 02:00:00
	LastStatus   string        `json:"lastStatus"`           // 最近一次执行状态
	LastResult   string        `json:"lastResult"`           // 最近一次执行结果
	DbInstanceId uint64        `json:"dbInstanceId"`         // 数据库实例ID
	Name         string        `json:"name"`                 // 备份任务名称
}

func (restore *DbBackup) MarshalJSON() ([]byte, error) {
	type dbBackup DbBackup
	restore.IntervalDay = uint64(restore.Interval / time.Hour / 24)
	return json.Marshal((*dbBackup)(restore))
}
