package sync

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/taskx"
)

func (app *DataSyncAppImpl) InitCronJob() {
	ctx := contextx.WithTraceId(context.Background())

	defer func() {
		if err := recover(); err != nil {
			logx.ErrorTraceContext(ctx, "the data synchronization task failed to initialize", err)
		}
	}()

	// 复位进程重启前遗留的「运行中」状态：失败必须留日志，否则任务永远停在运行中而无人知道原因
	if err := app.UpdateByCond(ctx, &entity.DataSyncTask{RunningState: entity.DataSyncTaskRunStateReady}, &entity.DataSyncTask{RunningState: entity.DataSyncTaskRunStateRunning}); err != nil {
		logx.ErrorfContext(ctx, "failed to reset running data sync tasks on startup: %s", err.Error())
	}

	// 把进程重启前遗留的「执行中」同步日志置为失败：执行协程随进程退出，无人收尾的日志会永远停留在「执行中」
	if err := app.ResetStaleRunningSyncLogs(ctx); err != nil {
		logx.ErrorfContext(ctx, "failed to reset running sync logs on startup: %s", err.Error())
	}

	if err := app.CursorByCond(&entity.DataSyncTaskQuery{Status: entity.DataSyncTaskStatusEnable}, func(dst *entity.DataSyncTask) error {
		app.addCronJob(ctx, dst)
		return nil
	}); err != nil {
		logx.ErrorTraceContext(ctx, "the db data sync task failed to initialize: %v", err)
	}
}

// syncTaskScheduled 同步任务是否会被注册为定时任务：非启用态只解绑。
// 保存前的表达式校验与 addCronJob 的绑定判定共用此函数，避免两处口径漂移。
func syncTaskScheduled(taskEntity *entity.DataSyncTask) bool {
	return taskEntity.Status == entity.DataSyncTaskStatusEnable
}

func (app *DataSyncAppImpl) addCronJob(ctx context.Context, taskEntity *entity.DataSyncTask) {
	key := taskEntity.TaskKey
	if !syncTaskScheduled(taskEntity) {
		taskx.UnbindCronTask(key)
		return
	}

	taskId := taskEntity.Id
	logx.InfofContext(ctx, "start add the data sync task job: %s, cron[%s]", taskEntity.TaskName, taskEntity.TaskCron)
	if err := taskx.BindCronTask(key, taskEntity.TaskCron, true, func() {
		if err := app.Run(context.Background(), taskId); err != nil {
			logx.ErrorfContext(ctx, "the data sync task failed to execute at a scheduled time: %s", err.Error())
		}
	}); err != nil {
		logx.ErrorTraceContext(ctx, "add db data sync job failed", err)
	}
}
