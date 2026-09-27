package transfer

import (
	"context"
	"fmt"
	"time"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

// 迁移执行日志：任务运行日志的创建、追加、实时缓存、收尾落库。

// transferLogEndColumns 日志收尾必须写入的列。
// 需显式指定：GORM以结构体更新时会跳过零值字段，而失败状态、0行、0耗时等收尾值恰为零值，
// 不指定列会导致这些值丢失（如状态停留在「执行中」）。
var transferLogEndColumns = []string{"status", "err_text", "duration_ms", "total_rows", "table_count", "run_log"}

func (app *DbTransferAppImpl) CreateLog(ctx context.Context, taskId uint64, purpose entity.DbTransferLogPurpose) (uint64, error) {
	task, err := app.GetById(taskId)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	log := &entity.DbTransferLog{
		TaskId:     taskId,
		CreateTime: &now,
		Mode:       task.Mode,
		Purpose:    purpose,
		Status:     entity.DbTransferLogStatusRunning,
	}
	if err := app.transferLogRepo.Insert(ctx, log); err != nil {
		return 0, err
	}
	return log.Id, nil
}

// transferPurposeOfMode 由迁移模式推导「迁移执行」的日志用途：文件模式为导出文件，否则为库到库迁移。
// 校验用途与此无关，由 Verify 直接指定 DbTransferLogPurposeVerify。
func transferPurposeOfMode(mode entity.TransferMode) entity.DbTransferLogPurpose {
	if mode == entity.DbTransferTaskModeFile {
		return entity.DbTransferLogPurposeExport
	}
	return entity.DbTransferLogPurposeTransfer
}

// GetLogPageList 获取任务执行日志列表（不含运行日志内容，列表页只需状态与指标）
func (app *DbTransferAppImpl) GetLogPageList(condition *entity.DbTransferLogQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferLog], error) {
	res, err := app.transferLogRepo.GetPageList(condition, orderBy...)
	if err != nil {
		return nil, err
	}
	// 执行中的任务指标只写缓存未落库，故查询第一页（最新一条）时用缓存的实时值覆盖DB中可能过时的数据。
	// 只读缓存不回查 DB：已结束任务的指标已落库无需合并，而回查会把 run_log 大文本整行读出
	if res != nil && len(res.List) > 0 && condition.PageNum <= 1 {
		latestLog := res.List[0]
		cachedLog := app.getTransferLogCache(latestLog.Id)
		if cachedLog != nil {
			if cachedLog.TotalRows > 0 {
				latestLog.TotalRows = cachedLog.TotalRows
			}
			if cachedLog.TableCount > 0 {
				latestLog.TableCount = cachedLog.TableCount
			}
			if cachedLog.DurationMs > 0 {
				latestLog.DurationMs = cachedLog.DurationMs
			}
		}
	}
	return res, nil
}

// GetLogWithRunLog 按日志 id 获取单条执行日志的运行日志与状态。
// 运行日志内容不在列表中返回，由该入口按需获取：执行中的日志命中缓存则取缓存，保证实时视图看到最新内容。
func (app *DbTransferAppImpl) GetLogWithRunLog(logId uint64) (*entity.DbTransferLog, error) {
	log := app.getTransferLog(logId)
	if log == nil {
		return nil, errorx.NewBizf("transfer log [%d] not found", logId)
	}
	return log, nil
}

// appendRunLine 向日志对象追加一行带时间戳的运行记录（纯内存操作，是否回写缓存由调用方决定）
func appendRunLine(log *entity.DbTransferLog, msg string) {
	ts := time.Now().Format("15:04:05.000")
	log.RunLog += fmt.Sprintf("[%s] %s\n", ts, msg)
}

// Log 向运行日志追加一行带时间戳的记录，并同步更新缓存以供前端实时查看。
// 仅写入 RunLog（用户可见），不重复写入系统日志，与 sync 模块 appendRunLog 行为一致。
func (app *DbTransferAppImpl) Log(ctx context.Context, logId uint64, msg string) {
	log := app.getTransferLog(logId)
	if log != nil {
		appendRunLine(log, msg)
		app.setTransferLog(logId, log)
	}
}

