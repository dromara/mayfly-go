package application

import (
	"context"
	"encoding/binary"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/model"

	"github.com/google/uuid"
)

type DbBackupApp struct {
	scheduler         *dbScheduler               `inject:"DbScheduler"`
	backupRepo        repository.DbBackup        `inject:"DbBackupRepo"`
	backupHistoryRepo repository.DbBackupHistory `inject:"DbBackupHistoryRepo"`
}

func (app *DbBackupApp) Init() error {
	var jobs []*entity.DbBackup
	if err := app.backupRepo.ListToDo(&jobs); err != nil {
		return err
	}
	if err := app.scheduler.AddJob(context.Background(), false, entity.DbJobTypeBackup, jobs); err != nil {
		return err
	}
	return nil
}

func (app *DbBackupApp) Close() {
	app.scheduler.Close()
}

func (app *DbBackupApp) Create(ctx context.Context, jobs []*entity.DbBackup) error {
	return app.scheduler.AddJob(ctx, true /* 保存到数据库 */, entity.DbJobTypeBackup, jobs)
}

func (app *DbBackupApp) Update(ctx context.Context, job *entity.DbBackup) error {
	return app.scheduler.UpdateJob(ctx, job)
}

func (app *DbBackupApp) Delete(ctx context.Context, jobId uint64) error {
	// todo: 删除数据库备份历史文件
	return app.scheduler.RemoveJob(ctx, entity.DbJobTypeBackup, jobId)
}

func (app *DbBackupApp) Enable(ctx context.Context, jobId uint64) error {
	return app.scheduler.EnableJob(ctx, entity.DbJobTypeBackup, jobId)
}

func (app *DbBackupApp) Disable(ctx context.Context, jobId uint64) error {
	return app.scheduler.DisableJob(ctx, entity.DbJobTypeBackup, jobId)
}

func (app *DbBackupApp) Start(ctx context.Context, jobId uint64) error {
	return app.scheduler.StartJobNow(ctx, entity.DbJobTypeBackup, jobId)
}

// GetPageList 分页获取数据库备份任务
func (app *DbBackupApp) GetPageList(condition *entity.DbJobQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.backupRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// GetDbNamesWithoutBackup 获取未配置定时备份的数据库名称
func (app *DbBackupApp) GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error) {
	return app.backupRepo.GetDbNamesWithoutBackup(instanceId, dbNames)
}

// GetHistoryPageList 分页获取数据库备份历史
func (app *DbBackupApp) GetHistoryPageList(condition *entity.DbBackupHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.backupHistoryRepo.GetHistories(condition, pageParam, toEntity, orderBy...)
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
