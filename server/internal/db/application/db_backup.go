package application

import (
	"context"
	"encoding/binary"
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/model"
	"time"

	"github.com/google/uuid"
)

func newDbBackupApp(repositories *repository.Repositories, dbApp Db) (*DbBackupApp, error) {
	app := &DbBackupApp{
		backupRepo:        repositories.Backup,
		instanceRepo:      repositories.Instance,
		backupHistoryRepo: repositories.BackupHistory,
		dbApp:             dbApp,
	}
	scheduler, err := newDbScheduler[*entity.DbBackup](
		repositories.Backup,
		withRunBackupTask(app))
	if err != nil {
		return nil, err
	}
	app.scheduler = scheduler
	return app, nil
}

type DbBackupApp struct {
	backupRepo        repository.DbBackup
	instanceRepo      repository.Instance
	backupHistoryRepo repository.DbBackupHistory
	dbApp             Db
	scheduler         *dbScheduler[*entity.DbBackup]
}

func (app *DbBackupApp) Close() {
	app.scheduler.Close()
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

func (app *DbBackupApp) Start(ctx context.Context, taskId uint64) error {
	return app.scheduler.StartTask(ctx, taskId)
}

// GetPageList 分页获取数据库备份任务
func (app *DbBackupApp) GetPageList(condition *entity.DbBackupQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.backupRepo.GetDbBackupList(condition, pageParam, toEntity, orderBy...)
}

// GetDbNamesWithoutBackup 获取未配置定时备份的数据库名称
func (app *DbBackupApp) GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error) {
	return app.backupRepo.GetDbNamesWithoutBackup(instanceId, dbNames)
}

// GetPageList 分页获取数据库备份历史
func (app *DbBackupApp) GetHistoryPageList(condition *entity.DbBackupHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.backupHistoryRepo.GetHistories(condition, pageParam, toEntity, orderBy...)
}

func withRunBackupTask(app *DbBackupApp) dbSchedulerOption[*entity.DbBackup] {
	return func(scheduler *dbScheduler[*entity.DbBackup]) {
		scheduler.RunTask = app.runTask
	}
}

func (app *DbBackupApp) runTask(ctx context.Context, task *entity.DbBackup) error {
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
	conn, err := app.dbApp.GetDbConnByInstanceId(task.DbInstanceId)
	if err != nil {
		return err
	}
	dbProgram := conn.GetDialect().GetDbProgram()
	binlogInfo, err := dbProgram.Backup(ctx, history)
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

	if err := app.backupHistoryRepo.Insert(ctx, history); err != nil {
		return err
	}
	return nil
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

// func newDbBackupHistoryApp(repositories *repository.Repositories) (*DbBackupHistoryApp, error) {
// 	app := &DbBackupHistoryApp{
// 		repo: repositories.BackupHistory,
// 	}
// 	return app, nil
// }

// type DbBackupHistoryApp struct {
// 	repo repository.DbBackupHistory
// }

// // GetPageList 分页获取数据库备份历史
// func (app *DbBackupHistoryApp) GetPageList(condition *entity.DbBackupHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
// 	return app.repo.GetHistories(condition, pageParam, toEntity, orderBy...)
// }
