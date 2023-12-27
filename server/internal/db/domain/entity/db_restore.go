package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

var _ DbTask = (*DbRestore)(nil)

// DbRestore 数据库恢复任务
type DbRestore struct {
	model.Model

	DbName              string        `gorm:"column(db_name)" json:"dbName"`                            // 数据库名
	StartTime           time.Time     `gorm:"column(start_time)" json:"startTime"`                      // 开始时间
	Interval            time.Duration `gorm:"column(interval)" json:"interval"`                         // 间隔时间: 为零表示单次执行，为正表示反复执行
	Enabled             bool          `gorm:"column(enabled)" json:"enabled"`                           // 是否启用
	Finished            bool          `gorm:"column(finished)" json:"finished"`                         // 是否完成
	Repeated            bool          `gorm:"column(repeated)" json:"repeated"`                         // 是否重复执行
	LastStatus          TaskStatus    `gorm:"column(last_status)" json:"lastStatus"`                    // 最近一次执行状态
	LastResult          string        `gorm:"column(last_result)" json:"lastResult"`                    // 最近一次执行结果
	LastTime            time.Time     `gorm:"column(last_time)" json:"lastTime"`                        // 最近一次执行时间
	PointInTime         time.Time     `gorm:"column(point_in_time)" json:"pointInTime"`                 // 指定数据库恢复的时间点
	DbBackupId          uint64        `gorm:"column(db_backup_id)" json:"dbBackupId"`                   // 用于恢复的数据库备份任务ID
	DbBackupHistoryId   uint64        `gorm:"column(db_backup_history_id)" json:"dbBackupHistoryId"`    // 用于恢复的数据库备份历史ID
	DbBackupHistoryName string        `gorm:"column(db_backup_history_name) json:"dbBackupHistoryName"` // 数据库备份历史名称
	DbInstanceId        uint64        `gorm:"column(db_instance_id)" json:"dbInstanceId"`
	Deadline            time.Time     `gorm:"-" json:"-"`
}

func (d *DbRestore) TableName() string {
	return "t_db_restore"
}

func (d *DbRestore) GetId() uint64 {
	if d == nil {
		return 0
	}
	return d.Id
}

func (d *DbRestore) GetDeadline() time.Time {
	return d.Deadline
}

func (d *DbRestore) Schedule() bool {
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

func (d *DbRestore) IsFinished() bool {
	return !d.Repeated && d.LastStatus == TaskSuccess
}

func (d *DbRestore) Update(task DbTask) bool {
	switch backup := task.(type) {
	case *DbRestore:
		d.StartTime = backup.StartTime
		d.Interval = backup.Interval
		return true
	}
	return false
}
