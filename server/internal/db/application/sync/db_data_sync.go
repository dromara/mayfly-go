package sync

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/taskx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// DbApp 数据同步依赖的窄接口：仅需按库获取连接（Go惯例：消费者定义接口，
// 主包DbAppImpl天然满足，避免子包反向依赖主包造成循环）
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

	// 构建同步查询 SQL（增量条件、排序、软删除过滤等）
	sqlStr, err := app.buildSyncQuery(ctx, task, srcConn, srcConn.GetDialect(), syncLog)
	if err != nil {
		syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
		logx.ErrorContext(ctx, syncLog.ErrText)
		return err
	}

	if err = app.doDataSync(ctx, sqlStr, task, srcConn, targetConn, syncLog, metrics); err != nil {
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

// buildSyncQuery 构建同步查询 SQL：根据同步模式组装增量条件、排序、软删除过滤等。
// 拆分自 Run 方法，降低单方法复杂度。
func (app *DataSyncAppImpl) buildSyncQuery(ctx context.Context, task *entity.DataSyncTask, srcConn *dbi.DbConn, dialect dbi.Dialect, syncLog *entity.DataSyncLog) (string, error) {
	// 是否需要「全量扫描源查询」（不叠加水位增量条件）：
	//   - 全量刷新(3)：TRUNCATE + 全量 INSERT，本就须读全表；
	//   - 增量+软删除(4) / 增量+硬删除(5)：删除对账靠「DELETE 目标 WHERE 唯一键 NOT IN 源主键集」实现，
	//     该主键集必须是**完整**源集合。若沿用水位增量只取本次新增/变更行，NOT IN 会把目标中「源仍存在、
	//     但未落入本次增量」的行一并误删（静默大规模数据丢失）。故这三种模式一律全量扫描源表。
	needsFullScan := task.SyncMode == entity.DataSyncModeFullRefresh ||
		task.SyncMode == entity.DataSyncModeIncrementalSoftDel ||
		task.SyncMode == entity.DataSyncModeIncrementalHardDel

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
		// Inclusive 行不丢弃同时间戳边界，依赖目标 UPSERT 幂等，与业界 at-least-once 对齐。
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

	// 组装查询sql
	sqlStr := fmt.Sprintf("%s %s %s %s %s", dataSQL, where, updSQL, softDelFilter, orderSQL)
	syncLog.DataSQLFull = sqlStr

	return sqlStr, nil
}

func (app *DataSyncAppImpl) doDataSync(ctx context.Context, sql string, task *entity.DataSyncTask, srcConn, targetConn *dbi.DbConn, syncLog *entity.DataSyncLog, metrics *SyncMetrics) error {
	app.appendRunLog(syncLog, "========================================")
	app.appendRunLog(syncLog, "开始执行同步任务: %s", task.TaskName)
	app.appendRunLog(syncLog, "========================================")
	app.appendRunLog(syncLog, "同步模式: %s", getSyncModeName(task.SyncMode))
	app.appendRunLog(syncLog, "目标表: %s.%s", task.TargetDbName, task.TargetTableName)
	app.appendRunLog(syncLog, "冲突策略: %s", getDuplicateStrategyName(task.DuplicateStrategy))
	if task.PageSize > 0 {
		app.appendRunLog(syncLog, "批次大小: %d 行", task.PageSize)
	}
	app.appendRunLog(syncLog, "源库: %s，目标库: %s", task.SrcDbName, task.TargetDbName)

	// task.FieldMap为json数组字符串 [{"src":"id","target":"id"}]，转为map
	var fieldMap []map[string]string
	err := json.Unmarshal([]byte(task.FieldMap), &fieldMap)
	if err != nil {
		return errorx.NewBizf("there was an error parsing the field map json: %s", err.Error())
	}
	app.appendRunLog(syncLog, "字段映射: %d 个字段", len(fieldMap))

	// 初始化转换引擎和过滤引擎
	app.appendRunLog(syncLog, "[3/6] 初始化处理引擎...")
	transformEngine, err := NewTransformEngine(task.TransformRules)
	if err != nil {
		return errorx.NewBizf("failed to initialize transform engine: %s", err.Error())
	}
	if transformEngine != nil {
		app.appendRunLog(syncLog, "✓ 转换引擎已启用")
	}
	filterEngine := NewFilterEngine(task.FilterCondition)
	if err := filterEngine.Validate(); err != nil {
		return errorx.NewBizf("invalid filter condition: %s", err.Error())
	}
	if filterEngine != nil {
		app.appendRunLog(syncLog, "✓ 过滤引擎已启用")
	}

	// 初始化冲突检测器（双向同步 + 配置了时间戳字段时启用）
	var conflictDetector *ConflictDetector
	if task.BiDirEnabled && task.BiDirTimestampField != "" {
		conflictDetector = NewConflictDetector(task.ConflictStrategy, task.BiDirTimestampField)
		logx.InfofContext(ctx, "bidirectional sync: conflict detection enabled for task [%s], strategy=%d, timestampField=%s",
			task.TaskName, task.ConflictStrategy, task.BiDirTimestampField)
		app.appendRunLog(syncLog, "✓ 双向同步冲突检测已启用 (字段: %s)", task.BiDirTimestampField)
	}

	// 获取目标表列信息
	app.appendRunLog(syncLog, "[4/6] 分析目标表结构...")
	targetTableColumns, err := targetConn.Metadata().GetColumns(task.TargetTableName)
	if err != nil {
		return errorx.NewBizf("failed to get target table columns: %s", err.Error())
	}
	app.appendRunLog(syncLog, "✓ 目标表共 %d 列", len(targetTableColumns))
	targetColumnName2Column := collx.ArrayToMap(targetTableColumns, func(column dbi.Column) string {
		return column.ColumnName
	})

	// Schema 演化检测（仅检测 fieldMap 映射的目标列是否在目标表中存在）
	app.appendRunLog(syncLog, "[5/6] 检测Schema变更...")
	schemaDetector := NewSchemaDetector(task.SchemaEvolveMode)
	if schemaDetector != nil {
		changes := schemaDetector.DetectChanges(fieldMap, targetTableColumns)
		if len(changes) > 0 {
			skipCols := schemaDetector.HandleChanges(changes, task.TaskName)
			syncLog.SchemaChanges = len(changes)
			app.appendRunLog(syncLog, "⚠ Schema变更检测: 发现 %d 处变更", len(changes))
			// 自动适配模式：过滤掉目标表中不存在的列映射
			if len(skipCols) > 0 {
				fieldMap = FilterFieldMapBySchemaChanges(fieldMap, skipCols)
				logx.InfofContext(ctx, "schema-evolve auto: skipped %d column mappings for task [%s]", len(skipCols), task.TaskName)
				app.appendRunLog(syncLog, "✓ Schema自动适配: 跳过 %d 个列映射", len(skipCols))
			}
		} else {
			app.appendRunLog(syncLog, "✓ Schema变更检测: 无变更")
		}
	} else {
		app.appendRunLog(syncLog, "- Schema检测已禁用")
	}

	// 目标库对应的insert columns
	targetInsertColumns := make([]dbi.Column, 0, len(fieldMap))
	for _, val := range fieldMap {
		column, ok := targetColumnName2Column[val["target"]]
		if !ok {
			return errorx.NewBizf("field map target column [%s] not found in target table [%s]", val["target"], task.TargetTableName)
		}
		if column.IsGenerated {
			logx.WarnfContext(ctx, "the target column [%s] of the synchronization task [%s] is a generated column, and its mapping will be ignored", val["target"], task.TaskName)
			continue
		}
		targetInsertColumns = append(targetInsertColumns, column)
	}
	if len(targetInsertColumns) == 0 {
		return errorx.NewBizf("no insertable columns are configured for the target table [%s]", task.TargetTableName)
	}

	// 构建目标表元信息
	targetTableMeta := BuildTargetTableMeta(targetConn, task.TargetTableName, targetTableColumns)
	if needsDeleteReconcile(task.SyncMode) {
		if err := validateReconcileKeys(targetConn.Info.Type, targetTableMeta, targetInsertColumns); err != nil {
			return err
		}
	}

	// 全量刷新模式：先清空目标表
	if task.SyncMode == entity.DataSyncModeFullRefresh {
		sqlGen := targetConn.GetDialect().GetSQLGenerator()
		truncateSQLs := sqlGen.GenTruncate(task.TargetTableName)
		if len(truncateSQLs) > 0 {
			for _, sql := range truncateSQLs {
				if _, execErr := targetConn.ExecContext(ctx, sql); execErr != nil {
					return errorx.NewBizf("failed to truncate target table [%s]: %s", task.TargetTableName, execErr.Error())
				}
			}
			logx.InfofContext(ctx, "full refresh mode: truncated target table [%s]", task.TargetTableName)
			app.appendRunLog(syncLog, "全量刷新模式: 已清空目标表 [%s]", task.TargetTableName)
		} else {
			return errorx.NewBizf("full refresh is not supported for target table [%s]: the dialect cannot clear the table", task.TargetTableName)
		}
	}

	// 记录总数
	total := 0
	batchSize := normalizeBatchSize(task.PageSize)
	result := make([]map[string]any, 0)
	lastBatchTime := time.Now()
	batchNum := 0

	// 如果有数据库别名，则从UpdField中去掉数据库别名
	updFieldName := task.UpdField
	if task.UpdField != "" && strings.Contains(task.UpdField, ".") {
		updFieldName = strings.Split(task.UpdField, ".")[1]
	}

	app.appendRunLog(syncLog, "[6/6] 开始执行数据同步...")
	app.appendRunLog(syncLog, "执行SQL: %s", truncateString(sql, 200))

	// 构建同步执行期上下文（封装 12 个参数为结构体）
	failedRowRecorder := NewFailedRowRecorder(defaultFailedRowCapacity)
	sec := &syncExecContext{
		fieldMap:          fieldMap,
		updFieldName:      updFieldName,
		task:              task,
		targetConn:        targetConn,
		targetColumns:     targetInsertColumns,
		targetTableMeta:   targetTableMeta,
		transformEngine:   transformEngine,
		filterEngine:      filterEngine,
		conflictDetector:  conflictDetector,
		metrics:           metrics,
		failedRowRecorder: failedRowRecorder,
	}

	// 源列缺失检测：从源查询结果首行的列信息零成本获取源列，检测 fieldMap 映射的源列是否缺失
	srcSchemaChecked := false

	_, err = srcConn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		// 首行到达时用真实源查询列做一次源列缺失检测（源列缺失会静默产生 NULL）
		if !srcSchemaChecked {
			srcSchemaChecked = true
			if filterEngine != nil {
				if err := filterEngine.root.validateFields(row); err != nil {
					return err
				}
			}
			if schemaDetector != nil && len(columns) > 0 {
				srcColNames := make([]string, 0, len(columns))
				for _, c := range columns {
					srcColNames = append(srcColNames, c.Name)
				}
				if srcChanges := schemaDetector.DetectSourceColumnChanges(fieldMap, srcColNames); len(srcChanges) > 0 {
					schemaDetector.HandleChanges(srcChanges, task.TaskName)
					syncLog.SchemaChanges += len(srcChanges)
					app.appendRunLog(syncLog, "⚠ 源列缺失检测: %d 个映射源列不存在于源查询结果，对应目标列将写入 NULL", len(srcChanges))
				}
			}
		}

		// 过滤引擎 - 不满足条件的行直接跳过
		if !filterEngine.Match(row) {
			metrics.SkipCount++
			return nil
		}

		total++
		result = append(result, row)

		// 按行数分批
		if total%batchSize == 0 {
			batchNum++
			batchStart := time.Now()
			if err := app.srcData2TargetDb(ctx, result, sec); err != nil {
				return err
			}

			// 目标写入已完成，但任务级水位属共享状态：仅在仍是本次锁持有者时推进，
			// 避免新实例接管后旧任务默默推水位把新执行的进度拉到旧位置。
			if app.runGuard.IsCurrentRun(task.Id, syncLog.RunId) && !app.runGuard.IsStopRequested(task.Id) {
				if pwErr := app.persistUpdFieldVal(ctx, task); pwErr != nil {
					logx.WarnfContext(ctx, "watermark persist failed (batch #%d): %s", batchNum, pwErr.Error())
				}
			} else {
				logx.WarnfContext(ctx, "sync task [%d] was preempted or stop-requested, skipping watermark persist (batch #%d)", task.Id, batchNum)
			}

			now := time.Now()
			batchDuration := now.Sub(batchStart).Milliseconds()
			if elapsed := now.Sub(lastBatchTime).Seconds(); elapsed > 0 {
				logx.DebugfContext(ctx, "data sync batch rate: %.0f rows/s", float64(batchSize)/elapsed)
			}
			lastBatchTime = now

			metrics.RecordBatch(len(result))

			syncLog.ErrText = i18n.T(imsg.DataSyncingMsg, "count", total)
			logx.InfoContext(ctx, syncLog.ErrText)
			syncLog.ResNum = total
			app.appendRunLog(syncLog, "  ✓ 批次 #%d 完成: %d 行, 耗时 %dms", batchNum, len(result), batchDuration)
			app.saveLog(syncLog)

			result = result[:0]

			// 以 runId 而非任务级锁存在性判断“本次执行是否仍有效”，避免新实例已接管时旧任务继续写入：
			// 旧逻辑 IsRunning(taskId) 在新实例 Acquire 后仍为 true，会默默多跑一批。
			// 批边界停止判定合并两个信号：
			// - IsCurrentRun：本地/Redis 归属，捕获"TTL 过期被他实例接管"
			// - IsStopRequested：Redis 停止标记，捕获"他实例调 StopTask"（本地 fast path 无法感知）
			if !app.runGuard.IsCurrentRun(task.Id, syncLog.RunId) || app.runGuard.IsStopRequested(task.Id) {
				return errorx.NewBiz("the task has been terminated or preempted")
			}

			// 批间可打断 sleep：定位为“目标侧节流”（缓解 replica lag / WAL apply 积压）；
			// 对 mysql/pg 默认 buffered 驱动的源侧几乎无效（长 SELECT 开头一次拉入客户端），
			// 真需减源压需开启分片扫描（P2）。tick 内检查 runId 归属，被抢占时立刻中止。
			if sleepMs := task.GetSleepBetweenBatchesMs(); sleepMs > 0 {
				sleepDur := time.Duration(sleepMs) * time.Millisecond
				alive := interruptibleSleep(sleepDur, 100*time.Millisecond, func() bool {
					return app.runGuard.IsCurrentRun(task.Id, syncLog.RunId) && !app.runGuard.IsStopRequested(task.Id)
				})
				if !alive {
					return errorx.NewBiz("the task has been terminated or preempted during batch sleep")
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 处理剩余的数据
	if len(result) > 0 {
		if err := app.srcData2TargetDb(ctx, result, sec); err != nil {
			return err
		}
		if app.runGuard.IsCurrentRun(task.Id, syncLog.RunId) && !app.runGuard.IsStopRequested(task.Id) {
			if pwErr := app.persistUpdFieldVal(ctx, task); pwErr != nil {
				logx.WarnfContext(ctx, "watermark persist failed (final batch): %s", pwErr.Error())
			}
		} else {
			logx.WarnfContext(ctx, "sync task [%d] was preempted or stop-requested, skipping watermark persist (final batch)", task.Id)
		}
		metrics.RecordBatch(len(result))
		batchNum++
	}

	if err := app.reconcileDeletes(ctx, sec, syncLog); err != nil {
		return err
	}

	logx.InfofContext(ctx, "synchronous task: [%s], finished execution, save records successfully: [%d]", task.TaskName, total)

	syncLog.ErrText = i18n.T(imsg.DataSyncSuccessMsg, "count", total)
	syncLog.ResNum = total

	// 输出详细的执行摘要
	app.appendRunLog(syncLog, "========================================")
	app.appendRunLog(syncLog, "同步执行完成")
	app.appendRunLog(syncLog, "========================================")
	app.appendRunLog(syncLog, "总计处理: %d 行", total)
	app.appendRunLog(syncLog, "执行批次: %d 批", batchNum)
	if metrics.InsertCount > 0 {
		app.appendRunLog(syncLog, "新增行数: %d", metrics.InsertCount)
	}
	if metrics.UpdateCount > 0 {
		app.appendRunLog(syncLog, "更新行数: %d", metrics.UpdateCount)
	}
	if metrics.DeleteCount > 0 {
		app.appendRunLog(syncLog, "删除行数: %d", metrics.DeleteCount)
	}
	if metrics.SkipCount > 0 {
		app.appendRunLog(syncLog, "跳过行数: %d", metrics.SkipCount)
	}
	if metrics.EndTime > metrics.StartTime {
		durationMs := metrics.EndTime - metrics.StartTime
		app.appendRunLog(syncLog, "总耗时: %d ms", durationMs)
		if durationMs > 0 {
			throughput := int64(metrics.TotalRows) * 1000 / durationMs
			app.appendRunLog(syncLog, "吞吐量: %d 行/秒", throughput)
		}
	}
	if syncLog.SchemaChanges > 0 {
		app.appendRunLog(syncLog, "Schema变更: %d 处", syncLog.SchemaChanges)
	}
	// 失败行摘要（Dead Letter）
	if failedRowRecorder != nil && failedRowRecorder.TotalCount() > 0 {
		app.appendRunLog(syncLog, "失败行: %s", failedRowRecorder.Summary())
	}
	app.appendRunLog(syncLog, "========================================")

	return nil
}

// syncExecContext 同步执行期上下文：封装单批次写入所需的全部运行时依赖，
// 避免 srcData2TargetDb 参数列表过长（原 12 个参数），同时便于后续扩展。
type syncExecContext struct {
	fieldMap          []map[string]string
	updFieldName      string
	task              *entity.DataSyncTask
	targetConn        *dbi.DbConn
	targetColumns     []dbi.Column
	targetTableMeta   *dbi.TargetTableMeta
	transformEngine   *TransformEngine
	filterEngine      *FilterEngine
	conflictDetector  *ConflictDetector
	metrics           *SyncMetrics
	failedRowRecorder *FailedRowRecorder
	reconcileKeys     [][]any
}

// srcData2TargetDb 源数据到目标库的转换与写入。
// 编排四个子步骤：批量冲突检测+转换 → 冲突键存量探测 → 事务写入 → 水位推进+指标计数。
func (app *DataSyncAppImpl) srcData2TargetDb(ctx context.Context, srcRes []map[string]any, sec *syncExecContext) (err error) {
	// Step 1: 批量冲突检测 + 转换引擎应用
	targetData, err := app.transformBatch(ctx, srcRes, sec)
	if err != nil {
		return err
	}

	if needsDeleteReconcile(sec.task.SyncMode) && len(targetData) != len(srcRes) {
		return fmt.Errorf("delete reconciliation cannot continue after transformation or conflict handling skipped rows")
	}
	if err := sec.collectReconcileKeys(targetData); err != nil {
		return err
	}

	// Step 2: 写入前探测冲突键存量（写入后所有键都已存在，再也分不出新增与更新）
	split, splitErr := probeWriteSplit(ctx, sec, targetData)

	// Step 3: 生成 SQL 并事务写入目标库
	if err := app.executeTargetInsert(ctx, targetData, srcRes, sec); err != nil {
		return err
	}

	// Step 4: 水位推进 + 指标更新
	if watermarkErr := advanceSyncWatermark(srcRes, sec.task, sec.updFieldName); watermarkErr != nil {
		logx.WarnfContext(ctx, "failed to advance the data sync watermark of task [%d]: %s", sec.task.Id, watermarkErr.Error())
	}
	if sec.metrics != nil {
		if splitErr != nil {
			// 探测失败只影响观测粒度，不值得让已成功写入的批次回滚，退回按策略名义计数的粗粒度口径
			logx.WarnfContext(ctx, "failed to split the sync write metrics of task [%d]: %s", sec.task.Id, splitErr.Error())
			split = coarseWriteSplit(cmp.Or(sec.task.DuplicateStrategy, dbi.DuplicateStrategyNone), len(targetData))
		}
		sec.metrics.RecordWrite(split.inserts, split.updates, split.ignored)
	}
	return nil
}

// transformBatch 对源数据批次应用冲突检测和转换规则，返回转换后的目标行集合。
// 冲突检测批量查询失败时返回 error，调用方须中止本批次（无冲突数据时不得继续写入，以免覆盖目标已变更行）。
func (app *DataSyncAppImpl) transformBatch(ctx context.Context, srcRes []map[string]any, sec *syncExecContext) ([]map[string]any, error) {
	fieldMap := sec.fieldMap
	task := sec.task
	targetDbConn := sec.targetConn
	targetTableMeta := sec.targetTableMeta
	transformEngine := sec.transformEngine
	metrics := sec.metrics

	// 批量冲突检测 — 分批查询目标行，消除 N+1 查询
	conflictMap, err := app.batchLoadConflictTargetRows(ctx, srcRes, sec)
	if err != nil {
		return nil, err
	}

	// 构建 target2Src 映射供冲突检测行内使用（仅当 conflictMap 非 nil 时）
	var target2Src map[string]string
	if conflictMap != nil {
		target2Src = sec.target2SrcMap()
	}

	targetData := make([]map[string]any, 0, len(srcRes))
	for _, srcData := range srcRes {
		// 冲突检测：从批量查询结果中查找
		if conflictMap != nil {
			pkKey := buildPKKey(srcData, target2Src, targetTableMeta)
			if pkKey != "" {
				if targetRow, exists := conflictMap[pkKey]; exists {
					if sec.conflictDetector.DetectConflict(srcData, targetRow) {
						if metrics != nil {
							metrics.SkipCount++
						}
						pkWhere := buildPKWhereClause(srcData, target2Src, targetTableMeta, escapeFnForDialect(targetDbConn.Info.Type))
						logx.DebugContext(ctx, sec.conflictDetector.ResolveConflict(task.TaskName, pkWhere))
						continue
					}
				}
			}
		}

		// 应用转换引擎
		var data map[string]any
		var skip bool
		if transformEngine != nil {
			data, skip = transformEngine.Transform(srcData, fieldMap, task)
		} else {
			data = make(map[string]any, len(fieldMap))
			for _, item := range fieldMap {
				val := srcData[item["src"]]
				if val == nil {
					switch task.NullStrategy {
					case entity.NullStrategyDefault:
						val = task.NullDefault
					case entity.NullStrategySkipRow:
						skip = true
					}
				}
				data[item["target"]] = val
			}
		}
		if skip {
			if metrics != nil {
				metrics.SkipCount++
			}
			continue
		}
		targetData = append(targetData, data)
	}
	return targetData, nil
}

// executeTargetInsert 将转换后的目标数据写入目标库。
//
// 优先「参数化直插」：目标方言实现 dbi.ParamInserter 且为直接插入（DuplicateStrategyNone）时，
// 生成单条占位符 INSERT、值交驱动绑定，规避文本回放的转义边界与精度损失；
// 否则（含 ignore/upsert 冲突处理、未实现该能力的方言）回退既有 GenInsert 文本路径。
// 失败时记录 Dead Letter 并回滚事务。
func (app *DataSyncAppImpl) executeTargetInsert(ctx context.Context, targetData []map[string]any, srcRes []map[string]any, sec *syncExecContext) (err error) {
	if len(targetData) == 0 {
		return nil
	}

	task := sec.task
	targetDbConn := sec.targetConn
	targetInsertColumns := sec.targetColumns
	targetTableMeta := sec.targetTableMeta

	targetValues := make([][]any, 0, len(targetData))
	for _, item := range targetData {
		values := make([]any, 0, len(targetInsertColumns))
		for _, column := range targetInsertColumns {
			values = append(values, item[column.ColumnName])
		}
		targetValues = append(targetValues, values)
	}

	// 参数路径按占位符预算拆分语句，所有语句仍在同一批次事务中提交。
	type execItem struct {
		sql  string
		args []any
	}
	var items []execItem
	strategy := cmp.Or(task.DuplicateStrategy, dbi.DuplicateStrategyNone)
	sqlGen := targetDbConn.GetDialect().GetSQLGenerator()
	if strategy == dbi.DuplicateStrategyNone {
		if pi := dbi.GetParamInserter(sqlGen); pi != nil {
			bound, err := bindSyncValues(targetDbConn.Info.Type, targetInsertColumns, targetValues)
			if err != nil {
				return err
			}
			if len(targetInsertColumns) == 0 {
				return fmt.Errorf("no target columns for parameterized insert")
			}
			const maxBindParams = 60000
			rowsPerStmt := max(1, maxBindParams/len(targetInsertColumns))
			for start := 0; start < len(bound); start += rowsPerStmt {
				stmt, args, perr := pi.GenInsertParams(task.TargetTableName, targetInsertColumns, bound[start:min(start+rowsPerStmt, len(bound))])
				if perr != nil {
					return errorx.NewBizf("failed to generate parameterized insert statement: %s", perr.Error())
				}
				if stmt == "" {
					return fmt.Errorf("parameterized insert generated an empty statement")
				}
				items = append(items, execItem{sql: stmt, args: args})
			}
		}
	}
	if items == nil {
		for _, s := range sqlGen.GenInsert(task.TargetTableName, targetInsertColumns, targetValues, strategy, targetTableMeta) {
			items = append(items, execItem{sql: s})
		}
	}

	if len(items) == 0 {
		return fmt.Errorf("target dialect generated no insert statements for a non-empty batch")
	}
	targetDbTx, err := targetDbConn.Begin()
	if err != nil {
		return errorx.NewBizf("failed to start the target database transaction: %s", err.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			dbi.RollbackTx(targetDbTx)
			err = fmt.Errorf("%v", r)
		}
	}()

	for _, item := range items {
		if _, execErr := targetDbConn.TxExecContext(ctx, targetDbTx, item.sql, item.args...); execErr != nil {
			dbi.RollbackTx(targetDbTx)
			if sec.failedRowRecorder != nil {
				for _, row := range srcRes {
					sec.failedRowRecorder.Record(row, execErr, 0)
				}
			}
			return execErr
		}
	}

	if commitErr := dbi.CommitTargetTx(targetDbConn, targetDbTx); commitErr != nil {
		if sec.failedRowRecorder != nil {
			for _, row := range srcRes {
				sec.failedRowRecorder.Record(row, commitErr, 0)
			}
		}
		return errorx.NewBizf("data synchronization - The target database transaction failed to commit: %s", commitErr.Error())
	}
	return nil
}

// doDataValidation 数据校验模式。
// 对比源/目标行数，并可选抽样 checksum 对比（按字段映射的主键列）。
func (app *DataSyncAppImpl) doDataValidation(ctx context.Context, task *entity.DataSyncTask, srcConn, targetConn *dbi.DbConn, syncLog *entity.DataSyncLog, metrics *SyncMetrics) {
	app.appendRunLog(syncLog, "开始数据校验模式: 任务 [%s]", task.TaskName)
	app.appendRunLog(syncLog, "源库: %s，目标库: %s", task.SrcDbName, task.TargetDbName)

	// 校验查询与同步查询(buildSyncQuery)一致：去除 DataSQL 尾部空白与分号，
	// 否则作为子查询包裹 `SELECT ... FROM (DataSQL) _src` 时尾分号会导致整条 SQL 语法错误
	dataSQL := strings.TrimRight(strings.TrimSpace(task.DataSQL), ";")

	// 对比全部映射字段的原值；无有效映射时不能声称内容校验通过。
	var keyColumns []string
	var srcKeyCols []string
	var fieldMap []map[string]string
	mapErr := json.Unmarshal([]byte(task.FieldMap), &fieldMap)
	if mapErr != nil || len(fieldMap) == 0 {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = "validation requires a valid, non-empty field mapping"
		return
	}
	srcQuote := srcConn.GetDialect().Quoter().QuoteIdent
	targetQuote := targetConn.GetDialect().Quoter().QuoteIdent
	for _, fm := range fieldMap {
		if fm["src"] == "" || fm["target"] == "" {
			syncLog.Status = entity.DataSyncTaskStateFail
			syncLog.ErrText = "validation field mapping contains an empty source or target column"
			return
		}
		keyColumns = append(keyColumns, targetQuote(fm["target"]))
		srcKeyCols = append(srcKeyCols, srcQuote(fm["src"]))
	}
	targetTable := quoteTargetTableRef(targetConn, task.TargetTableName)

	// 源数据行数
	srcCount := 0
	if _, srcErr := srcConn.WalkQueryRows(ctx, dataSQL, func(row map[string]any, columns []*dbi.QueryColumn) error {
		srcCount++
		return nil
	}); srcErr != nil {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = fmt.Sprintf("validation failed: source count query error: %s", srcErr.Error())
		app.appendRunLog(syncLog, "校验失败: 源库行数统计查询错误: %s", srcErr.Error())
		return
	}

	// 目标数据行数
	targetCountSQL := fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", targetTable)
	targetCount := 0
	if _, tgtErr := targetConn.WalkQueryRows(ctx, targetCountSQL, func(row map[string]any, columns []*dbi.QueryColumn) error {
		if cnt, ok := row["cnt"]; ok {
			targetCount = cast.ToInt(cnt)
		}
		return dbi.NewStopWalkQueryError("count done")
	}); tgtErr != nil {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = fmt.Sprintf("validation failed: target count query error: %s", tgtErr.Error())
		app.appendRunLog(syncLog, "校验失败: 目标库行数统计查询错误: %s", tgtErr.Error())
		return
	}

	// 行数对比
	if srcCount != targetCount {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = fmt.Sprintf("validation failed: source=%d, target=%d, diff=%d", srcCount, targetCount, srcCount-targetCount)
		syncLog.ResNum = srcCount
		metrics.TotalRows = srcCount
		app.appendRunLog(syncLog, "校验失败: 源=%d行，目标=%d行，差异=%d行", srcCount, targetCount, srcCount-targetCount)
		return
	}
	app.appendRunLog(syncLog, "行数校验通过: 源=%d行，目标=%d行", srcCount, targetCount)

	// 行数一致时，对比排序后的前 100 行映射字段原值。
	//
	// 配了实际转换规则时跳过内容比对：目标写入的列值是转换后的结果（upper、常量、表达式等），
	// 拿源原值与目标转换值直接比对会把“转换成功”误判为“内容不一致”，与同步写入语义相背。
	// 行数比对不依赖列内容，保留即可；内容比对需同层转换能力（目前不具备），宁缺勿误。
	//
	// 与 doDataSync 共享同一个构造入口 NewTransformEngine：“” / 非法 JSON / 空数组都归为无转换
	// （engine==nil）。非法 JSON 先失败返回，不默默“无转换”跟实际执行链路语义分叉。
	transformEngine, terr := NewTransformEngine(task.TransformRules)
	if terr != nil {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = fmt.Sprintf("validation failed: invalid TransformRules: %s", terr.Error())
		app.appendRunLog(syncLog, "校验失败: 字段转换规则解析错误: %s", terr.Error())
		return
	}
	if len(keyColumns) > 0 && srcCount > 0 && transformEngine != nil {
		app.appendRunLog(syncLog, "检测到字段转换规则，跳过内容比对（源原值与目标转换值天然不等），仅保留行数校验")
		logx.InfofContext(ctx, "data validation: skipping content check because TransformRules is configured (task=%d)", task.Id)
	} else if len(keyColumns) > 0 && srcCount > 0 {
		failValidation := func(errText, runMsg string) {
			syncLog.Status = entity.DataSyncTaskStateFail
			syncLog.ErrText = errText
			syncLog.ResNum = srcCount
			metrics.TotalRows = srcCount
			app.appendRunLog(syncLog, "%s", runMsg)
		}

		const checkSampleRows = 100
		// 两端分别排序抽样；不同排序规则可能选中不同样本，不能据此证明全量一致。
		// 先构造不含分页子句的基础 SQL，再按各自方言改写：源/目标可能异构，Oracle/DM/MSSQL 不识别裸 LIMIT。
		srcBaseCheckSQL := fmt.Sprintf("SELECT %s FROM (%s) _src ORDER BY %s",
			strings.Join(srcKeyCols, ", "), dataSQL, strings.Join(srcKeyCols, ", "))
		targetBaseCheckSQL := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s",
			strings.Join(keyColumns, ", "), targetTable, strings.Join(keyColumns, ", "))

		srcCheckSQL, pgErr := paginateTopN(srcConn, srcBaseCheckSQL, checkSampleRows)
		if pgErr != nil {
			failValidation(fmt.Sprintf("validation failed: build source checksum sql error: %s", pgErr.Error()),
				fmt.Sprintf("校验失败: 源端checksum SQL构建错误: %s", pgErr.Error()))
			return
		}
		targetCheckSQL, pgErr := paginateTopN(targetConn, targetBaseCheckSQL, checkSampleRows)
		if pgErr != nil {
			failValidation(fmt.Sprintf("validation failed: build target checksum sql error: %s", pgErr.Error()),
				fmt.Sprintf("校验失败: 目标端checksum SQL构建错误: %s", pgErr.Error()))
			return
		}

		// checksum 查询失败必须显式判失败：否则空结果会让两侧「相等」而假通过，掩盖跨方言执行错误
		srcKeys, srcCheckErr := collectKeyStrings(srcConn, ctx, srcCheckSQL)
		if srcCheckErr != nil {
			failValidation(fmt.Sprintf("validation failed: source checksum query error: %s", srcCheckErr.Error()),
				fmt.Sprintf("校验失败: 源端checksum查询错误: %s", srcCheckErr.Error()))
			return
		}
		targetKeys, tgtCheckErr := collectKeyStrings(targetConn, ctx, targetCheckSQL)
		if tgtCheckErr != nil {
			failValidation(fmt.Sprintf("validation failed: target checksum query error: %s", tgtCheckErr.Error()),
				fmt.Sprintf("校验失败: 目标端checksum查询错误: %s", tgtCheckErr.Error()))
			return
		}

		if len(srcKeys) != len(targetKeys) {
			failValidation(fmt.Sprintf("validation passed row count (%d) but checksum mismatch: source keys=%d, target keys=%d",
				srcCount, len(srcKeys), len(targetKeys)),
				fmt.Sprintf("校验失败: 行数一致但checksum数量不匹配(源%d个，目标%d个)", len(srcKeys), len(targetKeys)))
			return
		}
		// 逐行比较带列边界的映射值编码
		mismatch := false
		for i := range srcKeys {
			if srcKeys[i] != targetKeys[i] {
				mismatch = true
				break
			}
		}
		if mismatch {
			failValidation(fmt.Sprintf("validation passed row count (%d) but checksum content mismatch (first %d rows)", srcCount, checkSampleRows),
				fmt.Sprintf("校验失败: 行数一致但checksum内容不匹配(前%d行)", checkSampleRows))
			return
		}
		logx.InfofContext(ctx, "data validation: row count=%d, checksum passed (first %d rows match)", srcCount, checkSampleRows)
		app.appendRunLog(syncLog, "抽样校验通过: 前%d行映射字段原值匹配，不代表全部数据一致", len(srcKeys))
	}

	syncLog.Status = entity.DataSyncTaskStateSuccess
	syncLog.ErrText = fmt.Sprintf("validation passed: source=%d, target=%d", srcCount, targetCount)
	syncLog.ResNum = srcCount
	metrics.TotalRows = srcCount
	app.appendRunLog(syncLog, "数据校验完成: 源=%d行，目标=%d行", srcCount, targetCount)
}

func (app *DataSyncAppImpl) StopTask(ctx context.Context, taskId uint64) error {
	task := new(entity.DataSyncTask)
	task.Id = taskId
	task.RunningState = entity.DataSyncTaskRunStateStop
	if err := app.UpdateById(ctx, task); err != nil {
		return err
	}
	app.runGuard.Release(taskId)
	return nil
}

func (app *DataSyncAppImpl) InitCronJob() {
	ctx := contextx.WithTraceId(context.Background())

	defer func() {
		if err := recover(); err != nil {
			logx.ErrorTraceContext(ctx, "the data synchronization task failed to initialize", err)
		}
	}()

	_ = app.UpdateByCond(context.TODO(), &entity.DataSyncTask{RunningState: entity.DataSyncTaskRunStateReady}, &entity.DataSyncTask{RunningState: entity.DataSyncTaskRunStateRunning})

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

// 同步模块集中常量：消除魔法数字
const (
	// defaultSyncBatchSize 默认同步批次大小（PageSize <= 0 时使用）
	defaultSyncBatchSize = 500
	// defaultFailedRowCapacity 失败行记录器默认容量
	defaultFailedRowCapacity = 1000
	// runningSyncLogTTL 运行中日志缓存 TTL
	runningSyncLogTTL = time.Hour * 1
	// conflictBatchSize 冲突检测批量查询上限（防止 OR 条件过长）
	conflictBatchSize = 100
)

// GetDataSyncTaskApp 获取数据同步任务应用门面
func GetDataSyncTaskApp() DataSyncTask {
	return ioc.Get[DataSyncTask]()
}

// SyncBatch 单批同步执行（供集成测试和外部调用方使用）。
// 初始化完整的 syncExecContext，包含转换引擎、过滤引擎、失败行记录器等。
func (app *DataSyncAppImpl) SyncBatch(ctx context.Context, srcRes []map[string]any, fieldMap []map[string]string, updFieldName string, task *entity.DataSyncTask, targetConn *dbi.DbConn, targetColumns []dbi.Column, targetTableMeta *dbi.TargetTableMeta) error {
	sec := &syncExecContext{
		fieldMap:          fieldMap,
		updFieldName:      updFieldName,
		task:              task,
		targetConn:        targetConn,
		targetColumns:     targetColumns,
		targetTableMeta:   targetTableMeta,
		failedRowRecorder: NewFailedRowRecorder(defaultFailedRowCapacity),
		metrics:           NewSyncMetrics(),
	}
	return app.srcData2TargetDb(ctx, srcRes, sec)
}
