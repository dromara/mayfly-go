package form

import (
	"encoding/json"
	"time"
)

// DbBackupForm 数据库备份表单
type DbBackupForm struct {
	Id          uint64        `json:"id"`
	DbNames     string        `binding:"required" json:"dbNames"`   // 数据库名: 多个数据库名称用空格分隔开
	Name        string        `json:"name"`                         // 备份任务名称
	StartTime   time.Time     `binding:"required" json:"startTime"` // 开始时间: 2023-11-08 02:00:00
	Interval    time.Duration `json:"-"`                            // 间隔时间: 为零表示单次执行，为正表示反复执行
	IntervalDay uint64        `json:"intervalDay"`                  // 间隔天数: 为零表示单次执行，为正表示反复执行
	Repeated    bool          `json:"repeated"`                     // 是否重复执行
	MaxSaveDays int           `json:"maxSaveDays"`                  // 数据库备份历史保留天数，过期将自动删除
}

func (restore *DbBackupForm) UnmarshalJSON(data []byte) error {
	type dbBackupForm DbBackupForm
	if err := json.Unmarshal(data, (*dbBackupForm)(restore)); err != nil {
		return err
	}
	restore.Interval = time.Duration(restore.IntervalDay) * time.Hour * 24
	return nil
}
