package vo

import (
	"encoding/json"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/utils/timex"
	"time"
)

// DbBackup 数据库备份任务
type DbBackup struct {
	Id           uint64             `json:"id"`
	DbName       string             `json:"dbName"`               // 数据库名
	CreateTime   time.Time          `json:"createTime"`           // 创建时间
	StartTime    time.Time          `json:"startTime"`            // 开始时间
	Interval     time.Duration      `json:"-"`                    // 间隔时间
	IntervalDay  uint64             `json:"intervalDay" gorm:"-"` // 间隔天数
	MaxSaveDays  int                `json:"maxSaveDays"`          // 数据库备份历史保留天数，过期将自动删除
	Enabled      bool               `json:"enabled"`              // 是否启用
	EnabledDesc  string             `json:"enabledDesc"`          // 启用状态描述
	LastTime     timex.NullTime     `json:"lastTime"`             // 最近一次执行时间
	LastStatus   entity.DbJobStatus `json:"lastStatus"`           // 最近一次执行状态
	LastResult   string             `json:"lastResult"`           // 最近一次执行结果
	DbInstanceId uint64             `json:"dbInstanceId"`         // 数据库实例ID
	Name         string             `json:"name"`                 // 备份任务名称
}

func (backup *DbBackup) MarshalJSON() ([]byte, error) {
	type dbBackup DbBackup
	backup.IntervalDay = uint64(backup.Interval / time.Hour / 24)
	if len(backup.EnabledDesc) == 0 {
		if backup.Enabled {
			backup.EnabledDesc = "已启用"
		} else {
			backup.EnabledDesc = "已禁用"
		}
	}
	return json.Marshal((*dbBackup)(backup))
}

// DbBackupHistory 数据库备份历史
type DbBackupHistory struct {
	Id             uint64             `json:"id"`
	DbBackupId     uint64             `json:"dbBackupId"`
	CreateTime     time.Time          `json:"createTime"`
	DbName         string             `json:"dbName"` // 数据库名称
	Name           string             `json:"name"`   // 备份历史名称
	BinlogFileName string             `json:"binlogFileName"`
	LastTime       timex.NullTime     `json:"lastTime" gorm:"-"`   // 最近一次恢复时间
	LastStatus     entity.DbJobStatus `json:"lastStatus" gorm:"-"` // 最近一次恢复状态
	LastResult     string             `json:"lastResult" gorm:"-"` // 最近一次恢复结果
}
