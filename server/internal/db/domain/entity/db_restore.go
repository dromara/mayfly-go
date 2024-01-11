package entity

import (
	"context"
	"mayfly-go/pkg/runner"
	"mayfly-go/pkg/utils/timex"
)

var _ DbJob = (*DbRestore)(nil)

// DbRestore 数据库恢复任务
type DbRestore struct {
	*DbJobBaseImpl

	PointInTime         timex.NullTime `json:"pointInTime"`         // 指定数据库恢复的时间点
	DbBackupId          uint64         `json:"dbBackupId"`          // 用于恢复的数据库恢复任务ID
	DbBackupHistoryId   uint64         `json:"dbBackupHistoryId"`   // 用于恢复的数据库恢复历史ID
	DbBackupHistoryName string         `json:"dbBackupHistoryName"` // 数据库恢复历史名称
}

func (d *DbRestore) SetRun(fn func(ctx context.Context, job DbJob)) {
	d.run = func(ctx context.Context) {
		fn(ctx, d)
	}
}

func (d *DbRestore) SetRunnable(fn func(job DbJob, next runner.NextFunc) bool) {
	d.runnable = func(next runner.NextFunc) bool {
		return fn(d, next)
	}
}
