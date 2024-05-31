package entity

import (
	"mayfly-go/pkg/runner"
	"time"
)

var _ DbJob = (*DbBackup)(nil)

// DbBackup 数据库备份任务
type DbBackup struct {
	DbJobBaseImpl

	DbInstanceId uint64        // 数据库实例ID
	DbName       string        // 数据库名称
	Name         string        // 数据库备份名称
	Enabled      bool          // 是否启用
	EnabledDesc  string        // 启用状态描述
	StartTime    time.Time     // 开始时间
	Interval     time.Duration // 间隔时间
	MaxSaveDays  int           // 数据库备份历史保留天数，过期将自动删除
	Repeated     bool          // 是否重复执行
}

func (b *DbBackup) GetInstanceId() uint64 {
	return b.DbInstanceId
}

func (b *DbBackup) GetDbName() string {
	return b.DbName
}

func (b *DbBackup) GetJobType() DbJobType {
	return DbJobTypeBackup
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
	case DbJobRunning, DbJobFailed:
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

func (b *DbBackup) IsExpired() bool {
	return false
}

func (b *DbBackup) SetEnabled(enabled bool, desc string) {
	b.Enabled = enabled
	b.EnabledDesc = desc
}

func (b *DbBackup) Update(job runner.Job) {
	backup := job.(*DbBackup)
	b.StartTime = backup.StartTime
	b.Interval = backup.Interval
}

func (b *DbBackup) GetInterval() time.Duration {
	return b.Interval
}

func (b *DbBackup) GetKey() DbJobKey {
	return b.getKey(b.GetJobType())
}

func (b *DbBackup) SetStatus(status runner.JobStatus, err error) {
	b.setLastStatus(b.GetJobType(), status, err)
}
