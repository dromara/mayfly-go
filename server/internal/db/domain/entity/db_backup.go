package entity

import (
	"context"
	"mayfly-go/pkg/runner"
)

var _ DbJob = (*DbBackup)(nil)

// DbBackup 数据库备份任务
type DbBackup struct {
	*DbJobBaseImpl

	Name string `json:"Name"` // 数据库备份名称
}

func (d *DbBackup) SetRun(fn func(ctx context.Context, job DbJob)) {
	d.run = func(ctx context.Context) {
		fn(ctx, d)
	}
}

func (d *DbBackup) SetRunnable(fn func(job DbJob, next runner.NextFunc) bool) {
	d.runnable = func(next runner.NextFunc) bool {
		return fn(d, next)
	}
}
