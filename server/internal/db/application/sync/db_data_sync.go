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
	"mayfly-go/pkg/cache"
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

	GetTaskLogList(condition *entity.DataSyncLogQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncLog], error)
}

var _ (DataSyncTask) = (*DataSyncAppImpl)(nil)

type DataSyncAppImpl struct {
	base.AppImpl[*entity.DataSyncTask, repository.DataSyncTask]

	dbDataSyncLogRepo repository.DataSyncLog `inject:"T"`

	dbApp DbApp `inject:"T"`

	// runGuard 运行态原子守卫：替代原cache标记，修复IsRunning检查与标记间的竞态
	runGuard taskx.RunGuard[uint64]
}

var (
	whereReg = regexp.MustCompile(`(?i)where`)
)

func (app *DataSyncAppImpl) GetPageList(condition *entity.DataSyncTaskQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncTask], error) {
	return app.GetRepo().GetTaskList(condition, orderBy...)
}

func (app *DataSyncAppImpl) Save(ctx context.Context, taskEntity *entity.DataSyncTask) error {
	// 保存时校验DataSql与增量字段合法性（运行时Run内还会再校验，双重拦截）
	dialect, err := app.getSrcDialect(taskEntity)
	if err != nil {
		return err
	}
	if err := validateDataSyncSql(dialect, taskEntity); err != nil {
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

	// Phase 6.3: 双向同步 - 确保反向任务存在
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

	// Phase 6.3: 删除时同步清理关联的反向任务（防止循环引用导致无限递归）
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
	// 与 transfer 模块 Run 保持一致的占位优先策略
	if !app.runGuard.Acquire(id) {
		logx.WarnfContext(ctx, "[%d] the db sync task is running...", id)
		return errorx.NewBiz("the task is running")
	}

	task, err := app.GetById(id)
	if err != nil {
		app.runGuard.Release(id)
		return errorx.NewBiz("task not found")
	}

	logx.InfofContext(ctx, "start the data sync task: %s => %s", task.TaskName, task.TaskKey)

	updateStateTask := &entity.DataSyncTask{
		RunningState: entity.DataSyncTaskRunStateRunning,
	}
	updateStateTask.Id = id
	if err := app.UpdateById(ctx, updateStateTask); err != nil {
		app.runGuard.Release(id)
		return errorx.NewBizf("failed to update task running state: %s", err.Error())
	}

	// 提前创建日志记录并落库，使前端轮询可在执行期间查看实时日志
	now := time.Now()
	syncLog := &entity.DataSyncLog{
		TaskId:     task.Id,
		CreateTime: &now,
		Status:     entity.DataSyncTaskStateFail, // 默认失败，完成后更新
	}
	if err := app.dbDataSyncLogRepo.Save(ctx, syncLog); err != nil {
		app.runGuard.Release(id)
		return errorx.NewBizf("failed to create sync log: %s", err.Error())
	}
	// 缓存日志供 GetTaskLogList 实时读取 RunLog
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

		// 获取源库连接：同时用于运行时双重校验方言与后续增量字段类型探测
		srcConn, err := app.dbApp.GetDbConn(ctx, uint64(task.SrcDbId), task.SrcDbName)
		if err != nil {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			return
		}
		dialect := srcConn.GetDialect()
		// 运行时双重拦截：存量脏数据（校验引入前创建的任务）防御
		if verr := validateDataSyncSql(dialect, task); verr != nil {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", verr.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			return
		}

		// Phase 6.2: 数据校验模式
		if task.SyncMode == entity.DataSyncModeValidation {
			app.doDataValidation(ctx, task, syncLog, metrics)
			return
		}

		// 构建同步查询 SQL（增量条件、排序、软删除过滤等）
		sqlStr, err := app.buildSyncQuery(ctx, task, srcConn, dialect, syncLog)
		if err != nil {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			return
		}

		err = app.doDataSync(ctx, sqlStr, task, syncLog, metrics)
		if err != nil {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			syncLog.Status = entity.DataSyncTaskStateFail
		} else {
			syncLog.Status = entity.DataSyncTaskStateSuccess
		}
	})

	return nil
}

