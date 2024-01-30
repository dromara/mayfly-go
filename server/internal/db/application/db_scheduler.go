package application

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/runner"
	"reflect"
	"sync"
	"time"
)

const (
	maxRunning = 8
)

type dbScheduler struct {
	mutex              sync.Mutex
	runner             *runner.Runner[entity.DbJob]
	dbApp              Db                          `inject:"DbApp"`
	backupRepo         repository.DbBackup         `inject:"DbBackupRepo"`
	backupHistoryRepo  repository.DbBackupHistory  `inject:"DbBackupHistoryRepo"`
	restoreRepo        repository.DbRestore        `inject:"DbRestoreRepo"`
	restoreHistoryRepo repository.DbRestoreHistory `inject:"DbRestoreHistoryRepo"`
	binlogRepo         repository.DbBinlog         `inject:"DbBinlogRepo"`
	binlogHistoryRepo  repository.DbBinlogHistory  `inject:"DbBinlogHistoryRepo"`
}

func newDbScheduler() *dbScheduler {
	scheduler := &dbScheduler{}
	scheduler.runner = runner.NewRunner[entity.DbJob](maxRunning, scheduler.runJob,
		runner.WithScheduleJob[entity.DbJob](scheduler.scheduleJob),
		runner.WithRunnableJob[entity.DbJob](scheduler.runnableJob),
		runner.WithUpdateJob[entity.DbJob](scheduler.updateJob),
	)
	return scheduler
}

func (s *dbScheduler) scheduleJob(job entity.DbJob) (time.Time, error) {
	return job.Schedule()
}

func (s *dbScheduler) UpdateJob(ctx context.Context, job entity.DbJob) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_ = s.runner.Update(ctx, job)
	return nil
}

func (s *dbScheduler) Close() {
	s.runner.Close()
}

func (s *dbScheduler) AddJob(ctx context.Context, jobs any) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	reflectValue := reflect.ValueOf(jobs)
	switch reflectValue.Kind() {
	case reflect.Array, reflect.Slice:
		reflectLen := reflectValue.Len()
		for i := 0; i < reflectLen; i++ {
			job := reflectValue.Index(i).Interface().(entity.DbJob)
			_ = s.runner.Add(ctx, job)
		}
	default:
		job := jobs.(entity.DbJob)
		_ = s.runner.Add(ctx, job)
	}
	return nil
}

func (s *dbScheduler) RemoveJob(ctx context.Context, jobType entity.DbJobType, jobId uint64) error {
	// todo: 删除数据库备份历史文件
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.runner.Remove(ctx, entity.FormatJobKey(jobType, jobId)); err != nil {
		return err
	}
	return nil
}

func (s *dbScheduler) EnableJob(ctx context.Context, job entity.DbJob) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_ = s.runner.Add(ctx, job)
	return nil
}

func (s *dbScheduler) DisableJob(ctx context.Context, jobType entity.DbJobType, jobId uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_ = s.runner.Remove(ctx, entity.FormatJobKey(jobType, jobId))
	return nil
}

func (s *dbScheduler) StartJobNow(ctx context.Context, job entity.DbJob) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_ = s.runner.StartNow(ctx, job)
	return nil
}

func (s *dbScheduler) backup(ctx context.Context, dbProgram dbi.DbProgram, job entity.DbJob) error {
	id, err := NewIncUUID()
	if err != nil {
		return err
	}
	backup := job.(*entity.DbBackup)
	history := &entity.DbBackupHistory{
		Uuid:         id.String(),
		DbBackupId:   backup.Id,
		DbInstanceId: backup.DbInstanceId,
		DbName:       backup.DbName,
	}
	binlogInfo, err := dbProgram.Backup(ctx, history)
	if err != nil {
		return err
	}
	now := time.Now()
	name := backup.DbName
	if len(backup.Name) > 0 {
		name = fmt.Sprintf("%s-%s", backup.DbName, backup.Name)
	}
	history.Name = fmt.Sprintf("%s[%s]", name, now.Format(time.DateTime))
	history.CreateTime = now
	history.BinlogFileName = binlogInfo.FileName
	history.BinlogSequence = binlogInfo.Sequence
	history.BinlogPosition = binlogInfo.Position

	if err := s.backupHistoryRepo.Insert(ctx, history); err != nil {
		return err
	}
	return nil
}

