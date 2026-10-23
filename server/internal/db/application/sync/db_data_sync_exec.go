package sync

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"
)

// defaultFailedRowCapacity 失败行记录器默认容量
const defaultFailedRowCapacity = 1000

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
