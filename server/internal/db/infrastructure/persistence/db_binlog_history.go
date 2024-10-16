package persistence

import (
	"context"
	"errors"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"time"

	"gorm.io/gorm"
)

var _ repository.DbBinlogHistory = (*dbBinlogHistoryRepoImpl)(nil)

type dbBinlogHistoryRepoImpl struct {
	base.RepoImpl[*entity.DbBinlogHistory]
}

func NewDbBinlogHistoryRepo() repository.DbBinlogHistory {
	dr := &dbBinlogHistoryRepoImpl{}
	dr.M = new(entity.DbBinlogHistory)
	return dr
}

func (repo *dbBinlogHistoryRepoImpl) GetHistoryByTime(instanceId uint64, targetTime time.Time) (*entity.DbBinlogHistory, error) {
	qc := model.NewCond().
		Eq("db_instance_id", instanceId).
		Le("first_event_time", targetTime).
		OrderByDesc("first_event_time")
	history := &entity.DbBinlogHistory{}
	if err := repo.GetByCond(qc.Dest(history)); err != nil {
		return nil, err
	}
	return history, nil
}

func (repo *dbBinlogHistoryRepoImpl) GetHistories(instanceId uint64, start, target *entity.BinlogInfo) ([]*entity.DbBinlogHistory, error) {
	qc := model.NewCond().
		Eq("db_instance_id", instanceId).
		Ge("sequence", start.Sequence).
		Le("sequence", target.Sequence).
		OrderByAsc("sequence")
	histories, err := repo.SelectByCond(qc)
	if err != nil {
		return nil, err
	}
	if len(histories) == 0 {
		return nil, errors.New("未找到满足条件的 binlog 文件")
	}
	return histories, nil
}

func (repo *dbBinlogHistoryRepoImpl) GetLatestHistory(instanceId uint64) (*entity.DbBinlogHistory, bool, error) {
	history := &entity.DbBinlogHistory{}
	qc := model.NewCond().
		Eq("db_instance_id", instanceId).
		OrderByDesc("sequence").
		Dest(history)
	err := repo.GetByCond(qc)
	switch {
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
			Scopes(gormx.UndeleteScope).
			First(old).Error
		switch {
		case err == nil:
			return db.Model(old).Select("create_time", "file_size", "first_event_time", "last_event_time").Updates(history).Error
		case errors.Is(err, gorm.ErrRecordNotFound):
			return db.Create(history).Error
		default:
			return err
		}
	})
}

func (repo *dbBinlogHistoryRepoImpl) InsertWithBinlogFiles(ctx context.Context, instanceId uint64, binlogFiles []*entity.BinlogFile) error {
	if len(binlogFiles) == 0 {
		return nil
	}
	histories := make([]*entity.DbBinlogHistory, 0, len(binlogFiles))
	for _, fileOnServer := range binlogFiles {
		if !fileOnServer.Downloaded {
			break
		}
		history := &entity.DbBinlogHistory{
			CreateTime:     time.Now(),
			FileName:       fileOnServer.Name,
			FileSize:       fileOnServer.RemoteSize,
			Sequence:       fileOnServer.Sequence,
			FirstEventTime: fileOnServer.FirstEventTime,
			LastEventTime:  fileOnServer.LastEventTime,
			DbInstanceId:   instanceId,
		}
		histories = append(histories, history)
	}
	if len(histories) > 1 {
		if err := repo.BatchInsert(ctx, histories[:len(histories)-1]); err != nil {
			return err
		}
	}
	if len(histories) > 0 {
		if err := repo.Upsert(ctx, histories[len(histories)-1]); err != nil {
			return err
		}
	}
	return nil
}

func (repo *dbBinlogHistoryRepoImpl) GetHistoriesBeforeSequence(ctx context.Context, instanceId uint64, binlogSeq int64, histories *[]*entity.DbBinlogHistory) error {
	return global.Db.Model(repo.NewModel()).
		Where("db_instance_id = ?", instanceId).
		Where("sequence < ?", binlogSeq).
		Scopes(gormx.UndeleteScope).
		Order("id").
		Find(histories).
		Error
}