func (s *dbScheduler) restore(ctx context.Context, dbProgram dbi.DbProgram, job entity.DbJob) error {
	restore := job.(*entity.DbRestore)
	if restore.PointInTime.Valid {
		//if enabled, err := dbProgram.CheckBinlogEnabled(ctx); err != nil {
		//	return err
		//} else if !enabled {
		//	return errors.New("数据库未启用 BINLOG")
		//}
		//if enabled, err := dbProgram.CheckBinlogRowFormat(ctx); err != nil {
		//	return err
		//} else if !enabled {
		//	return errors.New("数据库未启用 BINLOG 行模式")
		//}
		//
		//latestBinlogSequence, earliestBackupSequence := int64(-1), int64(-1)
		//binlogHistory, ok, err := s.binlogHistoryRepo.GetLatestHistory(restore.DbInstanceId)
		//if err != nil {
		//	return err
		//}
		//if ok {
		//	latestBinlogSequence = binlogHistory.Sequence
		//} else {
		//	backupHistory, ok, err := s.backupHistoryRepo.GetEarliestHistory(restore.DbInstanceId)
		//	if err != nil {
		//		return err
		//	}
		//	if !ok {
		//		return nil
		//	}
		//	earliestBackupSequence = backupHistory.BinlogSequence
		//}
		//binlogFiles, err := dbProgram.FetchBinlogs(ctx, true, earliestBackupSequence, latestBinlogSequence)
		//if err != nil {
		//	return err
		//}
		//if err := s.binlogHistoryRepo.InsertWithBinlogFiles(ctx, restore.DbInstanceId, binlogFiles); err != nil {
		//	return err
		//}
		if err := s.fetchBinlog(ctx, dbProgram, job.GetInstanceId(), true); err != nil {
			return err
		}
		if err := s.restorePointInTime(ctx, dbProgram, restore); err != nil {
			return err
		}
	} else {
		backupHistory := &entity.DbBackupHistory{}
		if err := s.backupHistoryRepo.GetById(backupHistory, restore.DbBackupHistoryId); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = errors.New("备份历史已删除")
			}
			return err
		}
		if err := s.restoreBackupHistory(ctx, dbProgram, backupHistory); err != nil {
			return err
		}
	}

	history := &entity.DbRestoreHistory{
		CreateTime:  time.Now(),
		DbRestoreId: restore.Id,
	}
	if err := s.restoreHistoryRepo.Insert(ctx, history); err != nil {
		return err
	}
	return nil
}

//func (s *dbScheduler) updateLastStatus(ctx context.Context, job entity.DbJob) error {
//	switch typ := job.GetJobType(); typ {
//	case entity.DbJobTypeBackup:
//		return s.backupRepo.UpdateLastStatus(ctx, job)
//	case entity.DbJobTypeRestore:
//		return s.restoreRepo.UpdateLastStatus(ctx, job)
//	case entity.DbJobTypeBinlog:
//		return s.binlogRepo.UpdateLastStatus(ctx, job)
//	default:
//		panic(fmt.Errorf("无效的数据库任务类型: %v", typ))
//	}
//}

func (s *dbScheduler) updateJob(ctx context.Context, job entity.DbJob) error {
	switch typ := job.GetJobType(); typ {
	case entity.DbJobTypeBackup:
		return s.backupRepo.UpdateById(ctx, job)
	case entity.DbJobTypeRestore:
		return s.restoreRepo.UpdateById(ctx, job)
	case entity.DbJobTypeBinlog:
		return s.binlogRepo.UpdateById(ctx, job)
	default:
		return fmt.Errorf("无效的数据库任务类型: %v", typ)
	}
}

func (s *dbScheduler) runJob(ctx context.Context, job entity.DbJob) error {
	//job.SetLastStatus(entity.DbJobRunning, nil)
	//if err := s.updateLastStatus(ctx, job); err != nil {
	//	logx.Errorf("failed to update job status: %v", err)
	//	return
	//}

	//var errRun error
	conn, err := s.dbApp.GetDbConnByInstanceId(job.GetInstanceId())
	if err != nil {
		return err
	}
	dbProgram := conn.GetDialect().GetDbProgram()
	switch typ := job.GetJobType(); typ {
	case entity.DbJobTypeBackup:
		return s.backup(ctx, dbProgram, job)
	case entity.DbJobTypeRestore:
		return s.restore(ctx, dbProgram, job)
	case entity.DbJobTypeBinlog:
		return s.fetchBinlog(ctx, dbProgram, job.GetInstanceId(), false)
	default:
		return fmt.Errorf("无效的数据库任务类型: %v", typ)
	}
	//status := entity.DbJobSuccess
	//if errRun != nil {
	//	status = entity.DbJobFailed
	//}
	//job.SetLastStatus(status, errRun)
	//if err := s.updateLastStatus(ctx, job); err != nil {
	//	logx.Errorf("failed to update job status: %v", err)
	//	return
	//}
}

func (s *dbScheduler) runnableJob(job entity.DbJob, next runner.NextJobFunc[entity.DbJob]) (bool, error) {
	if job.IsExpired() {
		return false, runner.ErrJobExpired
	}
	const maxCountByInstanceId = 4
	const maxCountByDbName = 1
	var countByInstanceId, countByDbName int
	for item, ok := next(); ok; item, ok = next() {
		if job.GetInstanceId() == item.GetInstanceId() {
			countByInstanceId++
			if countByInstanceId >= maxCountByInstanceId {
				return false, nil
			}

			if relatedToBinlog(job.GetJobType()) {
				// todo: 恢复数据库前触发 BINLOG 同步，BINLOG 同步完成后才能恢复数据库
				if relatedToBinlog(item.GetJobType()) {
					return false, nil
				}
			}

			if job.GetDbName() == item.GetDbName() {
				countByDbName++
				if countByDbName >= maxCountByDbName {
					return false, nil
				}
			}
		}
	}
	return true, nil
}

