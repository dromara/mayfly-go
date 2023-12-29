package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/domain/service"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
	"time"
)

var _ service.DbBackupSvc = (*DbBackupSvcImpl)(nil)

type DbBackupSvcImpl struct {
	repo         repository.DbBackup
	instanceRepo repository.Instance
	scheduler    *Scheduler[*entity.DbBackup]
	binlogSvc    service.DbBinlogSvc
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

func withRunBackupTask(repositories *repository.Repositories, binlogSvc service.DbBinlogSvc) SchedulerOption[*entity.DbBackup] {
	return func(scheduler *Scheduler[*entity.DbBackup]) {
		scheduler.RunTask = func(ctx context.Context, task *entity.DbBackup) error {
			instance := new(entity.DbInstance)
			if err := repositories.Instance.GetById(instance, task.DbInstanceId); err != nil {
				return err
			}
			instance.PwdDecrypt()
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
			binlogInfo, err := NewDbInstanceSvc(instance, repositories).Backup(ctx, history)
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
			if err := binlogSvc.AddTaskIfNotExists(ctx, entity.NewDbBinlog(history)); err != nil {
				return err
			}
			return nil
		}
	}
}

var (
	backupResult = map[entity.TaskStatus]string{
		entity.TaskDelay:    "等待备份数据库",
		entity.TaskReady:    "准备备份数据库",
		entity.TaskReserved: "数据库备份中",
		entity.TaskSuccess:  "数据库备份成功",
		entity.TaskFailed:   "数据库备份失败",
	}
)

func withUpdateBackupStatus(repositories *repository.Repositories) SchedulerOption[*entity.DbBackup] {
	return func(scheduler *Scheduler[*entity.DbBackup]) {
		scheduler.UpdateTaskStatus = func(ctx context.Context, status entity.TaskStatus, lastErr error, task *entity.DbBackup) error {
			task.Finished = !task.Repeated && status == entity.TaskSuccess
			task.LastStatus = status
			var result = backupResult[status]
			if lastErr != nil {
				result = fmt.Sprintf("%v: %v", backupResult[status], lastErr)
			}
			task.LastResult = stringx.TruncateStr(result, entity.LastResultSize)
			task.LastTime = time.Now()
			return repositories.Backup.UpdateTaskStatus(ctx, task)
		}
	}
}

func NewDbBackupSvc(repositories *repository.Repositories, binlogSvc service.DbBinlogSvc) (service.DbBackupSvc, error) {
	scheduler, err := NewScheduler[*entity.DbBackup](
		withRunBackupTask(repositories, binlogSvc),
		withUpdateBackupStatus(repositories))
	if err != nil {
		return nil, err
	}
	svc := &DbBackupSvcImpl{
		repo:         repositories.Backup,
		instanceRepo: repositories.Instance,
		scheduler:    scheduler,
		binlogSvc:    binlogSvc,
	}
	err = svc.loadTasks(context.Background())
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func (svc *DbBackupSvcImpl) loadTasks(ctx context.Context) error {
	tasks := make([]*entity.DbBackup, 0, 64)
	cond := map[string]any{
		"Enabled":  true,
		"Finished": false,
	}
	if err := svc.repo.ListByCond(cond, &tasks); err != nil {
		return err
	}
	for _, task := range tasks {
		svc.scheduler.PushTask(ctx, task)
	}
	return nil
}

func (svc *DbBackupSvcImpl) AddTask(ctx context.Context, tasks ...*entity.DbBackup) error {
	for _, task := range tasks {
		if err := svc.repo.AddTask(ctx, task); err != nil {
			return err
		}
		svc.scheduler.PushTask(ctx, task)
	}
	return nil
}

func (svc *DbBackupSvcImpl) UpdateTask(ctx context.Context, task *entity.DbBackup) error {
	if err := svc.repo.UpdateById(ctx, task); err != nil {
		return err
	}
	svc.scheduler.UpdateTask(ctx, task)
	return nil
}

func (svc *DbBackupSvcImpl) DeleteTask(ctx context.Context, taskId uint64) error {
	// todo: 删除数据库备份历史文件
	task := new(entity.DbBackup)
	if err := svc.repo.GetById(task, taskId); err != nil {
		return err
	}
	if err := svc.binlogSvc.DeleteTask(ctx, task.DbInstanceId); err != nil {
		return err
	}
	if err := svc.repo.DeleteById(ctx, taskId); err != nil {
		return err
	}
	svc.scheduler.RemoveTask(taskId)
	return nil
}

func (svc *DbBackupSvcImpl) EnableTask(ctx context.Context, taskId uint64) error {
	if err := svc.repo.UpdateEnabled(ctx, taskId, true); err != nil {
		return err
	}
	task := new(entity.DbBackup)
	if err := svc.repo.GetById(task, taskId); err != nil {
		return err
	}
	svc.scheduler.UpdateTask(ctx, task)
	return nil
}

func (svc *DbBackupSvcImpl) DisableTask(ctx context.Context, taskId uint64) error {
	if err := svc.repo.UpdateEnabled(ctx, taskId, false); err != nil {
		return err
	}
	task := new(entity.DbBackup)
	if err := svc.repo.GetById(task, taskId); err != nil {
		return err
	}
	svc.scheduler.RemoveTask(taskId)
	return nil
}

// GetPageList 分页获取数据库备份任务
func (svc *DbBackupSvcImpl) GetPageList(condition *entity.DbBackupQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return svc.repo.GetDbBackupList(condition, pageParam, toEntity, orderBy...)
}
