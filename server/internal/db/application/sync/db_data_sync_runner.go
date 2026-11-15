package sync

import (
	"cmp"
	"context"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"strings"
	"time"
)

func (app *DataSyncAppImpl) Run(ctx context.Context, id uint64) error {
	// 先原子占位再查任务：消除 IsRunning→Acquire 之间的 TOCTOU 竞态窗口，
	// 与 transfer 模块 Run 保持一致的占位优先策略。AcquireWithRunId 额外返回本次执行的 runId，
	// 同一条链路上的日志行、启动收尾、停止检查、水位推进均围绕它判定归属（“同一次执行”）。
	runId, ok := app.runGuard.AcquireWithRunId(id)
	if !ok {
		logx.WarnfContext(ctx, "[%d] the db sync task is running...", id)
		return errorx.NewBiz("the task is running")
	}

	task, err := app.GetById(id)
	if err != nil {
		app.runGuard.ReleaseWithRunId(id, runId)
		return errorx.NewBiz("task not found")
	}

	logx.InfofContext(ctx, "start the data sync task: %s => %s", task.TaskName, task.TaskKey)

	updateStateTask := &entity.DataSyncTask{
		RunningState: entity.DataSyncTaskRunStateRunning,
	}
	updateStateTask.Id = id
	if err := app.UpdateById(ctx, updateStateTask); err != nil {
		app.runGuard.ReleaseWithRunId(id, runId)
		return errorx.NewBizf("failed to update task running state: %s", err.Error())
	}

	// 提前创建日志记录并落库，使前端轮询可在执行期间查看实时日志；
	// syncLog.RunId = 本次锁所有权值，下游所有写入/收尾/启动收尾都拿它定位“同一次执行”。
	now := time.Now()
	syncLog := &entity.DataSyncLog{
		TaskId:     task.Id,
		RunId:      runId,
		CreateTime: &now,
		Status:     entity.DataSyncTaskStateRunning, // 初始「执行中」，完成后由 endRunning 更新为成功/失败；进程异常退出遗留的执行中日志由 ResetStaleRunningSyncLogs 启动收尾
	}
	if err := app.dbDataSyncLogRepo.Save(ctx, syncLog); err != nil {
		app.runGuard.ReleaseWithRunId(id, runId)
		return errorx.NewBizf("failed to create sync log: %s", err.Error())
	}
	// 缓存日志供 GetLogPageList 实时读取 RunLog
	app.setRunningSyncLog(syncLog.Id, syncLog)

	gox.Go(func() {
		// 任务在后台异步执行，请求ctx随响应结束而取消；
		// 需脱离请求取消信号但保留traceId等链路信息，否则后续日志与同步执行均报ctx canceled
		ctx = context.WithoutCancel(ctx)
		metrics := NewSyncMetrics()
		metrics.StartTime = now.UnixMilli()

		defer func() {
			metrics.EndTime = time.Now().UnixMilli()
			metrics.ToSyncLog(syncLog)
			app.endRunning(task, syncLog)
		}()
		defer gox.Recover(func(err error) {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			syncLog.Status = entity.DataSyncTaskStateFail
		})

		// 获取源/目标库连接（源库连接同时用于运行时校验与增量字段类型探测）
		srcConn, err := app.dbApp.GetDbConn(ctx, uint64(task.SrcDbId), task.SrcDbName)
		if err != nil {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			return
		}
		targetConn, err := app.dbApp.GetDbConn(ctx, uint64(task.TargetDbId), task.TargetDbName)
		if err != nil {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			return
		}
		_ = app.runSync(ctx, task, srcConn, targetConn, syncLog, metrics)
	})

	return nil
}

