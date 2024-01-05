package vo

import (
	"encoding/json"
	"mayfly-go/pkg/utils/timex"
	"time"
)

// DbBackup 数据库备份任务
type DbBackup struct {
	Id           uint64         `json:"id"`
	DbName       string         `json:"dbName"`               // 数据库名
	CreateTime   time.Time      `json:"createTime"`           // 创建时间
	StartTime    time.Time      `json:"startTime"`            // 开始时间
	Interval     time.Duration  `json:"-"`                    // 间隔时间
	IntervalDay  uint64         `json:"intervalDay" gorm:"-"` // 间隔天数
	Enabled      bool           `json:"enabled"`              // 是否启用
	LastTime     timex.NullTime `json:"lastTime"`             // 最近一次执行时间
	LastStatus   string         `json:"lastStatus"`           // 最近一次执行状态
	LastResult   string         `json:"lastResult"`           // 最近一次执行结果
	DbInstanceId uint64         `json:"dbInstanceId"`         // 数据库实例ID
	Name         string         `json:"name"`                 // 备份任务名称
}

func (backup *DbBackup) MarshalJSON() ([]byte, error) {
	type dbBackup DbBackup
	backup.IntervalDay = uint64(backup.Interval / time.Hour / 24)
	return json.Marshal((*dbBackup)(backup))
}

// DbBackupHistory 数据库备份历史
type DbBackupHistory struct {
	Id         uint64    `json:"id"`
	DbBackupId uint64    `json:"dbBackupId"`
	CreateTime time.Time `json:"createTime"`
	DbName     string    `json:"dbName"` // 数据库名称
	Name       string    `json:"name"`   // 备份历史名称
}