// runEndTitles 运行日志收尾摘要标题，按本次执行的用途区分文案。
// 校验与迁移共用收尾逻辑，若统一用迁移措辞，会让人误以为一次只读校验执行了数据迁移
type runEndTitles struct {
	success    string // 成功收尾标题
	fail       string // 失败收尾标题
	tableCount string // 表数统计行的前缀（"迁移表数" vs "校验表数"）
	rowCount   string // 行数统计行的前缀（"迁移行数" vs "校验行数"）
}

var (
	// 数据迁移（库→库、库→文件）收尾标题
	transferEndTitles = runEndTitles{success: "迁移执行完成", fail: "迁移执行失败", tableCount: "迁移表数", rowCount: "迁移行数"}

	// 数据校验收尾标题：校验全程只读比对，不写入任何数据；统计行措辞随之改为"校验"
	verifyEndTitles = runEndTitles{success: "数据校验完成", fail: "数据校验未通过", tableCount: "校验表数", rowCount: "校验行数"}
)

// EndTransfer 迁移执行收尾
func (app *DbTransferAppImpl) EndTransfer(ctx context.Context, logId uint64, taskId uint64, msg string, err error, extra map[string]any) {
	app.endRun(ctx, logId, taskId, msg, err, transferEndTitles)
}

// EndVerify 数据校验执行收尾
func (app *DbTransferAppImpl) EndVerify(ctx context.Context, logId uint64, taskId uint64, msg string, err error) {
	app.endRun(ctx, logId, taskId, msg, err, verifyEndTitles)
}

func (app *DbTransferAppImpl) endRun(ctx context.Context, logId uint64, taskId uint64, msg string, err error, titles runEndTitles) {
	// runGuard.Release 延迟到任务状态更新之后，防止窗口期内另一个 Run 获取守卫并启动

	transferState := entity.DbTransferTaskRunStateSuccess
	logStatus := entity.DbTransferLogStatusSuccess
	if err != nil {
		msg = fmt.Sprintf("%s: %s", msg, err.Error())
		logx.ErrorContext(ctx, msg)
		transferState = entity.DbTransferTaskRunStateFail
		logStatus = entity.DbTransferLogStatusFail
	} else {
		logx.InfoContext(ctx, msg)
	}

	// 追加最终消息到运行日志
	log := app.getTransferLog(logId)
	if log != nil {
		// 计算耗时（防止负数：CreateTime 可能因时区/缓存问题晚于当前时间）
		if log.CreateTime != nil {
			ms := time.Since(*log.CreateTime).Milliseconds()
			if ms > 0 {
				log.DurationMs = ms
			}
		}

		// 输出执行摘要：必须追加到本对象而非经app.Log（其取的是缓存副本，与本对象互不影响），
		// 否则收尾摘要不会随下面的更新落库，日志只剩执行过程而看不到最终结果
		appendRunLine(log, "========================================")
		if err != nil {
			appendRunLine(log, titles.fail)
			appendRunLine(log, fmt.Sprintf("错误信息: %s", err.Error()))
		} else {
			appendRunLine(log, titles.success)
		}
		appendRunLine(log, "========================================")
		if log.TableCount > 0 {
			appendRunLine(log, fmt.Sprintf("%s: %d", titles.tableCount, log.TableCount))
		}
		if log.TotalRows > 0 {
			appendRunLine(log, fmt.Sprintf("%s: %d", titles.rowCount, log.TotalRows))
		}
		if log.DurationMs > 0 {
			appendRunLine(log, fmt.Sprintf("总耗时: %d ms", log.DurationMs))
			if log.TotalRows > 0 {
				throughput := log.TotalRows * 1000 / log.DurationMs
				appendRunLine(log, fmt.Sprintf("吞吐量: %d 行/秒", throughput))
			}
		}
		appendRunLine(log, "========================================")

		log.Status = logStatus
		log.ErrText = ""
		if err != nil {
			log.ErrText = err.Error()
		}
		// 回写缓存，保证前端实时视图与落库内容一致
		app.setTransferLog(logId, log)
		// 落库
		if err := app.transferLogRepo.UpdateById(context.Background(), log, transferLogEndColumns...); err != nil {
			logx.ErrorfContext(context.Background(), "failed to save transfer log [%d]: %s", log.Id, err.Error())
		}
	}

	// 修改任务状态
	task := new(entity.DbTransferTask)
	task.Id = taskId
	task.RunningState = transferState
	if err := app.UpdateById(context.Background(), task); err != nil {
		logx.ErrorfContext(context.Background(), "failed to update transfer task [%d] running state: %s", taskId, err.Error())
	}

	// 状态更新完成后再释放守卫，防止窗口期内另一个 Run 获取守卫并启动
	app.runGuard.Release(taskId)
	// 收敛跨实例停止标记：即使上面 Release 是本实例持有走的清理，Redis 里的停止标记仍需显式清；
	// 若标记设置方是本实例，这里等价幂等清理。5min TTL 是最终兜底。
	app.runGuard.ClearStopRequest(taskId)
}

