package persistence

import (
	"context"
	"fmt"
	"gorm.io/gorm/clause"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
)

var _ repository.DbBinlog = (*dbBinlogRepoImpl)(nil)

type dbBinlogRepoImpl struct {
	base.RepoImpl[*entity.DbBinlog]
}

func NewDbBinlogRepo() repository.DbBinlog {
	return &dbBinlogRepoImpl{}
}

func (d *dbBinlogRepoImpl) AddTaskIfNotExists(_ context.Context, task *entity.DbBinlog) error {
	if err := global.Db.Clauses(clause.OnConflict{DoNothing: true}).Create(task).Error; err != nil {
		return fmt.Errorf("启动 binlog 下载失败: %w", err)
	}
	return nil
}
