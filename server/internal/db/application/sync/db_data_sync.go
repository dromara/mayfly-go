package sync

import (
	"context"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/taskx"
	"mayfly-go/pkg/utils/stringx"
	"regexp"
)

type DbApp interface {
	GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error)
}

type DataSyncTask interface {
	base.App[*entity.DataSyncTask]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DataSyncTaskQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncTask], error)

	Save(ctx context.Context, instanceEntity *entity.DataSyncTask) error

	Delete(ctx context.Context, id uint64) error

	InitCronJob()

	Run(ctx context.Context, id uint64) error

	StopTask(ctx context.Context, id uint64) error

	GetLogPageList(condition *entity.DataSyncLogQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncLog], error)

	// GetLogWithRunLog 按日志 id 获取单条执行日志（含运行日志内容）
	GetLogWithRunLog(logId uint64) (*entity.DataSyncLog, error)
}

var _ (DataSyncTask) = (*DataSyncAppImpl)(nil)

type DataSyncAppImpl struct {
	base.AppImpl[*entity.DataSyncTask, repository.DataSyncTask]

	dbDataSyncLogRepo repository.DataSyncLog `inject:"T"`

	dbApp DbApp `inject:"T"`

	// runGuard 运行态原子守卫：把 IsRunning 的检查与置位合并为一次原子操作，消除检查-标记间的并发竞态
	runGuard taskx.RunGuard[uint64]
}

var (
	whereReg = regexp.MustCompile(`(?i)where`)
)

func (app *DataSyncAppImpl) GetPageList(condition *entity.DataSyncTaskQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncTask], error) {
	return app.GetRepo().GetPageList(condition, orderBy...)
}

func (app *DataSyncAppImpl) Save(ctx context.Context, taskEntity *entity.DataSyncTask) error {
	// 会被调度的任务先校验 cron：绑定失败只写日志，不拦下就是「保存成功但永不执行」；
	// 放在建连之前，避免为一个注定不合法的表达式去开源库连接
	if syncTaskScheduled(taskEntity) {
		if err := scheduler.ValidateSpec(taskEntity.TaskCron); err != nil {
			return errorx.NewBizI(ctx, imsg.ErrTaskCronInvalid, "cron", taskEntity.TaskCron, "reason", err.Error())
		}
	}

	// 保存时校验DataSQL与增量字段合法性（运行时Run内还会再校验，双重拦截）；
	// P1：同一个 srcConn 上顺带做一次增量字段索引存在性探测，无索引直接拒保存，不重复开连接。
	srcConn, err := app.getSrcConn(taskEntity)
	if err != nil {
		return err
	}
	if err := validateDataSyncSQL(srcConn.GetDialect(), taskEntity); err != nil {
		return err
	}
	if err := ValidateIncrementalFieldIndex(ctx, srcConn, taskEntity); err != nil {
		return err
	}

	if taskEntity.Id == 0 {
		// 新建时生成key
		taskEntity.TaskKey = stringx.RandUUID()
		err = app.Insert(ctx, taskEntity)
	} else {
		if taskEntity.TaskKey == "" {
			task, err := app.GetById(taskEntity.Id)
			if err != nil {
				return errorx.NewBiz("db sync task not found")
			}
			taskEntity.TaskKey = task.TaskKey
		}
		err = app.UpdateById(ctx, taskEntity)
	}
	if err != nil {
		return err
	}

	app.addCronJob(ctx, taskEntity)

	// 双向同步 - 确保反向任务存在
	if taskEntity.BiDirEnabled && taskEntity.Status == entity.DataSyncTaskStatusEnable {
		biDirMgr := NewBidirectionalSyncManager(app)
		if err := biDirMgr.EnsureReverseTask(ctx, taskEntity); err != nil {
			logx.WarnfContext(ctx, "bidirectional sync: failed to ensure reverse task: %s", err.Error())
		}
	}

	return nil
}

func (app *DataSyncAppImpl) Delete(ctx context.Context, id uint64) error {
	task, err := app.GetById(id)
	if err != nil {
		return errorx.NewBiz("sync task not found")
	}
	scheduler.RemoveByKey(task.TaskKey)
	app.runGuard.Release(id)

	// 删除时同步清理关联的反向任务（防止循环引用导致无限递归）
	if task.BiDirEnabled && task.ReverseTaskId > 0 && task.ReverseTaskId != id {
		if err := app.deleteReverseTask(ctx, task.ReverseTaskId, id); err != nil {
			logx.WarnfContext(ctx, "failed to delete reverse sync task [%d]: %s", task.ReverseTaskId, err.Error())
		}
	}

	return app.DeleteById(ctx, id)
}

// deleteReverseTask 删除反向任务，带循环引用保护。
func (app *DataSyncAppImpl) deleteReverseTask(ctx context.Context, reverseId uint64, originId uint64) error {
	reverseTask, err := app.GetById(reverseId)
	if err != nil {
		return errorx.NewBiz("reverse sync task not found")
	}
	scheduler.RemoveByKey(reverseTask.TaskKey)
	app.runGuard.Release(reverseId)

	// 反向任务的反向如果指向原始任务，不再递归（防止 A→B→A 循环）
	if reverseTask.BiDirEnabled && reverseTask.ReverseTaskId > 0 && reverseTask.ReverseTaskId != originId {
		if err := app.deleteReverseTask(ctx, reverseTask.ReverseTaskId, reverseId); err != nil {
			logx.WarnfContext(ctx, "failed to delete nested reverse sync task [%d]: %s", reverseTask.ReverseTaskId, err.Error())
		}
	}

	return app.DeleteById(ctx, reverseId)
}

func GetDataSyncTaskApp() DataSyncTask {
	return ioc.Get[DataSyncTask]()
}
