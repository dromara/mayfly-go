package persistence

import (
	"errors"
	"gorm.io/gorm"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

var _ repository.DbBackupHistory = (*dbBackupHistoryRepoImpl)(nil)

type dbBackupHistoryRepoImpl struct {
	base.RepoImpl[*entity.DbBackupHistory]
}

func NewDbBackupHistoryRepo() repository.DbBackupHistory {
	return &dbBackupHistoryRepoImpl{}
}

func (repo *dbBackupHistoryRepoImpl) GetHistories(condition *entity.DbBackupHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(new(entity.DbBackupHistory)).
		Eq("id", condition.Id).
		Eq0("db_instance_id", condition.DbInstanceId).
		In0("db_name", condition.InDbNames).
		Eq("db_backup_id", condition.DbBackupId).
		Eq("db_name", condition.DbName)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (repo *dbBackupHistoryRepoImpl) GetLatestHistory(instanceId uint64, dbName string, bi *entity.BinlogInfo) (*entity.DbBackupHistory, error) {
	history := &entity.DbBackupHistory{}
	db := global.Db
	err := db.Model(repo.GetModel()).
		Where("db_instance_id = ?", instanceId).
		Where("db_name = ?", dbName).
		Where(db.Where("binlog_sequence < ?", bi.Sequence).
			Or(db.Where("binlog_sequence = ?", bi.Sequence).
				Where("binlog_position <= ?", bi.Position))).
		Scopes(gormx.UndeleteScope).
		Order("binlog_sequence desc, binlog_position desc").
		First(history).Error
	if err != nil {
		return nil, err
	}
	return history, err
}

func (repo *dbBackupHistoryRepoImpl) GetEarliestHistory(instanceId uint64) (*entity.DbBackupHistory, bool, error) {
	history := &entity.DbBackupHistory{}
	db := global.Db.Model(repo.GetModel())
	err := db.Where("db_instance_id = ?", instanceId).
		Scopes(gormx.UndeleteScope).
		Order("binlog_sequence").
		First(history).Error
	switch {
	case err == nil:
		return history, true, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return history, false, nil
	default:
		return nil, false, err
	}
}