// runSync 在已给定源/目标连接下执行一轮同步：运行时校验 → 构建同步查询 → 写入/删除对账 或 数据校验。
// 供异步 Run 与导出入口 SyncTask 复用；负责落定 syncLog 的状态与错误文本，不负责并发守卫与任务态持久化。
func (app *DataSyncAppImpl) runSync(ctx context.Context, task *entity.DataSyncTask, srcConn, targetConn *dbi.DbConn, syncLog *entity.DataSyncLog, metrics *SyncMetrics) error {
	// 运行时双重拦截：存量脏数据（校验引入前创建的任务）防御
	if verr := validateDataSyncSQL(srcConn.GetDialect(), task); verr != nil {
		syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", verr.Error())
		logx.ErrorContext(ctx, syncLog.ErrText)
		return verr
	}

	// 数据校验模式：doDataValidation 自行落定状态与结果文本
	if task.SyncMode == entity.DataSyncModeValidation {
		app.doDataValidation(ctx, task, srcConn, targetConn, syncLog, metrics)
		if syncLog.Status == entity.DataSyncTaskStateFail {
			return errorx.NewBiz(syncLog.ErrText)
		}
		return nil
	}

	// 构建源查询规划（增量条件、排序、软删除过滤、主键分片区间）
	plan, err := app.planSyncQuery(ctx, task, srcConn, srcConn.GetDialect(), syncLog)
	if err != nil {
		syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
		logx.ErrorContext(ctx, syncLog.ErrText)
		return err
	}

	if err = app.doDataSync(ctx, plan, task, srcConn, targetConn, syncLog, metrics); err != nil {
		syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
		logx.ErrorContext(ctx, syncLog.ErrText)
		syncLog.Status = entity.DataSyncTaskStateFail
		return err
	}
	syncLog.Status = entity.DataSyncTaskStateSuccess
	return nil
}

// SyncTask 以调用方给定的源/目标连接同步执行一轮数据同步任务，返回执行结果并落定 syncLog。
// 与异步 Run 共享同一执行体（runSync），区别在于：同步返回、连接由调用方给定、不读写任务表与运行态。
// 供调度以外的集成/驱动场景使用（如同步任务预览、测试）。
func (app *DataSyncAppImpl) SyncTask(ctx context.Context, task *entity.DataSyncTask, srcConn, targetConn *dbi.DbConn, syncLog *entity.DataSyncLog, metrics *SyncMetrics) error {
	if metrics == nil {
		metrics = NewSyncMetrics()
	}
	runId, ok := app.runGuard.AcquireWithRunId(task.Id)
	if !ok {
		return errorx.NewBiz("the task is running")
	}
	defer app.runGuard.ReleaseWithRunId(task.Id, runId)
	// 将本次锁的 runId 落到调用方提供的 syncLog 上，使下游批次写入能走同一套“同一次执行”归属判定；
	// 测试/预览入口与 Run 入口共享同一条约束，不需要两套语义。
	if syncLog != nil {
		syncLog.RunId = runId
	}
	return app.runSync(ctx, task, srcConn, targetConn, syncLog, metrics)
}

// syncQueryPlan 一轮同步的源查询规划。
//
// 为什么不是一个拼好的字符串而是一次规划：分片扫描下同一轮要发 N 条源查询，而增量字段的
// 取值形态探测（跑一次源查询、首行即停）与增量条件/排序子句的拼装对每个分片都一样。
// 拼成字符串再逐片插主键区间条件就得对 SQL 做字符串手术（插在哪种子句之前容易错），
// 把「一次算好的片段」与「每片不同的区间」分开后，分片只是 sqlFor 的一个入参。
type syncQueryPlan struct {
	dataSQL string // 去尾部空白与分号后的源查询主体
	where   string // "where 1=1" 或空（源查询已含 WHERE 时不补）
	pred    string // 增量条件 + 软删除过滤（AND 链，接在分片区间之后）
	order   string // 默认排序（增量字段升序）；分片续扫时由 sqlForOrder 换成分片键升序
}

// newSyncQueryPlan 按四段组装一次规划，供执行链路与测试共用（不做任何数据库探测）。
func newSyncQueryPlan(dataSQL, where, pred, order string) *syncQueryPlan {
	return &syncQueryPlan{dataSQL: dataSQL, where: where, pred: strings.TrimSpace(pred), order: strings.TrimSpace(order)}
}

