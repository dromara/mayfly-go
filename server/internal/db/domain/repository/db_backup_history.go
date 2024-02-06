package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbBackupHistory interface {
	base.Repo[*entity.DbBackupHistory]

	// GetPageList 分页获取数据备份历史
	GetPageList(condition *entity.DbBackupHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	GetLatestHistoryForBinlog(instanceId uint64, dbName string, bi *entity.BinlogInfo) (*entity.DbBackupHistory, error)

	GetEarliestHistoryForBinlog(instanceId uint64) (*entity.DbBackupHistory, bool, error)

	GetHistories(backupHistoryIds []uint64, toEntity any) error

	UpdateDeleting(deleting bool, backupHistoryId ...uint64) (bool, error)
	UpdateRestoring(restoring bool, backupHistoryId ...uint64) (bool, error)
	ZeroBinlogInfo(backupHistoryId uint64) error
}
