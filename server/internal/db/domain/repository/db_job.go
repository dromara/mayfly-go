package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

type DbJobBase[T entity.DbJob] interface {
	base.Repo[T]

	// UpdateLastStatus 更新任务执行状态
	UpdateLastStatus(ctx context.Context, job entity.DbJob) error
}

type DbJob[T entity.DbJob] interface {
	DbJobBase[T]

	// AddJob 添加数据库任务
	AddJob(ctx context.Context, jobs any) error
	UpdateEnabled(ctx context.Context, jobId uint64, enabled bool) error
}
