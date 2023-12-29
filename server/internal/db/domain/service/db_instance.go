package service

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
)

type DbInstanceSvc interface {
	Backup(ctx context.Context, backupHistory *entity.DbBackupHistory) (*entity.BinlogInfo, error)
	Restore(ctx context.Context, task *entity.DbRestore) error
	FetchBinlogs(ctx context.Context, downloadLatestBinlogFile bool) error
}