// sqlFor 组装指定分片的完整源查询；cond 为空即整表单次查询（也是写入执行日志的那条完整 SQL）。
//
// 分片区间的落点：始终在 WHERE 子句链里（接在 where 片段之后、增量条件与排序之前）。
// planSyncQuery 保证「源查询无 WHERE 则 where=where 1=1，有则 where=」，因此这里拼 `and (...)`
// 总能接上；空片段不参与拼接，避免拼出双空格（DataSQLFull 会直接展示给用户）。
func (p *syncQueryPlan) sqlFor(cond string) string {
	return p.sqlForOrder(cond, p.order)
}

// sqlForOrder 同 sqlFor，但用传入的排序片段替换默认排序。
//
// 存在的理由（片内续扫的前提）：续扫用「最后读到的分片键值」做键集游标（`key > 该值`），
// 这要求该查询的结果是按分片键**有序**返回的——无序扫描时，未读行可能落在已读键值的左侧，
// 按 `>` 续扫会静默漏行。分片模式下把排序换成 `order by 分片键 asc` 后，已读行恰是键值最小的
// 一段，续扫点才能证明无损。默认的「按增量字段排序」不满足这个性质（分片键与增量字段不同序）。
func (p *syncQueryPlan) sqlForOrder(cond, order string) string {
	parts := make([]string, 0, 5)
	for _, seg := range []string{p.dataSQL, p.where} {
		if seg != "" {
			parts = append(parts, seg)
		}
	}
	if cond != "" {
		// 括号包裹：区间自身是 `pk >= a AND pk <= b`，包上后与周边 and 链的优先关系一目了然，
		// 也不依赖区间串自身不含 OR
		parts = append(parts, "and ("+cond+")")
	}
	if p.pred != "" {
		parts = append(parts, p.pred)
	}
	if order != "" {
		parts = append(parts, order)
	}
	// 片段必须用换行而非空格连接：行注释（`--` / mysql 的 `#`）到行尾结束，而 planSyncQuery
	// 里的 TrimSpace+TrimRight(";") 恰好会把注释行尾的换行裁掉。用空格连接时，拼接的
	// where/区间/tail 会整段落进用户 DataSQL 的尾行注释里——语句仍然合法，但分片区间完全消失：
	// 每一片读的都是同一张过滤后的整表，N 片 = N 倍重复写入且行数与目标一一对得上，任务自标成功。
	// 这是「静默大规模错数据」里最难发现的一类，换行连接从根上消除该形态（各方言行注释均以换行结束）。
	return strings.Join(parts, "\n")
}

