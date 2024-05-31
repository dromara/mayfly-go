package persistence

import (
	"context"
	"fmt"
	"gorm.io/gorm/clause"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/global"
)

var _ repository.DbBinlog = (*dbBinlogRepoImpl)(nil)

type dbBinlogRepoImpl struct {
	dbJobBaseImpl[*entity.DbBinlog]
}

func NewDbBinlogRepo() repository.DbBinlog {
	return &dbBinlogRepoImpl{}
}

func (d *dbBinlogRepoImpl) AddJobIfNotExists(_ context.Context, job *entity.DbBinlog) error {
	// todo: 如果存在已删除记录，如何处理？
	if err := global.Db.Clauses(clause.OnConflict{DoNothing: true}).Create(job).Error; err != nil {
		return fmt.Errorf("启动 binlog 下载失败: %w", err)
	}
	return nil
}

// AddJob 添加数据库任务
func (d *dbBinlogRepoImpl) AddJob(ctx context.Context, jobs any) error {
	panic("not implement, use AddJobIfNotExists")
}

func (d *dbBinlogRepoImpl) UpdateEnabled(_ context.Context, jobId uint64, enabled bool) error {
	panic("not implement")
}
