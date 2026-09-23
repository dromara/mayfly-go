package transfer

import (
	"context"
	"time"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/taskx"
)

// 迁移任务的定时调度与运行态控制。

func (app *DbTransferAppImpl) Stop(ctx context.Context, taskId uint64) error {
	task, err := app.GetById(taskId)
	if err != nil {
		return errorx.NewBiz("task not found")
	}

	if task.RunningState != entity.DbTransferTaskRunStateRunning {
		return errorx.NewBiz("the task is not being executed")
	}
	task.RunningState = entity.DbTransferTaskRunStateStop
	if err = app.UpdateById(ctx, task); err != nil {
		return err
	}

	app.runGuard.Release(taskId)
	return nil
}

func (d *DbTransferAppImpl) TimerDeleteTransferFile() {
	ctx := contextx.WithTraceId(context.Background())
	logx.DebugContext(ctx, "start deleting transfer files periodically...")
	scheduler.AddFun("@every 100m", func() {
		defer gox.Recover()
		dts, err := d.ListByCond(model.NewCond().Eq("mode", entity.DbTransferTaskModeFile).Ge("file_save_days", 1))
		if err != nil {
			logx.ErrorfContext(ctx, "the task to periodically get database transfer to file failed: %s", err.Error())
			return
		}
		for _, dt := range dts {
			needDelFiles, err := d.transferFileApp.ListByCond(model.NewCond().Eq("task_id", dt.Id).Le("create_time", time.Now().AddDate(0, 0, -dt.FileSaveDays)))
			if err != nil {
				logx.ErrorfContext(ctx, "failed to obtain the transfer file periodically: %s", err.Error())
				continue
			}
			for _, nf := range needDelFiles {
				if err := d.transferFileApp.Delete(context.Background(), nf.Id); err != nil {
					logx.ErrorfContext(ctx, "failed to delete transfer files periodically: %s", err.Error())
				}
			}
		}
	})
}

func (app *DbTransferAppImpl) addCronJob(ctx context.Context, taskEntity *entity.DbTransferTask) {
	key := taskEntity.TaskKey
	enabled := taskEntity.Status == entity.DbTransferTaskStatusEnable && taskEntity.CronEnabled == entity.DbTransferTaskCronEnabled
	if !enabled {
		taskx.UnbindCronTask(key)
		return
	}

	taskId := taskEntity.Id
	// 统一内核：移除旧绑定后按状态注册新任务
	if err := taskx.BindCronTask(key, taskEntity.Cron, true, func() {
		logx.InfofContext(ctx, "start the transfer task: %d", taskId)
		if _, err := app.Run(ctx, taskId); err != nil {
			logx.WarnContext(ctx, err.Error())
		}
	}); err != nil {
		logx.ErrorTraceContext(ctx, "add db transfer cron job failed", err)
	}
}

// IsRunning 判断任务是否执行中（供api层查询展示）
func (app *DbTransferAppImpl) IsRunning(taskId uint64) bool {
	return app.runGuard.IsRunning(taskId)
}