// planSyncQuery 构建源查询规划：根据同步模式组装增量条件、排序、软删除过滤等。
func (app *DataSyncAppImpl) planSyncQuery(ctx context.Context, task *entity.DataSyncTask, srcConn *dbi.DbConn, dialect dbi.Dialect, syncLog *entity.DataSyncLog) (*syncQueryPlan, error) {
	// 是否需要「全量扫描源查询」（不叠加水位增量条件）：见 syncModeNeedsFullScan（真源已收到校验层，
	// 与保存期的「会不会注入谓词」判定共用同一个口径）：
	//   - 全量刷新(3)：TRUNCATE + 全量 INSERT，本就须读全表；
	//   - 增量+软删除(4) / 增量+硬删除(5)：删除对账靠「DELETE 目标 WHERE 唯一键 NOT IN 源主键集」实现，
	//     该主键集必须是**完整**源集合。若沿用水位增量只取本次新增/变更行，NOT IN 会把目标中「源仍存在、
	//     但未落入本次增量」的行一并误删（静默大规模数据丢失）。故这三种模式一律全量扫描源表。
	needsFullScan := syncModeNeedsFullScan(task.SyncMode)

	// 增量合并/软删除/硬删除模式自动切为 UPSERT 策略（各方言原生语法）
	if task.SyncMode == entity.DataSyncModeIncrementalMerge ||
		task.SyncMode == entity.DataSyncModeIncrementalSoftDel ||
		task.SyncMode == entity.DataSyncModeIncrementalHardDel {
		task.DuplicateStrategy = dbi.DuplicateStrategyUpdate
	}

	// 多方言感知水位线格式化 + 多字段增量条件
	updSQL := ""
	orderSQL := ""
	if !needsFullScan && task.UpdFieldVal != "0" && task.UpdFieldVal != "" && task.UpdField != "" {
		updFieldDataType := dbi.DefaultDbDataType
		_, probeErr := srcConn.WalkQueryRows(ctx, task.DataSQL, func(row map[string]any, columns []*dbi.QueryColumn) error {
			for _, column := range columns {
				if strings.EqualFold(column.Name, cmp.Or(task.UpdFieldSrc, task.UpdField)) {
					updFieldDataType = column.DbDataType
					break
				}
			}
			return dbi.NewStopWalkQueryError("get column data type... ignore~")
		})
		if probeErr != nil {
			logx.WarnfContext(ctx, "failed to probe the data type of sync update field [%s]: %s", task.UpdField, probeErr.Error())
		}

		// 方言感知水位线格式化
		formattedVal := updFieldDataType.Codec.SQLValue(task.UpdFieldVal)
		// 边界语义：Auto 按模式取默认（合并/对账类为 Inclusive、追加/刷新/校验类为 Exclusive）；
		// Inclusive 行不丢弃同时间戳边界，依赖目标 UPSERT 幂等，即 at-least-once 交付语义。
		cmpOp := ">"
		if resolveCursorInclusivity(task.SyncMode, task.GetCursorInclusivity()) {
			cmpOp = ">="
		}
		updSQL = fmt.Sprintf("and %s %s %s", task.UpdField, cmpOp, formattedVal)

		// 辅助增量字段（多字段增量条件）：与主字段共用水位值，边界语义同主字段
		if task.UpdFieldSecondary != "" {
			updSQL += fmt.Sprintf(" and %s %s %s", task.UpdFieldSecondary, cmpOp, formattedVal)
		}
	}

	// 即使是首次同步，如果有更新字段也要添加排序，确保每次查询结果顺序一致
	if task.UpdField != "" {
		orderSQL = "order by " + task.UpdField + " asc "
	}

	// 解析器判断DataSQL是否已含where条件
	var where = "where 1=1"
	if dataSQLHasWhere(dialect, task) {
		where = ""
	}

	// 软删除模式：过滤掉源端已软删除的记录
	var softDelFilter string
	if task.SyncMode == entity.DataSyncModeIncrementalSoftDel && task.SoftDeleteField != "" && task.SoftDeleteValue != "" {
		// 构建软删除过滤条件：排除软删除标记值等于指定值的记录
		// 标记值使用源方言的字符串转义；标识符已由保存与运行校验限制。
		formattedValue := escapeFnForDialect(srcConn.Info.Type)(task.SoftDeleteValue)
		softDelFilter = fmt.Sprintf("and (%s != %s OR %s IS NULL)", task.SoftDeleteField, formattedValue, task.SoftDeleteField)
		app.appendRunLog(syncLog, "软删除模式: 过滤条件 [%s]", softDelFilter)
	}

	// DataSQL尾部可能带分号
	dataSQL := strings.TrimRight(strings.TrimSpace(task.DataSQL), ";")

	plan := newSyncQueryPlan(dataSQL, where, updSQL+" "+softDelFilter, orderSQL)
	syncLog.DataSQLFull = plan.sqlFor("")

	return plan, nil
}

func (app *DataSyncAppImpl) StopTask(ctx context.Context, taskId uint64) error {
	task := new(entity.DataSyncTask)
	task.Id = taskId
	task.RunningState = entity.DataSyncTaskRunStateStop
	if err := app.UpdateById(ctx, task); err != nil {
		return err
	}
	app.runGuard.RequestStop(taskId)
	return nil
}
