package form

import (
	"encoding/json"
	"mayfly-go/pkg/utils/timex"
	"time"
)

// DbRestoreForm 数据库备份表单
type DbRestoreForm struct {
	Id                  uint64         `json:"id"`
	DbName              string         `binding:"required" json:"dbName"`    // 数据库名
	StartTime           time.Time      `binding:"required" json:"startTime"` // 开始时间: 2023-11-08 02:00:00
	PointInTime         timex.NullTime `json:"pointInTime"`                  // 指定时间
	DbBackupId          uint64         `json:"dbBackupId"`                   // 数据库备份任务ID
	DbBackupHistoryId   uint64         `json:"dbBackupHistoryId"`            // 数据库备份历史ID
	DbBackupHistoryName string         `json:"dbBackupHistoryName"`          // 数据库备份历史名称
	Interval            time.Duration  `json:"-"`                            // 间隔时间: 为零表示单次执行，为正表示反复执行
	IntervalDay         uint64         `json:"intervalDay"`                  // 间隔天数: 为零表示单次执行，为正表示反复执行
	Repeated            bool           `json:"repeated"`                     // 是否重复执行
}

func (restore *DbRestoreForm) UnmarshalJSON(data []byte) error {
	type dbRestoreFormPtr *DbRestoreForm
	if err := json.Unmarshal(data, dbRestoreFormPtr(restore)); err != nil {
		return err
	}
	restore.Interval = time.Duration(restore.IntervalDay) * time.Hour * 24
	return nil
}
