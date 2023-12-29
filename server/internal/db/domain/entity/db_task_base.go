package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type TaskStatus int

const (
	TaskDelay TaskStatus = iota
	TaskReady
	TaskReserved
	TaskSuccess
	TaskFailed
)

const LastResultSize = 256

type DbTask interface {
	GetId() uint64
	GetDeadline() time.Time
	IsFinished() bool
	Schedule() bool
	Update(task DbTask) bool
}

func NewDbBTaskBase(enabled bool, repeated bool, startTime time.Time, interval time.Duration) *DbTaskBase {
	return &DbTaskBase{
		Enabled:   enabled,
		Repeated:  repeated,
		StartTime: startTime,
		Interval:  interval,
	}
}

type DbTaskBase struct {
	model.Model

	Enabled    bool          // 是否启用
	StartTime  time.Time     // 开始时间
	Interval   time.Duration // 间隔时间
	Finished   bool          // 是否完成
	Repeated   bool          // 是否重复执行
	LastStatus TaskStatus    // 最近一次执行状态
	LastResult string        // 最近一次执行结果
	LastTime   time.Time     // 最近一次执行时间
	Deadline   time.Time     `gorm:"-" json:"-"` // 计划执行时间
}

func (d *DbTaskBase) GetId() uint64 {
	if d == nil {
		return 0
	}
	return d.Id
}

func (d *DbTaskBase) GetDeadline() time.Time {
	return d.Deadline
}

func (d *DbTaskBase) Schedule() bool {
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

func (d *DbTaskBase) IsFinished() bool {
	return !d.Repeated && d.LastStatus == TaskSuccess
}

func (d *DbTaskBase) Update(task DbTask) bool {
	t := task.(*DbTaskBase)
	d.StartTime = t.StartTime
	d.Interval = t.Interval
	return true
}
