package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"time"
)

type DbBinlogHistory interface {
	base.Repo[*entity.DbBinlogHistory]

	GetHistories(instanceId uint64, start, target *entity.BinlogInfo) ([]*entity.DbBinlogHistory, error)

	GetHistoryByTime(instanceId uint64, targetTime time.Time) (*entity.DbBinlogHistory, error)

	GetLatestHistory(instanceId uint64) (*entity.DbBinlogHistory, bool, error)

	InsertWithBinlogFiles(ctx context.Context, instanceId uint64, binlogFiles []*entity.BinlogFile) error

	Upsert(ctx context.Context, history *entity.DbBinlogHistory) error

	GetHistoriesBeforeSequence(ctx context.Context, instanceId uint64, binlogSeq int64, histories *[]*entity.DbBinlogHistory) error
}
