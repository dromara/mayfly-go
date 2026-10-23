package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mayfly-go/internal/db/domain/entity"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

// 数据同步任务的运行日志生命周期：收尾落库、实时缓存读写。

func (app *DataSyncAppImpl) endRunning(taskEntity *entity.DataSyncTask, log *entity.DataSyncLog) {
	logx.InfoContext(context.Background(), log.ErrText)

	if log.Status == entity.DataSyncTaskStateRunning {
		log.Status = entity.DataSyncTaskStateFail
	}
	// 任务级状态（RecentState/UpdFieldVal/RunningState）只在仍是本次锁持有者时写：
	// 锁 TTL 自然过期后被新实例接管的场景下，旧收尾不得覆盖新执行的 RunningState=Running；
	// 旧收尾也不能推水位（新实例自己会写一份 UpdFieldVal）。
	if app.runGuard.IsCurrentRun(taskEntity.Id, log.RunId) {
		task := new(entity.DataSyncTask)
		task.Id = taskEntity.Id
		task.RecentState = log.Status
		task.UpdFieldVal = taskEntity.UpdFieldVal
		task.RunningState = entity.DataSyncTaskRunStateReady
		if err := app.UpdateById(context.Background(), task); err != nil {
			logx.ErrorfContext(context.Background(), "failed to update sync task [%d] state after execution: %s", taskEntity.Id, err.Error())
		}
	} else {
		logx.WarnfContext(context.Background(), "sync task [%d] was preempted, skipping task state update for run %s", taskEntity.Id, log.RunId)
	}
	// 失败原因必须能在「运行日志」弹窗里看到：GetLogWithRunLog 只回传 RunLog 字段，
	// 若错误只落在 ErrText 列，用户点"运行日志"看到的永远是断在半截的进度行 + 状态"失败"，
	// 无从判断真因（本次线上问题：目标表 0 列 → doDataSync 在 [5/6]-[6/6] 之间 return error，
	// ErrText 有内容但 RunLog 没写，弹窗一片祥和）。
	// 与 transfer 侧 endTransfer 用 appendRunLine 追加摘要行的做法对齐。
	if log.Status == entity.DataSyncTaskStateFail && log.ErrText != "" && !strings.Contains(log.RunLog, log.ErrText) {
		// ErrText 已由 runSync 侧的 i18n 模板带上"执行失败: "前缀，这里不再重复加前缀，
		// 只以分隔线包起，避免弹窗里出现"同步执行失败: 执行失败: xxx"的双重措辞。
		app.appendRunLog(log, "========================================")
		app.appendRunLog(log, "%s", log.ErrText)
		app.appendRunLog(log, "========================================")
	}
	// 失败要通知任务创建者：cron 触发的同步没人盯界面，只写日志等于静默失效。
	//
	// 只在本次执行仍是锁持有者时通知：被新实例接管的旧收尾属于「已被取代的那一轮」，
	// 为它发通知会把同一任务的失败刷屏，且真实结论应由接管方的收尾给出
	if log.Status == entity.DataSyncTaskStateFail && app.runGuard.IsCurrentRun(taskEntity.Id, log.RunId) && taskEntity.CreatorId != 0 {
		app.notifyTaskRunFail(taskEntity, log)
	}

	// 日志行属于本次执行，无论是否被接管都要落终态（防止实时缓存过期后前端看不到已完成日志）
	app.saveLog(log)
	app.delRunningSyncLog(log.Id)
	// 仅当本次 runId 仍是锁的当前持有者时才释放，避免锁 TTL 自然过期后被新实例接管时旧收尾误删新锁
	app.runGuard.ReleaseWithRunId(taskEntity.Id, log.RunId)
	// 收敛停止标记：RequestStop 若来自本实例已通过 Release 走通，若来自他实例则标记由这里清。
	// 即使这里失败，标记本身 5min TTL 会自愈，不会永久影响下一轮 Acquire。
	app.runGuard.ClearStopRequest(taskEntity.Id)
}

func (app *DataSyncAppImpl) saveLog(log *entity.DataSyncLog) {
	if app.dbDataSyncLogRepo == nil {
		return
	}
	if err := app.dbDataSyncLogRepo.Save(context.Background(), log); err != nil {
		logx.ErrorfContext(context.Background(), "failed to save sync log [%d]: %s", log.Id, err.Error())
	}
}

