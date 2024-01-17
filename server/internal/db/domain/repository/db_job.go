package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
)

type DbJobBase interface {
	// GetById 根据实体id查询
	GetById(e entity.DbJob, id uint64, cols ...string) error

	// UpdateById 根据实体id更新实体信息
	UpdateById(ctx context.Context, e entity.DbJob, columns ...string) error

	// DeleteById 根据实体主键删除实体
	DeleteById(ctx context.Context, id uint64) error

	// UpdateLastStatus 更新任务执行状态
	UpdateLastStatus(ctx context.Context, job entity.DbJob) error
}

type DbJob interface {
	DbJobBase

	// AddJob 添加数据库任务
	AddJob(ctx context.Context, jobs any) error
	UpdateEnabled(ctx context.Context, jobId uint64, enabled bool) error
}
