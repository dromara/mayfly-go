package dbm

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"path/filepath"
	"time"
)

type DbProgram interface {
	Backup(ctx context.Context, backupHistory *entity.DbBackupHistory) (*entity.BinlogInfo, error)

	FetchBinlogs(ctx context.Context, downloadLatestBinlogFile bool, earliestBackupSequence, latestBinlogSequence int64) ([]*entity.BinlogFile, error)

	ReplayBinlog(ctx context.Context, originalDatabase, targetDatabase string, restoreInfo *RestoreInfo) error

	RestoreBackupHistory(ctx context.Context, dbName string, dbBackupId uint64, dbBackupHistoryUuid string) error

	GetBinlogEventPositionAtOrAfterTime(ctx context.Context, binlogName string, targetTime time.Time) (position int64, parseErr error)
}

type RestoreInfo struct {
	BackupHistory   *entity.DbBackupHistory
	BinlogHistories []*entity.DbBinlogHistory
	StartPosition   int64
	TargetPosition  int64
	TargetTime      time.Time
}

func (ri *RestoreInfo) GetBinlogPaths(binlogDir string) []string {
	files := make([]string, 0, len(ri.BinlogHistories))
	for _, history := range ri.BinlogHistories {
		files = append(files, filepath.Join(binlogDir, history.FileName))
	}
	return files
}
