package application

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/logx"
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
	dbApp              Db
	backupRepo         repository.DbBackup
	backupHistoryRepo  repository.DbBackupHistory
	restoreRepo        repository.DbRestore
	restoreHistoryRepo repository.DbRestoreHistory
	binlogHistoryRepo  repository.DbBinlogHistory
}

func newDbScheduler(repositories *repository.Repositories) (*dbScheduler, error) {
	scheduler := &dbScheduler{
		runner:             runner.NewRunner[entity.DbJob](maxRunning),
		dbApp:              dbApp,
		backupRepo:         repositories.Backup,
		backupHistoryRepo:  repositories.BackupHistory,
		restoreRepo:        repositories.Restore,
		restoreHistoryRepo: repositories.RestoreHistory,
		binlogHistoryRepo:  repositories.BinlogHistory,
	}
	return scheduler, nil
}

func (s *dbScheduler) repo(typ entity.DbJobType) repository.DbJob {
	switch typ {
	case entity.DbJobTypeBackup:
		return s.backupRepo
	case entity.DbJobTypeRestore:
		return s.restoreRepo
	default:
		panic(errors.New(fmt.Sprintf("无效的数据库任务类型: %v", typ)))
	}
}

func (s *dbScheduler) UpdateJob(ctx context.Context, job entity.DbJob) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.repo(job.GetJobType()).UpdateById(ctx, job); err != nil {
		return err
	}
	job.SetRun(s.run)
	job.SetRunnable(s.runnable)
	_ = s.runner.UpdateOrAdd(ctx, job)
	return nil
}

func (s *dbScheduler) Close() {
	s.runner.Close()
}

func (s *dbScheduler) AddJob(ctx context.Context, saving bool, jobType entity.DbJobType, jobs any) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if saving {
		if err := s.repo(jobType).AddJob(ctx, jobs); err != nil {
			return err
		}
	}

	reflectValue := reflect.ValueOf(jobs)
	switch reflectValue.Kind() {
	case reflect.Array, reflect.Slice:
		reflectLen := reflectValue.Len()
		for i := 0; i < reflectLen; i++ {
			job := reflectValue.Index(i).Interface().(entity.DbJob)
			job.SetJobType(jobType)
			if !job.Schedule() {
				continue
			}
			job.SetRun(s.run)
			job.SetRunnable(s.runnable)
			_ = s.runner.Add(ctx, job)
		}
	default:
		job := jobs.(entity.DbJob)
		job.SetJobType(jobType)
		if !job.Schedule() {
			return nil
		}
		job.SetRun(s.run)
		job.SetRunnable(s.runnable)
		_ = s.runner.Add(ctx, job)
	}
	return nil
}

func (s *dbScheduler) RemoveJob(ctx context.Context, jobType entity.DbJobType, jobId uint64) error {
	// todo: 删除数据库备份历史文件
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.repo(jobType).DeleteById(ctx, jobId); err != nil {
		return err
	}
	_ = s.runner.Remove(ctx, entity.FormatJobKey(jobType, jobId))
	return nil
}

func (s *dbScheduler) EnableJob(ctx context.Context, jobType entity.DbJobType, jobId uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	repo := s.repo(jobType)
	job := entity.NewDbJob(jobType)
	if err := repo.GetById(job, jobId); err != nil {
		return err
	}
	if job.IsEnabled() {
		return nil
	}
	job.GetJobBase().Enabled = true
	if err := repo.UpdateEnabled(ctx, jobId, true); err != nil {
		return err
	}
	job.SetRun(s.run)
	job.SetRunnable(s.runnable)
	_ = s.runner.Add(ctx, job)
	return nil
}

func (s *dbScheduler) DisableJob(ctx context.Context, jobType entity.DbJobType, jobId uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	repo := s.repo(jobType)
	job := entity.NewDbJob(jobType)
	if err := repo.GetById(job, jobId); err != nil {
		return err
	}
	if !job.IsEnabled() {
		return nil
	}
	if err := repo.UpdateEnabled(ctx, jobId, false); err != nil {
		return err
	}
	_ = s.runner.Remove(ctx, job.GetKey())
	return nil
}

func (s *dbScheduler) StartJobNow(ctx context.Context, jobType entity.DbJobType, jobId uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	job := entity.NewDbJob(jobType)
	if err := s.repo(jobType).GetById(job, jobId); err != nil {
		return err
	}
	if !job.IsEnabled() {
		return errors.New("任务未启用")
	}
	job.GetJobBase().Deadline = time.Now()
	job.SetRun(s.run)
	job.SetRunnable(s.runnable)
	_ = s.runner.StartNow(ctx, job)
	return nil
}