// ResetStaleRunningLogs 启动收尾：把仍处于「执行中」的日志置为失败。
// 进程重启或异常退出会带走执行协程，这些日志无人收尾，不处理将永远停留在「执行中」。
func (app *DbTransferAppImpl) ResetStaleRunningLogs(ctx context.Context) error {
	return app.transferLogRepo.UpdateByCond(ctx,
		&entity.DbTransferLog{Status: entity.DbTransferLogStatusFail, ErrText: "transfer interrupted: server restarted"},
		&entity.DbTransferLog{Status: entity.DbTransferLogStatusRunning})
}

// setVerifyMetrics 回填校验记录的表数/行数指标到缓存（收尾时随 transferLogEndColumns 落库）。
// 表数=校验涉及的表数量，行数=各表源侧行数之和；校验只读比对，行数为「涉及行数」而非「迁移行数」。
func (app *DbTransferAppImpl) setVerifyMetrics(logId uint64, report *TransferVerifyReport) {
	if report == nil {
		return
	}
	log := app.getTransferLog(logId)
	if log == nil {
		return
	}
	var totalRows int64
	for _, res := range report.Results {
		totalRows += res.SrcCount
	}
	log.TableCount = len(report.Results)
	log.TotalRows = totalRows
	app.setTransferLog(logId, log)
}

// getTransferLogCache 只读迁移日志的运行中缓存（不回落 DB）：缓存里是执行中的日志，其指标为实时值
func (app *DbTransferAppImpl) getTransferLogCache(logId uint64) *entity.DbTransferLog {
	if logId == 0 {
		return nil
	}
	log := new(entity.DbTransferLog)
	if cache.Get(getTransferLogKey(logId), log) {
		return log
	}
	return nil
}

// getTransferLog 获取迁移日志（优先从内存缓存）
func (app *DbTransferAppImpl) getTransferLog(logId uint64) *entity.DbTransferLog {
	if logId == 0 {
		return nil // logId=0 表示无日志上下文（如单测直接调用），跳过 DB 查询
	}
	log := new(entity.DbTransferLog)
	if cache.Get(getTransferLogKey(logId), log) {
		return log
	}
	if app.transferLogRepo == nil {
		return nil // 仓储未注入时（测试场景）安全返回
	}
	log, err := app.transferLogRepo.GetById(logId)
	if err != nil {
		return nil
	}
	if log != nil {
		app.setTransferLog(logId, log)
	}
	return log
}

// setTransferLog 设置迁移日志缓存
func (app *DbTransferAppImpl) setTransferLog(logId uint64, log *entity.DbTransferLog) {
	cache.Set(getTransferLogKey(logId), log, time.Duration(TransferLogCacheTTLSeconds)*time.Second)
}

func getTransferLogKey(logId uint64) string {
	return fmt.Sprintf("mayfly:db_transfer_log:%d", logId)
}