// ResetStaleRunningSyncLogs 启动收尾：把确认失活的「执行中」同步日志置为失败。
//
// 不能无条件刷新所有 Running 日志——多实例共享任务库时，其他实例可能正在执行；
// 也不能仅看 “任务级锁是否被任一实例持有”——同一任务上旧一次崩溃遗留的 Running 日志与
// 新一次正常执行的 Running 日志共享同一个 taskId，仅按 taskId 判断会把两者都当作活跃。
//
// 日志行自带的 runId 精确到“哪一次执行”：当前锁持有的 runId 与新写入一致时跳过（真活跃），
// 不一致或无锁即确认失活。旧行 run_id 列为空（升级前数据）无法匹配任何锁，一律收尾。
func (app *DataSyncAppImpl) ResetStaleRunningSyncLogs(ctx context.Context) error {
	runningLogs, err := app.dbDataSyncLogRepo.SelectByCond(&entity.DataSyncLog{Status: entity.DataSyncTaskStateRunning}, "id", "task_id", "run_id")
	if err != nil {
		return fmt.Errorf("list running sync logs: %w", err)
	}
	for _, log := range runningLogs {
		if log.RunId != "" && app.runGuard.IsCurrentRun(log.TaskId, log.RunId) {
			// 日志属于当前锁持有者，仍在活跃执行，不收尾
			continue
		}
		upd := &entity.DataSyncLog{}
		upd.Id = log.Id
		upd.Status = entity.DataSyncTaskStateFail
		upd.ErrText = "sync interrupted: server restarted"
		// 显式传列：避免后续新增零值枚举时 GORM 结构体更新默默丢字段
		if err := app.dbDataSyncLogRepo.UpdateById(ctx, upd, "status", "err_text"); err != nil {
			logx.ErrorfContext(ctx, "failed to reset stale running sync log [%d]: %s", log.Id, err.Error())
		}
	}
	return nil
}

// runningSyncLogTTL 运行中日志缓存 TTL
const runningSyncLogTTL = time.Hour

// getRunningSyncLog 获取运行中同步任务的缓存日志（供 GetLogPageList 实时读取）
func (app *DataSyncAppImpl) getRunningSyncLog(logId uint64) *entity.DataSyncLog {
	var log entity.DataSyncLog
	if cache.Get(getRunningSyncLogKey(logId), &log) {
		return &log
	}
	return nil
}

// setRunningSyncLog 缓存运行中同步任务的日志（每次 appendRunLog 时更新）
func (app *DataSyncAppImpl) setRunningSyncLog(logId uint64, log *entity.DataSyncLog) {
	cache.Set(getRunningSyncLogKey(logId), log, runningSyncLogTTL)
}

// delRunningSyncLog 删除运行中同步任务的缓存日志
func (app *DataSyncAppImpl) delRunningSyncLog(logId uint64) {
	cache.Del(getRunningSyncLogKey(logId))
}

func getRunningSyncLogKey(logId uint64) string {
	return fmt.Sprintf("mayfly:data_sync_log:%d", logId)
}

// notifyTaskRunFail 通知同步任务创建者本次执行失败。
//
// 收件人取创建者而非 ctx 登录账号：cron 执行上下文没有登录用户。
// 通知本身失败只记日志，不影响收尾流程
func (app *DataSyncAppImpl) notifyTaskRunFail(taskEntity *entity.DataSyncTask, log *entity.DataSyncLog) {
	errText := log.ErrText
	if errText == "" {
		errText = "task failed without error text"
	}
	global.EventBus.Publish(context.Background(), event.EventTopicMsgTmplSend, msgdto.DbTaskRunFailEvent(taskEntity.TaskName, errText, taskEntity.CreatorId))
}

// GetLogPageList 获取任务执行日志列表（不含运行日志内容，列表页只需状态与指标）
func (app *DataSyncAppImpl) GetLogPageList(condition *entity.DataSyncLogQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncLog], error) {
	return app.dbDataSyncLogRepo.GetPageList(condition, orderBy...)
}

// GetLogWithRunLog 按日志 id 获取单条执行日志的运行日志与状态。
// 运行日志内容不在列表中返回，由该入口按需获取：执行中的日志只存于实时缓存，故优先读缓存
func (app *DataSyncAppImpl) GetLogWithRunLog(logId uint64) (*entity.DataSyncLog, error) {
	if log := app.getRunningSyncLog(logId); log != nil {
		return log, nil
	}
	log, err := app.dbDataSyncLogRepo.GetById(logId)
	if err != nil {
		return nil, err
	}
	if log == nil {
		return nil, errorx.NewBizf("data sync log [%d] not found", logId)
	}
	return log, nil
}
