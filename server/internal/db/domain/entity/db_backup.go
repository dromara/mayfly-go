package entity

import (
	"mayfly-go/pkg/runner"
	"time"
)

var _ DbJob = (*DbBackup)(nil)

// DbBackup 数据库备份任务
type DbBackup struct {
	*DbJobBaseImpl

	Enabled   bool          // 是否启用
	StartTime time.Time     // 开始时间
	Interval  time.Duration // 间隔时间
	Repeated  bool          // 是否重复执行
	DbName    string        // 数据库名称
	Name      string        // 数据库备份名称
}

func (b *DbBackup) GetDbName() string {
	return b.DbName
}

func (b *DbBackup) Schedule() (time.Time, error) {
	if b.IsFinished() {
		return time.Time{}, runner.ErrJobFinished
	}
	if !b.Enabled {
		return time.Time{}, runner.ErrJobDisabled
	}
	switch b.LastStatus {
	case DbJobSuccess:
		lastTime := b.LastTime.Time
		if lastTime.Before(b.StartTime) {
			lastTime = b.StartTime.Add(-b.Interval)
		}
		return lastTime.Add(b.Interval - lastTime.Sub(b.StartTime)%b.Interval), nil
	case DbJobFailed:
		return time.Now().Add(time.Minute), nil
	default:
		return b.StartTime, nil
	}
}

func (b *DbBackup) IsFinished() bool {
	return !b.Repeated && b.LastStatus == DbJobSuccess
}

func (b *DbBackup) IsEnabled() bool {
	return b.Enabled
}

func (b *DbBackup) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *DbBackup) Update(job runner.Job) {
	backup := job.(*DbBackup)
	b.StartTime = backup.StartTime
	b.Interval = backup.Interval
}

func (b *DbBackup) GetInterval() time.Duration {
	return b.Interval
}
