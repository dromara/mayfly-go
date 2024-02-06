package application

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"math"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/timex"
	"sync"
	"time"

	"github.com/google/uuid"
)

const maxBackupHistoryDays = 30

var (
	errRestoringBackupHistory = errors.New("正在从备份历史中恢复数据库")
)

type DbBackupApp struct {
	scheduler         *dbScheduler               `inject:"DbScheduler"`
	backupRepo        repository.DbBackup        `inject:"DbBackupRepo"`
	backupHistoryRepo repository.DbBackupHistory `inject:"DbBackupHistoryRepo"`
	restoreRepo       repository.DbRestore       `inject:"DbRestoreRepo"`
	dbApp             Db                         `inject:"DbApp"`
	mutex             sync.Mutex
	closed            chan struct{}
	wg                sync.WaitGroup
	ctx               context.Context
	cancel            context.CancelFunc
}

func (app *DbBackupApp) Init() error {
	var jobs []*entity.DbBackup
	if err := app.backupRepo.ListToDo(&jobs); err != nil {
		return err
	}
	if err := app.scheduler.AddJob(context.Background(), jobs); err != nil {
		return err
	}
	app.ctx, app.cancel = context.WithCancel(context.Background())
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		for app.ctx.Err() == nil {
			if err := app.prune(app.ctx); err != nil {
				logx.Errorf("清理数据库备份历史失败: %s", err.Error())
				timex.SleepWithContext(app.ctx, time.Minute*15)
				continue
			}
			timex.SleepWithContext(app.ctx, time.Hour*24)
		}
	}()
	return nil
}

func (app *DbBackupApp) prune(ctx context.Context) error {
	var jobs []*entity.DbBackup
	if err := app.backupRepo.ListByCond(map[string]any{}, &jobs); err != nil {
		return err
	}
	for _, job := range jobs {
		if ctx.Err() != nil {
			return nil
		}
		var histories []*entity.DbBackupHistory
		historyCond := map[string]any{
			"db_backup_id": job.Id,
		}
		if err := app.backupHistoryRepo.ListByCondOrder(historyCond, &histories, "id"); err != nil {
			return err
		}
		expiringTime := time.Now().Add(-math.MaxInt64)
		if job.MaxSaveDays > 0 {
			expiringTime = time.Now().Add(-time.Hour * 24 * time.Duration(job.MaxSaveDays+1))
		}
		for _, history := range histories {
			if ctx.Err() != nil {
				return nil
			}
			if history.CreateTime.After(expiringTime) {
				break
			}
			err := app.DeleteHistory(ctx, history.Id)
			if errors.Is(err, errRestoringBackupHistory) {
				break
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (app *DbBackupApp) Close() {
	app.scheduler.Close()
	if app.cancel != nil {
		app.cancel()
		app.cancel = nil
	}
	app.wg.Wait()
}

func (app *DbBackupApp) Create(ctx context.Context, jobs []*entity.DbBackup) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if err := app.backupRepo.AddJob(ctx, jobs); err != nil {
		return err
	}
	return app.scheduler.AddJob(ctx, jobs)
}

func (app *DbBackupApp) Update(ctx context.Context, job *entity.DbBackup) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if err := app.backupRepo.UpdateById(ctx, job); err != nil {
		return err
	}
	_ = app.scheduler.UpdateJob(ctx, job)
	return nil
}

func (app *DbBackupApp) Delete(ctx context.Context, jobId uint64) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if err := app.scheduler.RemoveJob(ctx, entity.DbJobTypeBackup, jobId); err != nil {
		return err
	}
	history := &entity.DbBackupHistory{
		DbBackupId: jobId,
	}
	err := app.backupHistoryRepo.GetBy(history, "name")
	switch {
	default:
		return err
	case err == nil:
		return fmt.Errorf("请先删除关联的数据库备份历史【%s】", history.Name)
	case errors.Is(err, gorm.ErrRecordNotFound):
	}
	if err := app.backupRepo.DeleteById(ctx, jobId); err != nil {
		return err
	}
	return nil
}

func (app *DbBackupApp) Enable(ctx context.Context, jobId uint64) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	repo := app.backupRepo
	job := &entity.DbBackup{}
	if err := repo.GetById(job, jobId); err != nil {
		return err
	}
	if job.IsEnabled() {
		return nil
	}
	if job.IsExpired() {
		return errors.New("任务已过期")
	}
	_ = app.scheduler.EnableJob(ctx, job)
	if err := repo.UpdateEnabled(ctx, jobId, true); err != nil {
		logx.Errorf("数据库备份任务已启用( jobId: %d )，任务状态保存失败: %v", jobId, err)
		return err
	}
	return nil
}

