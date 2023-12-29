package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

const BinlogDownloadInterval = time.Minute * 15

var _ DbTask = (*DbBinlog)(nil)

// DbBinlog 数据库备份任务
type DbBinlog struct {
	model.Model

	StartTime    time.Time     `gorm:"column(start_time)" json:"startTime"`   // 开始时间: 2023-11-08 02:00:00
	Interval     time.Duration `gorm:"column(interval)" json:"interval"`      // 间隔时间: 为零表示单次执行，为正表示反复执行
	Enabled      bool          `gorm:"column(enabled)" json:"enabled"`        // 是否启用
	LastStatus   TaskStatus    `gorm:"column(last_status)" json:"lastStatus"` // 最近一次执行状态
	LastResult   string        `gorm:"column(last_result)" json:"lastResult"` // 最近一次执行结果
	LastTime     time.Time     `gorm:"column(last_time)" json:"lastTime"`     // 最近一次执行时间: 2023-11-08 02:00:00
	DbInstanceId uint64        `gorm:"column(db_instance_id)" json:"dbInstanceId"`
	Deadline     time.Time     `gorm:"-" json:"-"`
}

func NewDbBinlog(history *DbBackupHistory) *DbBinlog {
	binlogTask := &DbBinlog{
		StartTime:    time.Now(),
		Enabled:      true,
		Interval:     BinlogDownloadInterval,
		DbInstanceId: history.DbInstanceId,
	}
	binlogTask.Id = binlogTask.DbInstanceId
	return binlogTask
}

func (d *DbBinlog) TableName() string {
	return "t_db_binlog"
}

func (d *DbBinlog) GetId() uint64 {
	if d == nil {
		return 0
	}
	return d.Id
}

func (d *DbBinlog) GetDeadline() time.Time {
	return d.Deadline
}

func (d *DbBinlog) Schedule() bool {
	if !d.Enabled {
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

func (d *DbBinlog) IsFinished() bool {
	return false
}

func (d *DbBinlog) Update(task DbTask) bool {
	switch t := task.(type) {
	case *DbBinlog:
		d.StartTime = t.StartTime
		d.Interval = t.Interval
		return true
	}
	return false
}
