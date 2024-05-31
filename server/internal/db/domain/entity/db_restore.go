package entity

import (
	"mayfly-go/pkg/runner"
	"mayfly-go/pkg/utils/timex"
	"time"
)

var _ DbJob = (*DbRestore)(nil)

// DbRestore 数据库恢复任务
type DbRestore struct {
	DbJobBaseImpl

	DbInstanceId        uint64         // 数据库实例ID
	DbName              string         // 数据库名称
	Enabled             bool           // 是否启用
	EnabledDesc         string         // 启用状态描述
	StartTime           time.Time      // 开始时间
	Interval            time.Duration  // 间隔时间
	Repeated            bool           // 是否重复执行
	PointInTime         timex.NullTime `json:"pointInTime"`         // 指定数据库恢复的时间点
	DbBackupId          uint64         `json:"dbBackupId"`          // 用于恢复的数据库恢复任务ID
	DbBackupHistoryId   uint64         `json:"dbBackupHistoryId"`   // 用于恢复的数据库恢复历史ID
	DbBackupHistoryName string         `json:"dbBackupHistoryName"` // 数据库恢复历史名称
}

func (r *DbRestore) GetInstanceId() uint64 {
	return r.DbInstanceId
}

func (r *DbRestore) GetDbName() string {
	return r.DbName
}

func (r *DbRestore) Schedule() (time.Time, error) {
	if !r.Enabled {
		return time.Time{}, runner.ErrJobDisabled
	}
	switch r.LastStatus {
	case DbJobSuccess, DbJobFailed:
		return time.Time{}, runner.ErrJobFinished
	default:
		if time.Now().Sub(r.StartTime) > time.Hour {
			return time.Time{}, runner.ErrJobExpired
		}
		return r.StartTime, nil
	}
}

func (r *DbRestore) IsEnabled() bool {
	return r.Enabled
}

func (r *DbRestore) SetEnabled(enabled bool, desc string) {
	r.Enabled = enabled
	r.EnabledDesc = desc
}

func (r *DbRestore) IsExpired() bool {
	return !r.Repeated && time.Now().After(r.StartTime.Add(time.Hour))
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

func (r *DbRestore) GetJobType() DbJobType {
	return DbJobTypeRestore
}

func (r *DbRestore) GetKey() DbJobKey {
	return r.getKey(r.GetJobType())
}

func (r *DbRestore) SetStatus(status DbJobStatus, err error) {
	r.setLastStatus(r.GetJobType(), status, err)
}
