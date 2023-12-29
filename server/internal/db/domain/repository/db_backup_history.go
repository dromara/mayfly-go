package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbBackupHistory interface {
	base.Repo[*entity.DbBackupHistory]

	// GetDbBackupHistories 分页获取数据备份历史
	GetHistories(condition *entity.DbBackupHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
	GetLatestHistory(instanceId uint64, dbName string, bi *entity.BinlogInfo) (*entity.DbBackupHistory, error)
	GetEarliestHistory(instanceId uint64) (*entity.DbBackupHistory, error)
}