func (s *dbScheduler) backupMysql(ctx context.Context, job entity.DbJob) error {
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
	conn, err := s.dbApp.GetDbConnByInstanceId(backup.DbInstanceId)
	if err != nil {
		return err
	}
	dbProgram := conn.GetDialect().GetDbProgram()
	binlogInfo, err := dbProgram.Backup(ctx, history)
	if err != nil {
		return err
	}
	now := time.Now()
	name := backup.Name
	if len(name) == 0 {
		name = backup.DbName
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

func (s *dbScheduler) restoreMysql(ctx context.Context, job entity.DbJob) error {
	restore := job.(*entity.DbRestore)
	conn, err := s.dbApp.GetDbConnByInstanceId(restore.DbInstanceId)
	if err != nil {
		return err
	}
	dbProgram := conn.GetDialect().GetDbProgram()
	if restore.PointInTime.Valid {
		latestBinlogSequence, earliestBackupSequence := int64(-1), int64(-1)
		binlogHistory, ok, err := s.binlogHistoryRepo.GetLatestHistory(restore.DbInstanceId)
		if err != nil {
			return err
		}
		if ok {
			latestBinlogSequence = binlogHistory.Sequence
		} else {
			backupHistory, ok, err := s.backupHistoryRepo.GetEarliestHistory(restore.DbInstanceId)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			earliestBackupSequence = backupHistory.BinlogSequence
		}
		binlogFiles, err := dbProgram.FetchBinlogs(ctx, true, earliestBackupSequence, latestBinlogSequence)
		if err != nil {
			return err
		}
		if err := s.binlogHistoryRepo.InsertWithBinlogFiles(ctx, restore.DbInstanceId, binlogFiles); err != nil {
			return err
		}
		if err := s.restorePointInTime(ctx, dbProgram, restore); err != nil {
			return err
		}
	} else {
		if err := s.restoreBackupHistory(ctx, dbProgram, restore); err != nil {
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

func (s *dbScheduler) run(ctx context.Context, job entity.DbJob) {
	job.SetLastStatus(entity.DbJobRunning, nil)
	if err := s.repo(job.GetJobType()).UpdateLastStatus(ctx, job); err != nil {
		logx.Errorf("failed to update job status: %v", err)
		return
	}

	var errRun error
	switch typ := job.GetJobType(); typ {
	case entity.DbJobTypeBackup:
		errRun = s.backupMysql(ctx, job)
	case entity.DbJobTypeRestore:
		errRun = s.restoreMysql(ctx, job)
	default:
		errRun = errors.New(fmt.Sprintf("无效的数据库任务类型: %v", typ))
	}
	status := entity.DbJobSuccess
	if errRun != nil {
		status = entity.DbJobFailed
	}
	job.SetLastStatus(status, errRun)
	if err := s.repo(job.GetJobType()).UpdateLastStatus(ctx, job); err != nil {
		logx.Errorf("failed to update job status: %v", err)
		return
	}
}

func (s *dbScheduler) runnable(job entity.DbJob, next runner.NextFunc) bool {
	const maxCountByInstanceId = 4
	const maxCountByDbName = 1
	var countByInstanceId, countByDbName int
	jobBase := job.GetJobBase()
	for item, ok := next(); ok; item, ok = next() {
		itemBase := item.(entity.DbJob).GetJobBase()
		if jobBase.DbInstanceId == itemBase.DbInstanceId {
			countByInstanceId++
			if countByInstanceId >= maxCountByInstanceId {
				return false
			}
			if jobBase.DbName == itemBase.DbName {
				countByDbName++
				if countByDbName >= maxCountByDbName {
					return false
				}
			}
		}
	}
	return true
}

func (s *dbScheduler) restorePointInTime(ctx context.Context, program dbm.DbProgram, job *entity.DbRestore) error {
	binlogHistory, err := s.binlogHistoryRepo.GetHistoryByTime(job.DbInstanceId, job.PointInTime.Time)
	if err != nil {
		return err
	}
	position, err := program.GetBinlogEventPositionAtOrAfterTime(ctx, binlogHistory.FileName, job.PointInTime.Time)
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
	restoreInfo := &dbm.RestoreInfo{
		BackupHistory:   backupHistory,
		BinlogHistories: binlogHistories,
		StartPosition:   backupHistory.BinlogPosition,
		TargetPosition:  target.Position,
		TargetTime:      job.PointInTime.Time,
	}
	if err := program.RestoreBackupHistory(ctx, backupHistory.DbName, backupHistory.DbBackupId, backupHistory.Uuid); err != nil {
		return err
	}
	return program.ReplayBinlog(ctx, job.DbName, job.DbName, restoreInfo)
}

func (s *dbScheduler) restoreBackupHistory(ctx context.Context, program dbm.DbProgram, job *entity.DbRestore) error {
	backupHistory := &entity.DbBackupHistory{}
	if err := s.backupHistoryRepo.GetById(backupHistory, job.DbBackupHistoryId); err != nil {
		return err
	}
	return program.RestoreBackupHistory(ctx, backupHistory.DbName, backupHistory.DbBackupId, backupHistory.Uuid)
}