func relatedToBinlog(typ entity.DbJobType) bool {
	return typ == entity.DbJobTypeRestore || typ == entity.DbJobTypeBinlog
}

func (s *dbScheduler) restorePointInTime(ctx context.Context, dbProgram dbi.DbProgram, job *entity.DbRestore) error {
	binlogHistory, err := s.binlogHistoryRepo.GetHistoryByTime(job.DbInstanceId, job.PointInTime.Time)
	if err != nil {
		return err
	}
	position, err := dbProgram.GetBinlogEventPositionAtOrAfterTime(ctx, binlogHistory.FileName, job.PointInTime.Time)
	if err != nil {
		return err
	}
	target := &entity.BinlogInfo{
		FileName: binlogHistory.FileName,
		Sequence: binlogHistory.Sequence,
		Position: position,
	}
	backupHistory, err := s.backupHistoryRepo.GetLatestHistory(job.DbInstanceId, job.DbName, target)
	if err != nil {
		return err
	}
	start := &entity.BinlogInfo{
		FileName: backupHistory.BinlogFileName,
		Sequence: backupHistory.BinlogSequence,
		Position: backupHistory.BinlogPosition,
	}
	binlogHistories, err := s.binlogHistoryRepo.GetHistories(job.DbInstanceId, start, target)
	if err != nil {
		return err
	}
	restoreInfo := &dbi.RestoreInfo{
		BackupHistory:   backupHistory,
		BinlogHistories: binlogHistories,
		StartPosition:   backupHistory.BinlogPosition,
		TargetPosition:  target.Position,
		TargetTime:      job.PointInTime.Time,
	}
	if err := dbProgram.ReplayBinlog(ctx, job.DbName, job.DbName, restoreInfo); err != nil {
		return err
	}
	if err := s.restoreBackupHistory(ctx, dbProgram, backupHistory); err != nil {
		return err
	}
	// 由于 ReplayBinlog 未记录 BINLOG 事件，系统自动备份，避免数据丢失
	backup := &entity.DbBackup{
		DbInstanceId: backupHistory.DbInstanceId,
		DbName:       backupHistory.DbName,
		Enabled:      true,
		Repeated:     false,
		StartTime:    time.Now(),
		Interval:     0,
		Name:         "系统备份",
	}
	backup.Id = backupHistory.DbBackupId
	if err := s.backup(ctx, dbProgram, backup); err != nil {
		return err
	}
	return nil
}

func (s *dbScheduler) restoreBackupHistory(ctx context.Context, program dbi.DbProgram, backupHistory *entity.DbBackupHistory) (retErr error) {
	ok, err := s.backupHistoryRepo.UpdateRestoring(true, backupHistory.Id)
	if err != nil {
		return err
	}
	defer func() {
		_, err = s.backupHistoryRepo.UpdateRestoring(false, backupHistory.Id)
		if err == nil {
			return
		}
		if retErr == nil {
			retErr = err
			return
		}
		retErr = fmt.Errorf("%w, %w", retErr, err)
	}()
	if !ok {
		return errors.New("关联的数据库备份历史已删除")
	}
	return program.RestoreBackupHistory(ctx, backupHistory.DbName, backupHistory.DbBackupId, backupHistory.Uuid)
}

func (s *dbScheduler) fetchBinlog(ctx context.Context, dbProgram dbi.DbProgram, instanceId uint64, downloadLatestBinlogFile bool) error {
	if enabled, err := dbProgram.CheckBinlogEnabled(ctx); err != nil {
		return err
	} else if !enabled {
		return errors.New("数据库未启用 BINLOG")
	}
	if enabled, err := dbProgram.CheckBinlogRowFormat(ctx); err != nil {
		return err
	} else if !enabled {
		return errors.New("数据库未启用 BINLOG 行模式")
	}

	latestBinlogSequence, earliestBackupSequence := int64(-1), int64(-1)
	binlogHistory, ok, err := s.binlogHistoryRepo.GetLatestHistory(instanceId)
	if err != nil {
		return err
	}
	if ok {
		latestBinlogSequence = binlogHistory.Sequence
	} else {
		backupHistory, ok, err := s.backupHistoryRepo.GetEarliestHistory(instanceId)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		earliestBackupSequence = backupHistory.BinlogSequence
	}
	binlogFiles, err := dbProgram.FetchBinlogs(ctx, downloadLatestBinlogFile, earliestBackupSequence, latestBinlogSequence)
	if err != nil {
		return err
	}
	return s.binlogHistoryRepo.InsertWithBinlogFiles(ctx, instanceId, binlogFiles)
}
