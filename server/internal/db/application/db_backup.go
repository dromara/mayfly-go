package application

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/infrastructure/service"
	"mayfly-go/pkg/model"
	"time"
)

func newDbBackupApp(repositories *repository.Repositories) (*DbBackupApp, error) {
	scheduler, err := newDbScheduler[*entity.DbBackup](
		repositories.Backup,
		withRunBackupTask(repositories))
	if err != nil {
		return nil, err
	}
	app := &DbBackupApp{
		repo:         repositories.Backup,
		instanceRepo: repositories.Instance,
		scheduler:    scheduler,
	}
	return app, nil
}

type DbBackupApp struct {
	repo         repository.DbBackup
	instanceRepo repository.Instance
	scheduler    *dbScheduler[*entity.DbBackup]
}

func (app *DbBackupApp) Close() {
	app.Close()
}

func (app *DbBackupApp) Create(ctx context.Context, tasks ...*entity.DbBackup) error {
	return app.scheduler.AddTask(ctx, tasks...)
}

func (app *DbBackupApp) Save(ctx context.Context, task *entity.DbBackup) error {
	return app.scheduler.UpdateTask(ctx, task)
}

func (app *DbBackupApp) Delete(ctx context.Context, taskId uint64) error {
	// todo: 删除数据库备份历史文件
	return app.scheduler.DeleteTask(ctx, taskId)
}

func (app *DbBackupApp) Enable(ctx context.Context, taskId uint64) error {
	return app.scheduler.EnableTask(ctx, taskId)
}

func (app *DbBackupApp) Disable(ctx context.Context, taskId uint64) error {
	return app.scheduler.DisableTask(ctx, taskId)
}

// GetPageList 分页获取数据库备份任务
func (app *DbBackupApp) GetPageList(condition *entity.DbBackupQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.repo.GetDbBackupList(condition, pageParam, toEntity, orderBy...)
}

// GetDbNamesWithoutBackup 获取未配置定时备份的数据库名称
func (app *DbBackupApp) GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error) {
	return app.repo.GetDbNamesWithoutBackup(instanceId, dbNames)
}

func withRunBackupTask(repositories *repository.Repositories) dbSchedulerOption[*entity.DbBackup] {
	return func(scheduler *dbScheduler[*entity.DbBackup]) {
		scheduler.RunTask = func(ctx context.Context, task *entity.DbBackup) error {
			instance := new(entity.DbInstance)
			if err := repositories.Instance.GetById(instance, task.DbInstanceId); err != nil {
				return err
			}
			if err := instance.PwdDecrypt(); err != nil {
				return err
			}
			id, err := NewIncUUID()
			if err != nil {
				return err
			}
			history := &entity.DbBackupHistory{
				Uuid:         id.String(),
				DbBackupId:   task.Id,
				DbInstanceId: task.DbInstanceId,
				DbName:       task.DbName,
			}
			binlogInfo, err := service.NewDbInstanceSvc(instance, repositories).Backup(ctx, history)
			if err != nil {
				return err
			}
			now := time.Now()
			name := task.Name
			if len(name) == 0 {
				name = task.DbName
			}
			history.Name = fmt.Sprintf("%s[%s]", name, now.Format(time.DateTime))
			history.CreateTime = now
			history.BinlogFileName = binlogInfo.FileName
			history.BinlogSequence = binlogInfo.Sequence
			history.BinlogPosition = binlogInfo.Position

			if err := repositories.BackupHistory.Insert(ctx, history); err != nil {
				return err
			}
			return nil
		}
	}
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

func newDbBackupHistoryApp(repositories *repository.Repositories) (*DbBackupHistoryApp, error) {
	app := &DbBackupHistoryApp{
		repo: repositories.BackupHistory,
	}
	return app, nil
}

type DbBackupHistoryApp struct {
	repo repository.DbBackupHistory
}

// GetPageList 分页获取数据库备份历史
func (app *DbBackupHistoryApp) GetPageList(condition *entity.DbBackupHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.repo.GetHistories(condition, pageParam, toEntity, orderBy...)
}