// buildSyncQuery 构建同步查询 SQL：根据同步模式组装增量条件、排序、软删除过滤等。
// 拆分自 Run 方法，降低单方法复杂度。
func (app *DataSyncAppImpl) buildSyncQuery(ctx context.Context, task *entity.DataSyncTask, srcConn *dbi.DbConn, dialect dbi.Dialect, syncLog *entity.DataSyncLog) (string, error) {
	// 根据同步模式决定是否需要增量条件
	isFullMode := task.SyncMode == entity.DataSyncModeFullRefresh

	// 增量合并/软删除/硬删除模式自动切为 UPSERT 策略（各方言原生语法）
	if task.SyncMode == entity.DataSyncModeIncrementalMerge ||
		task.SyncMode == entity.DataSyncModeIncrementalSoftDel ||
		task.SyncMode == entity.DataSyncModeIncrementalHardDel {
		task.DuplicateStrategy = dbi.DuplicateStrategyUpdate
	}

	// Phase 2: 多方言感知水位线格式化 + 多字段增量条件
	updSql := ""
	orderSql := ""
	if !isFullMode && task.UpdFieldVal != "0" && task.UpdFieldVal != "" && task.UpdField != "" {
		updFieldDataType := dbi.DefaultDbDataType
		_, probeErr := srcConn.WalkQueryRows(ctx, task.DataSql, func(row map[string]any, columns []*dbi.QueryColumn) error {
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
		formattedVal := updFieldDataType.DataType.SQLValue(task.UpdFieldVal)
		updSql = fmt.Sprintf("and %s > %s", task.UpdField, formattedVal)

		// Phase 2: 辅助增量字段（多字段增量条件）
		if task.UpdFieldSecondary != "" {
			updSql += fmt.Sprintf(" and %s > %s", task.UpdFieldSecondary, formattedVal)
		}
	}

	// 即使是首次同步，如果有更新字段也要添加排序，确保每次查询结果顺序一致
	if task.UpdField != "" {
		orderSql = "order by " + task.UpdField + " asc "
	}

	// 解析器判断DataSql是否已含where条件
	var where = "where 1=1"
	if dataSqlHasWhere(dialect, task) {
		where = ""
	}

	// Phase 4: Mode 4 软删除模式 - 过滤掉源端已软删除的记录
	var softDelFilter string
	if task.SyncMode == entity.DataSyncModeIncrementalSoftDel && task.SoftDeleteField != "" && task.SoftDeleteValue != "" {
		// 构建软删除过滤条件：排除软删除标记值等于指定值的记录
		// SoftDeleteValue 需转义单引号防止 SQL 注入（SoftDeleteField 已在 validateDataSyncSql 中白名单校验）
		escapedValue := strings.ReplaceAll(task.SoftDeleteValue, "'", "''")
		softDelFilter = fmt.Sprintf("and (%s != '%s' OR %s IS NULL)", task.SoftDeleteField, escapedValue, task.SoftDeleteField)
		app.appendRunLog(syncLog, "软删除模式: 过滤条件 [%s]", softDelFilter)
	}

	// DataSql尾部可能带分号
	dataSql := strings.TrimRight(strings.TrimSpace(task.DataSql), ";")

	// 组装查询sql
	sqlStr := fmt.Sprintf("%s %s %s %s %s", dataSql, where, updSql, softDelFilter, orderSql)
	syncLog.DataSqlFull = sqlStr

	return sqlStr, nil
}

func (app *DataSyncAppImpl) doDataSync(ctx context.Context, sql string, task *entity.DataSyncTask, syncLog *entity.DataSyncLog, metrics *SyncMetrics) error {
	app.appendRunLog(syncLog, "========================================")
	app.appendRunLog(syncLog, "开始执行同步任务: %s", task.TaskName)
	app.appendRunLog(syncLog, "========================================")
	app.appendRunLog(syncLog, "同步模式: %s", getSyncModeName(task.SyncMode))
	app.appendRunLog(syncLog, "目标表: %s.%s", task.TargetDbName, task.TargetTableName)
	app.appendRunLog(syncLog, "冲突策略: %s", getDuplicateStrategyName(task.DuplicateStrategy))
	if task.PageSize > 0 {
		app.appendRunLog(syncLog, "批次大小: %d 行", task.PageSize)
	}

	// 获取源数据库连接
	app.appendRunLog(syncLog, "----------------------------------------")
	app.appendRunLog(syncLog, "[1/6] 连接源数据库...")
	srcConn, err := app.dbApp.GetDbConn(ctx, uint64(task.SrcDbId), task.SrcDbName)
	if err != nil {
		return errorx.NewBizf("failed to connect to the source database: %s", err.Error())
	}
	app.appendRunLog(syncLog, "✓ 源库连接成功: %s", task.SrcDbName)

	// 获取目标数据库连接
	app.appendRunLog(syncLog, "[2/6] 连接目标数据库...")
	targetConn, err := app.dbApp.GetDbConn(ctx, uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		return errorx.NewBizf("failed to connect to the target database: %s", err.Error())
	}
	app.appendRunLog(syncLog, "✓ 目标库连接成功: %s", task.TargetDbName)

	// task.FieldMap为json数组字符串 [{"src":"id","target":"id"}]，转为map
	var fieldMap []map[string]string
	err = json.Unmarshal([]byte(task.FieldMap), &fieldMap)
	if err != nil {
		return errorx.NewBizf("there was an error parsing the field map json: %s", err.Error())
	}
	app.appendRunLog(syncLog, "字段映射: %d 个字段", len(fieldMap))

	// Phase 3: 初始化转换引擎和过滤引擎
	app.appendRunLog(syncLog, "[3/6] 初始化处理引擎...")
	transformEngine, err := NewTransformEngine(task.TransformRules)
	if err != nil {
		return errorx.NewBizf("failed to initialize transform engine: %s", err.Error())
	}
	if transformEngine != nil {
		app.appendRunLog(syncLog, "✓ 转换引擎已启用")
	}
	filterEngine := NewFilterEngine(task.FilterCondition)
	if filterEngine != nil {
		app.appendRunLog(syncLog, "✓ 过滤引擎已启用")
	}

	// Phase 6.3: 初始化冲突检测器（双向同步 + 配置了时间戳字段时启用）
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

	// Phase 5: Schema 演化检测（仅检测 fieldMap 映射的目标列是否在目标表中存在）
	app.appendRunLog(syncLog, "[5/6] 检测Schema变更...")
	schemaDetector := NewSchemaDetector(task.SchemaEvolveMode)
	if schemaDetector != nil {
		changes := schemaDetector.DetectChanges(fieldMap, targetTableColumns)
		if len(changes) > 0 {
			skipCols := schemaDetector.HandleChanges(nil, changes, task.TaskName)
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
	targetTableMeta := dbi.BuildTargetTableMeta(targetConn, task.TargetTableName, targetTableColumns)

	// 全量刷新模式：先清空目标表
	if task.SyncMode == entity.DataSyncModeFullRefresh {
		sqlGen := targetConn.GetDialect().GetSQLGenerator()
		truncateSqls := sqlGen.GenTruncate(task.TargetTableName)
		if len(truncateSqls) > 0 {
			for _, sql := range truncateSqls {
				if _, execErr := targetConn.ExecContext(ctx, sql); execErr != nil {
					return errorx.NewBizf("failed to truncate target table [%s]: %s", task.TargetTableName, execErr.Error())
				}
			}
			logx.InfofContext(ctx, "full refresh mode: truncated target table [%s]", task.TargetTableName)
			app.appendRunLog(syncLog, "全量刷新模式: 已清空目标表 [%s]", task.TargetTableName)
		} else {
			logx.WarnfContext(ctx, "full refresh mode: target dialect does not support TRUNCATE for table [%s], proceeding with upsert only", task.TargetTableName)
			app.appendRunLog(syncLog, "全量刷新模式: 目标方言不支持TRUNCATE，仅使用upsert")
		}
	}

	// 硬删除模式：收集所有源数据主键值
	var allSrcPKValuess [][]any
	isHardDelete := task.SyncMode == entity.DataSyncModeIncrementalHardDel
	// 软删除模式：同样需要收集源PK用于目标端清理
	isSoftDelete := task.SyncMode == entity.DataSyncModeIncrementalSoftDel

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

	_, err = srcConn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		// Phase 3: 过滤引擎 - 不满足条件的行直接跳过
		if !filterEngine.Match(row) {
			metrics.SkipCount++
			return nil
		}

		total++
		result = append(result, row)

		// 硬删除/软删除模式：收集源数据主键值
		if (isHardDelete || isSoftDelete) && len(targetTableMeta.UniqueColumns) > 0 {
			pkVals := make([]any, 0, len(targetTableMeta.UniqueColumns))
			for _, pkCol := range targetTableMeta.UniqueColumns {
				if v, ok := lookupRowValue(row, pkCol); ok {
					pkVals = append(pkVals, v)
				}
			}
			if len(pkVals) == len(targetTableMeta.UniqueColumns) {
				allSrcPKValuess = append(allSrcPKValuess, pkVals)
			}
		}

		// 按行数分批
		if total%batchSize == 0 {
			batchNum++
			batchStart := time.Now()
			if err := app.srcData2TargetDb(ctx, result, sec); err != nil {
				return err
			}

			if pwErr := app.persistUpdFieldVal(ctx, task); pwErr != nil {
				logx.WarnfContext(ctx, "watermark persist failed (batch #%d): %s", batchNum, pwErr.Error())
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

			if !app.runGuard.IsRunning(task.Id) {
				return errorx.NewBiz("the task has been terminated manually")
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
		if pwErr := app.persistUpdFieldVal(ctx, task); pwErr != nil {
			logx.WarnfContext(ctx, "watermark persist failed (final batch): %s", pwErr.Error())
		}
		metrics.RecordBatch(len(result))
	}

	// 硬删除/软删除模式：删除目标表中源库不存在的记录
	if (isHardDelete || isSoftDelete) && len(allSrcPKValuess) > 0 && len(targetTableMeta.UniqueColumns) > 0 {
		sqlGen := targetConn.GetDialect().GetSQLGenerator()
		deleteSqls := sqlGen.GenBatchDelete(task.TargetTableName, targetTableMeta.UniqueColumns, allSrcPKValuess, targetTableMeta)
		deleteFailed := false
		deleteSuccessPKs := 0
		// GenBatchDelete 按 batchSize 切分 PK 集合，每批生成一条 DELETE ... IN 语句
		for i, dsql := range deleteSqls {
			if _, execErr := targetConn.ExecContext(ctx, dsql); execErr != nil {
				logx.WarnfContext(ctx, "delete execution failed on table [%s]: %s", task.TargetTableName, execErr.Error())
				deleteFailed = true
				continue
			}
			// 计算本批实际包含的 PK 行数（最后一批可能不足 batchSize）
			batchStart := i * batchSize
			batchEnd := batchStart + batchSize
			if batchEnd > len(allSrcPKValuess) {
				batchEnd = len(allSrcPKValuess)
			}
			deleteSuccessPKs += batchEnd - batchStart
		}
		if len(deleteSqls) > 0 {
			modeName := "硬删除"
			if isSoftDelete {
				modeName = "软删除"
			}
			// 记录实际成功删除的 PK 数（每批 batchSize 个），而非源库 PK 总数
			metrics.DeleteCount += deleteSuccessPKs
			logx.InfofContext(ctx, "%s模式: 执行 %d 条批量删除语句，成功删除 %d 行，源库 PK 总数 %d",
				modeName, len(deleteSqls), deleteSuccessPKs, len(allSrcPKValuess))
			app.appendRunLog(syncLog, "  ✓ %s模式: 执行 %d 条批量删除语句，成功删除 %d 行", modeName, len(deleteSqls), deleteSuccessPKs)
			if deleteFailed {
				app.appendRunLog(syncLog, "  ⚠ 部分删除语句执行失败，请检查目标库状态")
			}
		}
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
}

// srcData2TargetDb 源数据到目标库的转换与写入。
// 编排三个子步骤：批量冲突检测+转换 → 事务写入 → 水位推进+指标更新。
func (app *DataSyncAppImpl) srcData2TargetDb(ctx context.Context, srcRes []map[string]any, sec *syncExecContext) (err error) {
	// Step 1: 批量冲突检测 + 转换引擎应用
	targetData := app.transformBatch(ctx, srcRes, sec)

	// Step 2: 生成 SQL 并事务写入目标库
	if err := app.executeTargetInsert(ctx, targetData, srcRes, sec); err != nil {
		return err
	}

	// Step 3: 水位推进 + 指标更新
	if watermarkErr := advanceSyncWatermark(srcRes, sec.task, sec.updFieldName); watermarkErr != nil {
		logx.WarnfContext(ctx, "failed to advance the data sync watermark of task [%d]: %s", sec.task.Id, watermarkErr.Error())
	}
	if sec.metrics != nil {
		switch sec.task.DuplicateStrategy {
		case dbi.DuplicateStrategyUpdate:
			sec.metrics.UpdateCount += len(targetData)
		default:
			sec.metrics.InsertCount += len(targetData)
		}
	}
	return nil
}

// transformBatch 对源数据批次应用冲突检测和转换规则，返回转换后的目标行集合。
func (app *DataSyncAppImpl) transformBatch(ctx context.Context, srcRes []map[string]any, sec *syncExecContext) []map[string]any {
	fieldMap := sec.fieldMap
	task := sec.task
	targetDbConn := sec.targetConn
	targetTableMeta := sec.targetTableMeta
	transformEngine := sec.transformEngine
	metrics := sec.metrics

	// 批量冲突检测 — 分批查询目标行，消除 N+1 查询
	conflictMap := app.batchLoadConflictTargetRows(ctx, srcRes, sec)

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
						pkWhere := buildPKWhereClause(srcData, target2Src, targetTableMeta, targetDbConn.GetDialect())
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
	return targetData
}

// executeTargetInsert 将转换后的目标数据生成 SQL 并在事务中写入目标库。
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

	targetDialect := targetDbConn.GetDialect()
	sqls := targetDialect.GetSQLGenerator().GenInsert(
		task.TargetTableName, targetInsertColumns, targetValues,
		cmp.Or(task.DuplicateStrategy, dbi.DuplicateStrategyNone), targetTableMeta)

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

	for _, sql := range sqls {
		if _, execErr := targetDbConn.TxExecContext(ctx, targetDbTx, sql); execErr != nil {
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

// doDataValidation Phase 6.2: 数据校验模式。
// 对比源/目标行数，并可选抽样 checksum 对比（按字段映射的主键列）。
func (app *DataSyncAppImpl) doDataValidation(ctx context.Context, task *entity.DataSyncTask, syncLog *entity.DataSyncLog, metrics *SyncMetrics) {
	app.appendRunLog(syncLog, "开始数据校验模式: 任务 [%s]", task.TaskName)

	srcConn, err := app.dbApp.GetDbConn(ctx, uint64(task.SrcDbId), task.SrcDbName)
	if err != nil {
		syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
		app.appendRunLog(syncLog, "源库连接失败: %s", err.Error())
		return
	}
	app.appendRunLog(syncLog, "源库连接成功")

	targetConn, err := app.dbApp.GetDbConn(ctx, uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
		app.appendRunLog(syncLog, "目标库连接失败: %s", err.Error())
		return
	}
	app.appendRunLog(syncLog, "目标库连接成功")

	// 从字段映射中提取主键列用于 checksum
	var keyColumns []string
	var srcKeyCols []string
	var fieldMap []map[string]string
	if err := json.Unmarshal([]byte(task.FieldMap), &fieldMap); err == nil {
		for _, fm := range fieldMap {
			keyColumns = append(keyColumns, fm["target"])
			srcKeyCols = append(srcKeyCols, fm["src"])
		}
	}

	// 源数据行数
	srcCount := 0
	if _, srcErr := srcConn.WalkQueryRows(ctx, task.DataSql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		srcCount++
		return nil
	}); srcErr != nil {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = fmt.Sprintf("validation failed: source count query error: %s", srcErr.Error())
		app.appendRunLog(syncLog, "校验失败: 源库行数统计查询错误: %s", srcErr.Error())
		return
	}

	// 目标数据行数
	targetCountSQL := fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", task.TargetTableName)
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

	// 行数一致时，进行抽样 checksum 对比（前 100 行主键值）
	if len(keyColumns) > 0 && srcCount > 0 {
		// 源端 checksum：在 DataSql 基础上限制前 100 行取映射的源列（ORDER BY 保证与目标端排序一致）
		srcCheckSQL := fmt.Sprintf("SELECT %s FROM (%s) _src ORDER BY %s LIMIT 100",
			strings.Join(srcKeyCols, ", "), task.DataSql, strings.Join(srcKeyCols, ", "))
		// 目标端 checksum：从目标表取前 100 行映射的目标列
		targetCheckSQL := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s LIMIT 100",
			strings.Join(keyColumns, ", "), task.TargetTableName, strings.Join(keyColumns, ", "))

		srcKeys := collectKeyStrings(srcConn, ctx, srcCheckSQL)
		targetKeys := collectKeyStrings(targetConn, ctx, targetCheckSQL)

		if len(srcKeys) != len(targetKeys) {
			syncLog.Status = entity.DataSyncTaskStateFail
			syncLog.ErrText = fmt.Sprintf("validation passed row count (%d) but checksum mismatch: source keys=%d, target keys=%d",
				srcCount, len(srcKeys), len(targetKeys))
			syncLog.ResNum = srcCount
			metrics.TotalRows = srcCount
			app.appendRunLog(syncLog, "校验失败: 行数一致但checksum数量不匹配(源%d个，目标%d个)", len(srcKeys), len(targetKeys))
			return
		}
		// 逐行比较主键拼接值
		mismatch := false
		for i := range srcKeys {
			if srcKeys[i] != targetKeys[i] {
				mismatch = true
				break
			}
		}
		if mismatch {
			syncLog.Status = entity.DataSyncTaskStateFail
			syncLog.ErrText = fmt.Sprintf("validation passed row count (%d) but checksum content mismatch (first 100 rows)", srcCount)
			syncLog.ResNum = srcCount
			metrics.TotalRows = srcCount
			app.appendRunLog(syncLog, "校验失败: 行数一致但checksum内容不匹配(前100行)")
			return
		}
		logx.InfofContext(ctx, "data validation: row count=%d, checksum passed (first 100 rows match)", srcCount)
		app.appendRunLog(syncLog, "checksum校验通过: 前100行主键值匹配")
	}

	syncLog.Status = entity.DataSyncTaskStateSuccess
	syncLog.ErrText = fmt.Sprintf("validation passed: source=%d, target=%d", srcCount, targetCount)
	syncLog.ResNum = srcCount
	metrics.TotalRows = srcCount
	app.appendRunLog(syncLog, "数据校验完成: 源=%d行，目标=%d行", srcCount, targetCount)
}

// collectKeyStrings 执行查询 SQL 并将每行的多列值拼接为单个字符串切片。
func collectKeyStrings(conn *dbi.DbConn, ctx context.Context, sql string) []string {
	var keys []string
	_, _ = conn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		parts := make([]string, 0, len(columns))
		for _, col := range columns {
			parts = append(parts, fmt.Sprintf("%v", row[col.Key]))
		}
		keys = append(keys, strings.Join(parts, "|"))
		return nil
	})
	return keys
}

// advanceSyncWatermark 推进同步水位
func advanceSyncWatermark(srcRes []map[string]any, task *entity.DataSyncTask, updFieldName string) error {
	if len(srcRes) == 0 {
		return nil
	}
	waterField := cmp.Or(task.UpdFieldSrc, updFieldName)
	if waterField == "" {
		return nil
	}
	val, ok := lookupRowValue(srcRes[len(srcRes)-1], waterField)
	if !ok {
		return fmt.Errorf("update field [%s] not found in the last synced row, keep watermark [%s] unchanged", waterField, task.UpdFieldVal)
	}
	if val == nil {
		return fmt.Errorf("update field [%s] of the last synced row is NULL, keep watermark [%s] unchanged", waterField, task.UpdFieldVal)
	}
	task.UpdFieldVal = cast.ToString(val)
	return nil
}

// lookupRowValue 从查询行中取列值
func lookupRowValue(row map[string]any, column string) (any, bool) {
	if column == "" {
		return nil, false
	}
	if val, ok := row[column]; ok {
		return val, true
	}
	for name, val := range row {
		if strings.EqualFold(name, column) {
			return val, true
		}
	}
	return nil, false
}

func (app *DataSyncAppImpl) persistUpdFieldVal(ctx context.Context, task *entity.DataSyncTask) error {
	if task.UpdField == "" {
		return nil
	}
	ut := &entity.DataSyncTask{Id: task.Id, UpdFieldVal: task.UpdFieldVal}
	if err := app.UpdateById(ctx, ut); err != nil {
		return fmt.Errorf("failed to persist the data sync watermark of task [%d]: %s", task.Id, err.Error())
	}
	return nil
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

func (app *DataSyncAppImpl) endRunning(taskEntity *entity.DataSyncTask, log *entity.DataSyncLog) {
	logx.InfoContext(context.Background(), log.ErrText)

	state := log.Status
	task := new(entity.DataSyncTask)
	task.Id = taskEntity.Id
	task.RecentState = state
	task.UpdFieldVal = taskEntity.UpdFieldVal
	task.RunningState = entity.DataSyncTaskRunStateReady
	if err := app.UpdateById(context.Background(), task); err != nil {
		logx.ErrorfContext(context.Background(), "failed to update sync task [%d] state after execution: %s", taskEntity.Id, err.Error())
	}
	app.saveLog(log)
	app.delRunningSyncLog(log.Id)
	app.runGuard.Release(task.Id)
}

func (app *DataSyncAppImpl) saveLog(log *entity.DataSyncLog) {
	if err := app.dbDataSyncLogRepo.Save(context.Background(), log); err != nil {
		logx.ErrorfContext(context.Background(), "failed to save sync log [%d]: %s", log.Id, err.Error())
	}
}

// getRunningSyncLog 获取运行中同步任务的缓存日志（供 GetTaskLogList 实时读取）
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

func (app *DataSyncAppImpl) InitCronJob() {
	ctx := contextx.WithTraceId(context.Background())

	defer func() {
		if err := recover(); err != nil {
			logx.ErrorTraceContext(ctx, "the data synchronization task failed to initialize", err)
		}
	}()

	_ = app.UpdateByCond(context.TODO(), &entity.DataSyncTask{RunningState: entity.DataSyncTaskRunStateReady}, &entity.DataSyncTask{RunningState: entity.DataSyncTaskRunStateRunning})

	if err := app.CursorByCond(&entity.DataSyncTaskQuery{Status: entity.DataSyncTaskStatusEnable}, func(dst *entity.DataSyncTask) error {
		app.addCronJob(ctx, dst)
		return nil
	}); err != nil {
		logx.ErrorTraceContext(ctx, "the db data sync task failed to initialize: %v", err)
	}
}

func (app *DataSyncAppImpl) GetTaskLogList(condition *entity.DataSyncLogQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncLog], error) {
	res, err := app.dbDataSyncLogRepo.GetTaskLogList(condition, orderBy...)
	if err != nil {
		return nil, err
	}
	// 如果查询的是最新日志（第一页），尝试从缓存获取最新的 RunLog
	if res != nil && len(res.List) > 0 && condition.PageNum <= 1 {
		latestLog := res.List[0]
		cachedLog := app.getRunningSyncLog(latestLog.Id)
		if cachedLog != nil && cachedLog.RunLog != "" {
			latestLog.RunLog = cachedLog.RunLog
		}
	}
	return res, nil
}

func (app *DataSyncAppImpl) addCronJob(ctx context.Context, taskEntity *entity.DataSyncTask) {
	key := taskEntity.TaskKey
	if taskEntity.Status != entity.DataSyncTaskStatusEnable {
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

func normalizeBatchSize(size int) int {
	if size <= 0 {
		return defaultSyncBatchSize
	}
	return size
}

// appendRunLog 向运行日志追加一行带时间戳的记录，并同步更新缓存以供前端实时查看。
// 格式：[HH:MM:SS.mmm] 消息内容\n
func (app *DataSyncAppImpl) appendRunLog(syncLog *entity.DataSyncLog, format string, args ...any) {
	if syncLog == nil {
		return
	}
	ts := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] %s\n", ts, fmt.Sprintf(format, args...))
	syncLog.RunLog += line
	// 同步更新缓存，供 GetTaskLogList 实时读取
	if syncLog.Id > 0 {
		app.setRunningSyncLog(syncLog.Id, syncLog)
	}
}

// getSyncModeName 返回同步模式的可读名称（仅用于 RunLog 日志输出，非前端展示）
func getSyncModeName(mode entity.DataSyncMode) string {
	switch mode {
	case entity.DataSyncModeFullRefresh:
		return "全量刷新"
	case entity.DataSyncModeIncrementalAppend:
		return "增量追加"
	case entity.DataSyncModeIncrementalMerge:
		return "增量合并"
	case entity.DataSyncModeIncrementalSoftDel:
		return "软删除模式"
	case entity.DataSyncModeIncrementalHardDel:
		return "硬删除模式"
	case entity.DataSyncModeValidation:
		return "数据校验"
	default:
		return fmt.Sprintf("未知模式(%d)", mode)
	}
}

// getDuplicateStrategyName 返回冲突策略的可读名称（仅用于 RunLog 日志输出，非前端展示）
func getDuplicateStrategyName(strategy int) string {
	switch strategy {
	case dbi.DuplicateStrategyIgnore:
		return "忽略"
	case dbi.DuplicateStrategyUpdate:
		return "更新"
	default:
		return fmt.Sprintf("未知策略(%d)", strategy)
	}
}

// truncateString 截断字符串，超过长度时添加省略号
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

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
