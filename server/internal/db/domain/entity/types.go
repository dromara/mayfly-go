package entity

import "time"

type TaskStatus int

const (
	TaskDelay TaskStatus = iota
	TaskReady
	TaskReserved
	TaskSuccess
	TaskFailed
)

type DbTask interface {
	GetId() uint64
	GetDeadline() time.Time
	IsFinished() bool
	Schedule() bool
	Update(task DbTask) bool
}
