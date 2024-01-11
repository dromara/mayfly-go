package repository

import (
	"context"
	"gorm.io/gorm"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/model"
)

type DbJob interface {
	// AddJob 添加数据库任务
	AddJob(ctx context.Context, jobs any) error
	// GetById 根据实体id查询
	GetById(e entity.DbJob, id uint64, cols ...string) error
	// GetPageList 分页获取数据库任务列表
	GetPageList(condition *entity.DbJobQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
	// UpdateById 根据实体id更新实体信息
	UpdateById(ctx context.Context, e entity.DbJob, columns ...string) error
	// BatchInsertWithDb 使用指定gorm db执行，主要用于事务执行
	BatchInsertWithDb(ctx context.Context, db *gorm.DB, es any) error
	// DeleteById 根据实体主键删除实体
	DeleteById(ctx context.Context, id uint64) error

	UpdateLastStatus(ctx context.Context, job entity.DbJob) error
	UpdateEnabled(ctx context.Context, jobId uint64, enabled bool) error
	ListToDo(jobs any) error
	ListRepeating(jobs any) error
}