func (app *DbBackupApp) Disable(ctx context.Context, jobId uint64) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	repo := app.backupRepo
	job := &entity.DbBackup{}
	if err := repo.GetById(job, jobId); err != nil {
		return err
	}
	if !job.IsEnabled() {
		return nil
	}
	_ = app.scheduler.DisableJob(ctx, entity.DbJobTypeBackup, jobId)
	if err := repo.UpdateEnabled(ctx, jobId, false); err != nil {
		logx.Errorf("数据库恢复任务已禁用( jobId: %d )，任务状态保存失败: %v", jobId, err)
		return err
	}
	return nil
}

func (app *DbBackupApp) StartNow(ctx context.Context, jobId uint64) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	job := &entity.DbBackup{}
	if err := app.backupRepo.GetById(job, jobId); err != nil {
		return err
	}
	if !job.IsEnabled() {
		return errors.New("任务未启用")
	}
	_ = app.scheduler.StartJobNow(ctx, job)
	return nil
}

// GetPageList 分页获取数据库备份任务
func (app *DbBackupApp) GetPageList(condition *entity.DbBackupQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.backupRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// GetDbNamesWithoutBackup 获取未配置定时备份的数据库名称
func (app *DbBackupApp) GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error) {
	return app.backupRepo.GetDbNamesWithoutBackup(instanceId, dbNames)
}

// GetHistoryPageList 分页获取数据库备份历史
func (app *DbBackupApp) GetHistoryPageList(condition *entity.DbBackupHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.backupHistoryRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (app *DbBackupApp) GetHistories(backupHistoryIds []uint64, toEntity any) error {
	return app.backupHistoryRepo.GetHistories(backupHistoryIds, toEntity)
}

func NewIncUUID() (uuid.UUID, error) {
	var uid uuid.UUID
	now, seq, err := uuid.GetTime()
	if err != nil {
		return uid, err
	}
	timeHi := uint32((now >> 28) & 0xffffffff)
	timeMid := uint16((now >> 12) & 0xffff)
	timeLow := uint16(now & 0x0fff)
	timeLow |= 0x1000 // Version 1

	binary.BigEndian.PutUint32(uid[0:], timeHi)
	binary.BigEndian.PutUint16(uid[4:], timeMid)
	binary.BigEndian.PutUint16(uid[6:], timeLow)
	binary.BigEndian.PutUint16(uid[8:], seq)

	copy(uid[10:], uuid.NodeID())

	return uid, nil
}

func (app *DbBackupApp) DeleteHistory(ctx context.Context, historyId uint64) (retErr error) {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if _, err := app.backupHistoryRepo.UpdateDeleting(false, historyId); err != nil {
		return err
	}
	ok, err := app.backupHistoryRepo.UpdateDeleting(true, historyId)
	if err != nil {
		return err
	}
	if !ok {
		return errRestoringBackupHistory
	}
	job := &entity.DbBackupHistory{}
	if err := app.backupHistoryRepo.GetById(job, historyId); err != nil {
		return err
	}
	conn, err := app.dbApp.GetDbConnByInstanceId(job.DbInstanceId)
	if err != nil {
		return err
	}
	dbProgram, err := conn.GetDialect().GetDbProgram()
	if err != nil {
		return err
	}
	if err := dbProgram.RemoveBackupHistory(ctx, job.DbBackupId, job.Uuid); err != nil {
		return err
	}
	return app.backupHistoryRepo.DeleteById(ctx, historyId)
}
