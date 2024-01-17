package entity

import (
	"mayfly-go/pkg/runner"
	"mayfly-go/pkg/utils/timex"
	"time"
)

var _ DbJob = (*DbRestore)(nil)

// DbRestore 数据库恢复任务
type DbRestore struct {
	*DbJobBaseImpl

	DbName              string         // 数据库名称
	Enabled             bool           // 是否启用
	StartTime           time.Time      // 开始时间
	Interval            time.Duration  // 间隔时间
	Repeated            bool           // 是否重复执行
	PointInTime         timex.NullTime `json:"pointInTime"`         // 指定数据库恢复的时间点
	DbBackupId          uint64         `json:"dbBackupId"`          // 用于恢复的数据库恢复任务ID
	DbBackupHistoryId   uint64         `json:"dbBackupHistoryId"`   // 用于恢复的数据库恢复历史ID
	DbBackupHistoryName string         `json:"dbBackupHistoryName"` // 数据库恢复历史名称
}

func (r *DbRestore) GetDbName() string {
	return r.DbName
}

func (r *DbRestore) Schedule() (time.Time, error) {
	var deadline time.Time
	if r.IsFinished() || !r.Enabled {
		return deadline, runner.ErrFinished
	}
	switch r.LastStatus {
	case DbJobSuccess:
		lastTime := r.LastTime.Time
		if lastTime.Before(r.StartTime) {
			lastTime = r.StartTime.Add(-r.Interval)
		}
		deadline = lastTime.Add(r.Interval - lastTime.Sub(r.StartTime)%r.Interval)
	case DbJobFailed:
		deadline = time.Now().Add(time.Minute)
	default:
		deadline = r.StartTime
	}
	return deadline, nil
}

func (r *DbRestore) IsEnabled() bool {
	return r.Enabled
}

func (r *DbRestore) SetEnabled(enabled bool) {
	r.Enabled = enabled
}

func (r *DbRestore) IsFinished() bool {
	return !r.Repeated && r.LastStatus == DbJobSuccess
}

func (r *DbRestore) Update(job runner.Job) {
	restore := job.(*DbRestore)
	r.StartTime = restore.StartTime
	r.Interval = restore.Interval
}

func (r *DbRestore) GetInterval() time.Duration {
	return r.Interval
}
