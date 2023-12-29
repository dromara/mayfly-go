package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

var _ DbTask = (*DbBackup)(nil)

// DbBackup 数据库备份任务
type DbBackup struct {
	model.Model

	Name         string        `gorm:"column(db_name)" json:"name"`           // 备份任务名称
	DbName       string        `gorm:"column(db_name)" json:"dbName"`         // 数据库名
	StartTime    time.Time     `gorm:"column(start_time)" json:"startTime"`   // 开始时间: 2023-11-08 02:00:00
	Interval     time.Duration `gorm:"column(interval)" json:"interval"`      // 间隔时间: 为零表示单次执行，为正表示反复执行
	Enabled      bool          `gorm:"column(enabled)" json:"enabled"`        // 是否启用
	Finished     bool          `gorm:"column(finished)" json:"finished"`      // 是否完成
	Repeated     bool          `gorm:"column(repeated)" json:"repeated"`      // 是否重复执行
	LastStatus   TaskStatus    `gorm:"column(last_status)" json:"lastStatus"` // 最近一次执行状态
	LastResult   string        `gorm:"column(last_result)" json:"lastResult"` // 最近一次执行结果
	LastTime     time.Time     `gorm:"column(last_time)" json:"lastTime"`     // 最近一次执行时间: 2023-11-08 02:00:00
	DbInstanceId uint64        `gorm:"column(db_instance_id)" json:"dbInstanceId"`
	Deadline     time.Time     `gorm:"-" json:"-"`
}

func (d *DbBackup) TableName() string {
	return "t_db_backup"
}

func (d *DbBackup) GetId() uint64 {
	if d == nil {
		return 0
	}
	return d.Id
}

func (d *DbBackup) GetDeadline() time.Time {
	return d.Deadline
}

func (d *DbBackup) Schedule() bool {
	if d.Finished || !d.Enabled {
		return false
	}
	switch d.LastStatus {
	case TaskSuccess:
		if d.Interval == 0 {
			return false
		}
		lastTime := d.LastTime
		if d.LastTime.Sub(d.StartTime) < 0 {
			lastTime = d.StartTime.Add(-d.Interval)
		}
		d.Deadline = lastTime.Add(d.Interval - d.LastTime.Sub(d.StartTime)%d.Interval)
	case TaskFailed:
		d.Deadline = time.Now().Add(time.Minute)
	default:
		d.Deadline = d.StartTime
	}
	return true
}

func (d *DbBackup) IsFinished() bool {
	return !d.Repeated && d.LastStatus == TaskSuccess
}

func (d *DbBackup) Update(task DbTask) bool {
	switch t := task.(type) {
	case *DbBackup:
		d.StartTime = t.StartTime
		d.Interval = t.Interval
		return true
	}
	return false
}
