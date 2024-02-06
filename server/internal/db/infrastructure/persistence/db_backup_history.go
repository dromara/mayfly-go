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

func (repo *dbBackupHistoryRepoImpl) GetPageList(condition *entity.DbBackupHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(repo.GetModel()).
		Eq("id", condition.Id).
		Eq0("db_instance_id", condition.DbInstanceId).
		In0("db_name", condition.InDbNames).
		Eq("db_backup_id", condition.DbBackupId).
		Eq("db_name", condition.DbName)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (repo *dbBackupHistoryRepoImpl) GetHistories(backupHistoryIds []uint64, toEntity any) error {
	return global.Db.Model(repo.GetModel()).
		Where("id in ?", backupHistoryIds).
		Where("deleting = false").
		Scopes(gormx.UndeleteScope).
		Find(toEntity).
		Error
}

func (repo *dbBackupHistoryRepoImpl) GetLatestHistoryForBinlog(instanceId uint64, dbName string, bi *entity.BinlogInfo) (*entity.DbBackupHistory, error) {
	history := &entity.DbBackupHistory{}
	db := global.Db
	err := db.Model(repo.GetModel()).
		Where("db_instance_id = ?", instanceId).
		Where("db_name = ?", dbName).
		Where(db.Where("binlog_sequence < ?", bi.Sequence).
			Or(db.Where("binlog_sequence = ?", bi.Sequence).
				Where("binlog_position <= ?", bi.Position))).
		Where("binlog_sequence > 0").
		Where("deleting = false").
		Scopes(gormx.UndeleteScope).
		Order("binlog_sequence desc, binlog_position desc").
		First(history).Error
	if err != nil {
		return nil, err
	}
	return history, err
}

func (repo *dbBackupHistoryRepoImpl) GetEarliestHistoryForBinlog(instanceId uint64) (*entity.DbBackupHistory, bool, error) {
	history := &entity.DbBackupHistory{}
	db := global.Db.Model(repo.GetModel())
	err := db.Where("db_instance_id = ?", instanceId).
		Where("binlog_sequence > 0").
		Where("deleting = false").
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

func (repo *dbBackupHistoryRepoImpl) UpdateDeleting(deleting bool, backupHistoryId ...uint64) (bool, error) {
	db := global.Db.Model(repo.GetModel()).
		Where("id in ?", backupHistoryId).
		Where("restoring = false").
		Scopes(gormx.UndeleteScope).
		Update("deleting", deleting)
	if db.Error != nil {
		return false, db.Error
	}
	if db.RowsAffected != int64(len(backupHistoryId)) {
		return false, nil
	}
	return true, nil
}

func (repo *dbBackupHistoryRepoImpl) UpdateRestoring(restoring bool, backupHistoryId ...uint64) (bool, error) {
	db := global.Db.Model(repo.GetModel()).
		Where("id in ?", backupHistoryId).
		Where("deleting = false").
		Scopes(gormx.UndeleteScope).
		Update("restoring", restoring)
	if db.Error != nil {
		return false, db.Error
	}
	if db.RowsAffected != int64(len(backupHistoryId)) {
		return false, nil
	}
	return true, nil
}

func (repo *dbBackupHistoryRepoImpl) ZeroBinlogInfo(backupHistoryId uint64) error {
	return global.Db.Model(repo.GetModel()).
		Where("id = ?", backupHistoryId).
		Where("restoring = false").
		Scopes(gormx.UndeleteScope).
		Updates(&map[string]any{
			"binlog_file_name": "",
			"binlog_sequence":  0,
			"binlog_position":  0,
		}).Error
}
