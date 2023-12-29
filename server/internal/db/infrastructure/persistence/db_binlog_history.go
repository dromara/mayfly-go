package persistence

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"time"
)

var _ repository.DbBinlogHistory = (*dbBinlogHistoryRepoImpl)(nil)

type dbBinlogHistoryRepoImpl struct {
	base.RepoImpl[*entity.DbBinlogHistory]
}

func NewDbBinlogHistoryRepo() repository.DbBinlogHistory {
	return &dbBinlogHistoryRepoImpl{}
}

func (repo *dbBinlogHistoryRepoImpl) GetHistoryByTime(instanceId uint64, targetTime time.Time) (*entity.DbBinlogHistory, error) {
	gdb := gormx.NewQuery(repo.GetModel()).
		Eq("db_instance_id", instanceId).
		Le("first_event_time", targetTime).
		Undeleted().
		OrderByDesc("first_event_time").
		GenGdb()
	history := &entity.DbBinlogHistory{}
	err := gdb.First(history).Error
	if err != nil {
		return nil, err
	}
	return history, err
}

func (repo *dbBinlogHistoryRepoImpl) GetHistories(instanceId uint64, start, target *entity.BinlogInfo) ([]*entity.DbBinlogHistory, error) {
	gdb := gormx.NewQuery(repo.GetModel()).
		Eq("db_instance_id", instanceId).
		Ge("sequence", start.Sequence).
		Le("sequence", target.Sequence).
		Undeleted().
		OrderByAsc("sequence").
		GenGdb()
	var histories []*entity.DbBinlogHistory
	err := gdb.Find(&histories).Error
	if err != nil {
		return nil, err
	}
	if len(histories) == 0 {
		return nil, errors.New("未找到满足条件的 binlog 文件")
	}
	return histories, err
}

func (repo *dbBinlogHistoryRepoImpl) GetLatestHistory(instanceId uint64) (*entity.DbBinlogHistory, bool, error) {
	gdb := gormx.NewQuery(repo.GetModel()).
		Eq("db_instance_id", instanceId).
		Undeleted().
		OrderByDesc("sequence").
		GenGdb()
	history := &entity.DbBinlogHistory{}

	switch err := gdb.First(history).Error; {
	case err == nil:
		return history, true, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return history, false, nil
	default:
		return nil, false, err
	}
}

func (repo *dbBinlogHistoryRepoImpl) Upsert(_ context.Context, history *entity.DbBinlogHistory) error {
	return gormx.Tx(func(db *gorm.DB) error {
		old := &entity.DbBinlogHistory{}
		err := db.Where("db_instance_id = ?", history.DbInstanceId).
			Where("sequence = ?", history.Sequence).
			First(old).Error
		switch {
		case err == nil:
			return db.Model(old).Select("create_time", "file_size", "first_event_time").Updates(history).Error
		case errors.Is(err, gorm.ErrRecordNotFound):
			return db.Create(history).Error
		default:
			return err
		}
	})
}
