package sync

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dbi/value"
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

// shardQueryBudget 单条源查询的存活预算：超出即在批边界主动中断该查询，并按分片键续扫。
//
// 为什么需要它：源查询在服务端存活的是「整条查询的时长」，而每一批的目标写入就发生在读取回调里，
// 于是「一条查询 = 该片的读 + 该片的写」。写入速率远低于读取时（写慢的入库/宽表/索引多），
// 再小的片也可能超过源库的 query_timeout（StarRocks/Doris 默认 300s，且被 kill 的报告
// `Error 1105 ... query is timeout` 要到下一次读行才暴露）。只有把「查询存活时长」本身压到预算内，
// 才与写入速率解耦；到点后按最后读到的分片键续扫，读取位置不丢不重。
//
// 30s 的取值：比常见 query_timeout（30/60/300s）低一个数量级，给「最后一批写入 + 续扫语句规划」
// 留足余量；同时不会把一轮大表同步切成过碎的查询。变量而非常量仅为集成测试能缩短它。
var shardQueryBudget = 30 * time.Second

// budgetCheckRows 存活预算的检查粒度（行）：逐行取时钟在千万行级别上是白烧，按间隔检查让单条查询
// 最多超出「一个间隔的读取时长」。
const budgetCheckRows = 512

func (app *DataSyncAppImpl) doDataSync(ctx context.Context, plan *syncQueryPlan, task *entity.DataSyncTask, srcConn, targetConn *dbi.DbConn, syncLog *entity.DataSyncLog, metrics *SyncMetrics) error {
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

	// 全量刷新会先清空目标表（TRUNCATE 是 DDL，MySQL/PG 下不可回滚），所以必须先确认源查询真的可执行。
	// 否则源侧断连、权限被回收、SQL 运行期报错时，目标表已被清空且无法撤销 ——
	// "目标全空 + 任务失败"是不可逆损失，而这次读取只是验证可读性（读首行即停）。
	if task.SyncMode == entity.DataSyncModeFullRefresh {
		probeSQL := plan.sqlFor("")
		if rewritten, rwErr := srcConn.GetDialect().GetSQLParser().RewritePagination(probeSQL, 0, 1); rwErr == nil && strings.TrimSpace(rewritten) != "" {
			probeSQL = rewritten
		}
		_, probeErr := srcConn.WalkQueryRows(ctx, probeSQL, func(row map[string]any, columns []*dbi.QueryColumn) error {
			return dbi.NewStopWalkQueryError("source probe: first row read")
		})
		if probeErr != nil {
			return errorx.NewBizf("full refresh aborted: source query is not readable, target table was NOT cleared: %s", probeErr.Error())
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

	// 如果有数据库别名，则从UpdField中去掉数据库别名
	updFieldName := task.UpdField
	if task.UpdField != "" && strings.Contains(task.UpdField, ".") {
		updFieldName = strings.Split(task.UpdField, ".")[1]
	}

	app.appendRunLog(syncLog, "[6/6] 开始执行数据同步...")

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
		schemaDetector:    schemaDetector,
		conflictDetector:  conflictDetector,
		metrics:           metrics,
		failedRowRecorder: failedRowRecorder,
	}

	scanner := newSyncSourceScanner(app, ctx, task, srcConn, syncLog, sec)

	// 主键分片扫描：能分片时把「一整轮只发一次的大 SELECT」拆成 N 条按主键闭区间限定的短查询。
	// 不可分片时 conditions 为单个空串，行为与历史一致（一条流式大查询）。
	scan := planSourceScan(ctx, srcConn, task)
	app.appendRunLog(syncLog, "源读取方式: %s", scan.note)
	app.appendRunLog(syncLog, "执行SQL: %s", truncateString(plan.sqlFor(""), 200))
	if err := scanner.runShards(plan, scan); err != nil {
		return err
	}
	total, batchNum := scanner.total, scanner.batchNum

	// 整轮成功：无条件推进一次水位（含「数据正好整除、没有尾批」的情况）。
	// 批间不再推进（见上方说明），若这里也只放在尾批分支，整除时会永远不推进 → 每轮重放。
	if app.runGuard.IsCurrentRun(task.Id, syncLog.RunId) && !app.runGuard.IsStopRequested(task.Id) {
		if pwErr := app.persistUpdFieldVal(ctx, task); pwErr != nil {
			logx.WarnfContext(ctx, "watermark persist failed (final batch): %s", pwErr.Error())
		}
	} else {
		logx.WarnfContext(ctx, "sync task [%d] was preempted or stop-requested, skipping watermark persist (final batch)", task.Id)
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

// syncSourceScanner 一轮源读取的批次编排。
//
// 从 doDataSync 里收出来的原因：主键分片扫描下同一轮要跑 N 条源查询，若把批次体复制进分片循环
// 就会出现两套「攒批→写目标→计数/日志/停止判定」实现，行为极易分叉。收成一份后，
// 单次流式与分片扫描只是 runShards 入参不同。
type syncSourceScanner struct {
	app     *DataSyncAppImpl
	ctx     context.Context
	task    *entity.DataSyncTask
	srcConn *dbi.DbConn
	syncLog *entity.DataSyncLog
	sec     *syncExecContext

	batchSize int
	// 轮级累计：跨分片不重置，否则行数/批次号与目标写入粒度会在片间从头再来
	total         int
	batchNum      int
	result        []map[string]any
	lastBatchTime time.Time
	// schemaChecked 源列缺失检测只需在整轮首个到达的行上做一次（后续行与后续分片列形状相同）
	schemaChecked bool

	// shardKeyLabel 本轮分片键在结果集里的列标签；空 = 不可续扫（整表查询或取不到标签）
	shardKeyLabel string
	// lastKeyVal 当前查询里已读且已冲批的最后一行的分片键值（续扫游标）；每换一条查询重置
	lastKeyVal any
	// queryStartedAt 当前查询的发出时刻（存活预算从这里起算）
	queryStartedAt time.Time
	// readRows 当前查询已读行数（含被过滤掉的行）
	readRows int
	// nextBudgetCheck 下一次检查存活预算的已读行数阀值
	nextBudgetCheck int
	// queryBudgetHit 当前查询是否因超预算被主动中断（由 runShard 决定续扫还是收尾）
	queryBudgetHit bool
}

func newSyncSourceScanner(app *DataSyncAppImpl, ctx context.Context, task *entity.DataSyncTask, srcConn *dbi.DbConn, syncLog *entity.DataSyncLog, sec *syncExecContext) *syncSourceScanner {
	return &syncSourceScanner{
		app:           app,
		ctx:           ctx,
		task:          task,
		srcConn:       srcConn,
		syncLog:       syncLog,
		sec:           sec,
		batchSize:     normalizeBatchSize(task.PageSize),
		result:        make([]map[string]any, 0),
		lastBatchTime: time.Now(),
	}
}

// runShards 按分片计划逐条执行源查询：shards 为单个空串即整表单次查询（历史行为）。
//
// 片尾必须冲掉未满批：尾部不足 PageSize 的剩余行只能靠这里写出；同时每一片的数据都以
// 自己的目标侧事务批次提交，不会出现「一片的行跟下一片挤在同一事务里」这种更难推理的归属。
func (s *syncSourceScanner) runShards(plan *syncQueryPlan, scan sourceScanPlan) error {
	for i, cond := range scan.shards {
		if err := s.runShard(plan, scan, cond); err != nil {
			return err
		}
		// 只有一片时整轮 SQL 已在上面打过，不重复刷一条同样内容
		if len(scan.shards) > 1 {
			// 每片只一行且只带区间、不重打整条 SQL：片 SQL 与整轮 SQL 只差一个谓词尾段，
			// 上千片时 200 字符的重复文本会先把 RunLog 撑到上限（已被 appendRunLog 裁剪，
			// 但日志变短本身就是信息损失），区间落在这一行里足够定位
			s.app.appendRunLog(s.syncLog, "  ✓ 分片 %d/%d 读取完成 [%s]，累计 %d 行", i+1, len(scan.shards), cond, s.total)
			s.app.saveLog(s.syncLog)
			// 分片边界是真正的源侧停止点：上一条短查询已结束、下一条尚未发出，此时被抢占/停止就停下不会跨片双写。
			// 末片不判：数据已全部读完并写完，此处报「被终止」只会让水位不推进、不做删除对账，
			// 下一轮整轮重放——追加模式下就是重复行，比「本轮当成完成」严格更差。
			if i+1 < len(scan.shards) && !s.alive() {
				return errorx.NewBiz("the task has been terminated or preempted at shard boundary")
			}
		}
	}
	return nil
}

// runShard 读一个分片区间；查询存活超出预算时按分片键续扫，直到该区间读完。
//
// 续扫的正确性依赖两个前提，缺一不续（宁可回到「查询可能超时」也不静默少读）：
//  1. 分片键由数据库保证唯一（主键或单列唯一索引，见 shard.DetectIntShardKey）——`key > 游标`
//     才不会把同键值的未读行跳过去；
//  2. 结果按该键升序（见 sqlForOrder）——已读行构成键空间的**前缀**，游标之后的正好是未读部分。
func (s *syncSourceScanner) runShard(plan *syncQueryPlan, scan sourceScanPlan, baseCond string) error {
	resumable := scan.shardKey != "" && scan.keyLabel != ""
	order := shardScanOrder(resumable, s.srcConn.GetDialect().Quoter().QuoteIdent, scan.shardKey)
	s.shardKeyLabel = scan.keyLabel
	cond := baseCond
	for {
		s.beginQuery()
		if err := s.walk(plan.sqlForOrder(cond, order)); err != nil {
			return err
		}
		if err := s.flushBatch(); err != nil {
			return err
		}
		if !resumable || !s.queryBudgetHit {
			return nil
		}
		// 续扫谓词永远从**本片的原始区间**重建（不在上一条的 cond 上继续追加）：
		// 追加式拼接会让谓词随续扫次数线性变长（O(n²) 文本），且一旦某条谓词写错就无法回退。
		next, ok := continuationCond(baseCond, scan.shardKey, s.srcConn.GetDialect().Quoter().QuoteIdent, s.lastKeyVal)
		if !ok {
			// 读到了行却拿不到可续扫的整型游标：宁可报错，也不静默只读半个区间
			return errorx.NewBizf("shard scan cannot resume on key [%s]: last read value %v is not an integer", scan.shardKey, s.lastKeyVal)
		}
		s.app.appendRunLog(s.syncLog, "  ↻ 片内查询超预算 %ds，从 [%s] 续扫，累计 %d 行", int(shardQueryBudget.Seconds()), next, s.total)
		s.app.saveLog(s.syncLog)
		cond = next
		// 续扫边界与分片边界同性质：上一条查询已结束、下一条尚未发出，此时被抢占/停止停下不会双写
		if !s.alive() {
			return errorx.NewBiz("the task has been terminated or preempted at shard resume boundary")
		}
	}
}

// shardScanOrder 返回一个分片查询应使用的排序片段：
// 可续扫时**必须**按分片键升序（已读行构成键空间前缀，`key > 游标` 才能证明无损）；
// 不可续扫时返回空串，沿用默认的「按增量字段排序」（与历史行为一致）。
//
// 单独成函数的原因：这条排序是「续扫不丢行」的**充分条件**，必须能被单测直接铉住；
// 写在 runShard 内联分支里，删除/改错只能靠真实库的偶发漏行发现。
func shardScanOrder(resumable bool, quote func(string) string, shardKey string) string {
	if !resumable || shardKey == "" {
		return ""
	}
	return "order by " + quote(shardKey) + " asc"
}

// beginQuery 为一条新源查询重置预算窗口与续扫游标。
func (s *syncSourceScanner) beginQuery() {
	s.queryStartedAt = time.Now()
	s.readRows = 0
	s.nextBudgetCheck = s.budgetStep()
	s.queryBudgetHit = false
	s.lastKeyVal = nil
}

// budgetStep 存活预算的检查步长（行）：取「固定上限」与「一个批次」的较小值。
//
// 只看固定上限会在批次比它大时（PageSize 调大）把停止点推后一个整批，
// 于是一条查询的存活时长可多出「一批的读取+写入」；按批大小对齐后，
// 每条查询最多多活一个小批，与 PageSize 无关。
func (s *syncSourceScanner) budgetStep() int {
	if s.batchSize > 0 && s.batchSize < budgetCheckRows {
		return s.batchSize
	}
	return budgetCheckRows
}

// continuationCond 把「最后读到的分片键值」拼成下一条查询的区间片段。
// 返回 false 表示该值不可归一为整数，无法形成严格前进的续扫点。
func continuationCond(baseCond, shardKey string, quote func(string) string, lastKey any) (string, bool) {
	keyVal, ok := value.ValToInt64(lastKey)
	if !ok {
		return "", false
	}
	next := fmt.Sprintf("%s > %d", quote(shardKey), keyVal)
	if strings.TrimSpace(baseCond) == "" {
		return next, true
	}
	return baseCond + " AND " + next, true
}

// walk 遍历一条源查询的结果集，逐行交给 handleRow
func (s *syncSourceScanner) walk(querySQL string) error {
	_, err := s.srcConn.WalkQueryRows(s.ctx, querySQL, s.handleRow)
	return err
}

// handleRow 行级处理：源列检测（一次）→ 过滤 → 攒批，满批则写出并做批边界停止判定与可打断 sleep
func (s *syncSourceScanner) handleRow(row map[string]any, columns []*dbi.QueryColumn) error {
	// 首行到达时用真实源查询列做一次源列缺失检测（源列缺失会静默产生 NULL）
	if !s.schemaChecked {
		s.schemaChecked = true
		if s.sec.filterEngine != nil {
			if err := s.sec.filterEngine.root.validateFields(row); err != nil {
				return err
			}
		}
		if s.sec.schemaDetector != nil && len(columns) > 0 {
			srcColNames := make([]string, 0, len(columns))
			for _, c := range columns {
				srcColNames = append(srcColNames, c.Name)
			}
			if srcChanges := s.sec.schemaDetector.DetectSourceColumnChanges(s.sec.fieldMap, srcColNames); len(srcChanges) > 0 {
				s.sec.schemaDetector.HandleChanges(srcChanges, s.task.TaskName)
				s.syncLog.SchemaChanges += len(srcChanges)
				s.app.appendRunLog(s.syncLog, "⚠ 源列缺失检测: %d 个映射源列不存在于源查询结果，对应目标列将写入 NULL", len(srcChanges))
			}
		}
	}

	// 片内查询存活预算：必须在把本行记入续扫游标**之前**判。到这里时 s.result 里是本行之前的
	// 全部已读行，冲批后它们都已写入；游标停在上一行，续扫从该值之后读起，当前行会被重新读到、
	// 恰好写一次（重读无害，漏读不可恢复）。
	if s.shardKeyLabel != "" && s.readRows >= s.nextBudgetCheck {
		// 步长与首次检查同口径（budgetStep）：否则首次按批对齐、后续按固定 512，
		// 大批次下查询存活时长会多出一个固定步长的读+写，与「按批对齐」的声明不一致
		s.nextBudgetCheck = s.readRows + s.budgetStep()
		// 游标为空说明本查询还没读到可续扫的行（NULL 键片、取不到标签）——中断也无处续，只能读完
		if s.lastKeyVal != nil && time.Since(s.queryStartedAt) > shardQueryBudget {
			if err := s.flushBatch(); err != nil {
				return err
			}
			s.queryBudgetHit = true
			return dbi.NewStopWalkQueryError("shard query budget exceeded, resume from the last read shard key")
		}
	}
	s.readRows++
	// 续扫游标：读到即算（被过滤掉的行也属于已扫过的位置）
	if s.shardKeyLabel != "" {
		if v, ok := lookupRowValue(row, s.shardKeyLabel); ok {
			s.lastKeyVal = v
		}
	}

	// 过滤引擎 - 不满足条件的行直接跳过
	if !s.sec.filterEngine.Match(row) {
		s.sec.metrics.SkipCount++
		return nil
	}

	s.total++
	s.result = append(s.result, row)

	if len(s.result) < s.batchSize {
		return nil
	}
	if err := s.flushBatch(); err != nil {
		return err
	}

	// 以 runId 而非任务级锁存在性判断“本次执行是否仍有效”，避免新实例已接管时旧任务继续写入：
	// 旧逻辑 IsRunning(taskId) 在新实例 Acquire 后仍为 true，会默默多跑一批。
	// 批边界停止判定合并两个信号：
	// - IsCurrentRun：本地/Redis 归属，捕获"TTL 过期被他实例接管"
	// - IsStopRequested：Redis 停止标记，捕获"他实例调 StopTask"（本地 fast path 无法感知）
	if !s.alive() {
		return errorx.NewBiz("the task has been terminated or preempted")
	}

	// 批间可打断 sleep：定位为“目标侧节流”（缓解 replica lag / WAL apply 积压）。
	// 单次流式读源时对源库几乎无效（一条长 SELECT 从头到尾都在跑）；开启主键分片扫描后，
	// 真正的源侧间歇是分片之间（见 runShards），这里只影响写入节奏。tick 内检查归属，被抢占时立刻中止。
	if sleepMs := s.task.GetSleepBetweenBatchesMs(); sleepMs > 0 {
		sleepDur := time.Duration(sleepMs) * time.Millisecond
		if !interruptibleSleep(sleepDur, 100*time.Millisecond, s.alive) {
			return errorx.NewBiz("the task has been terminated or preempted during batch sleep")
		}
	}
	return nil
}

// alive 本次执行是否仍归本实例持有（未被抢占、也未收到停止请求）
func (s *syncSourceScanner) alive() bool {
	return s.app.runGuard.IsCurrentRun(s.task.Id, s.syncLog.RunId) && !s.app.runGuard.IsStopRequested(s.task.Id)
}

// flushBatch 将已攒下的行作为一个批次写入目标库，并更新轮级计数、指标与运行日志。
// 空缓冲直接返回（分片尾与整轮尾两处冲批调用点无需各自预判长度）。
func (s *syncSourceScanner) flushBatch() error {
	if len(s.result) == 0 {
		return nil
	}
	s.batchNum++
	batchStart := time.Now()
	if err := s.app.srcData2TargetDb(s.ctx, s.result, s.sec); err != nil {
		return err
	}
	// 水位**不在批间持久化**，只在本轮全部批次成功后推进一次（见 doDataSync 末尾说明）。
	//
	// 原因：增量续跑条件是 `upd {>,>=} V`。若批间就把 V 推进到「本批末行的 upd 值」，
	// 一旦在本批之后失败/停止/被抢占，下一轮以 `> V` 续跑时，与 V 同值但位于后续批次
	// 尚未写入的行会被永久跳过（UpdField 非唯一时必然发生，例如秒级时间戳）。
	//
	// 取舍：改为「至少一次」语义 —— 中断后整轮重放，重复由目标侧的冲突策略
	// （Ignore/Update/upsert）或主键约束显式暴露，而不是静默少写数据。
	// 正确性优先于重放成本；纯 Append 且目标无主键时的重复是有意的 fail-loud 行为。
	now := time.Now()
	batchDuration := now.Sub(batchStart).Milliseconds()
	if elapsed := now.Sub(s.lastBatchTime).Seconds(); elapsed > 0 {
		logx.DebugfContext(s.ctx, "data sync batch rate: %.0f rows/s", float64(len(s.result))/elapsed)
	}
	s.lastBatchTime = now

	s.sec.metrics.RecordBatch(len(s.result))

	s.syncLog.ErrText = i18n.T(imsg.DataSyncingMsg, "count", s.total)
	logx.InfoContext(s.ctx, s.syncLog.ErrText)
	s.syncLog.ResNum = s.total
	s.app.appendRunLog(s.syncLog, "  ✓ 批次 #%d 完成: %d 行, 耗时 %dms", s.batchNum, len(s.result), batchDuration)
	s.app.saveLog(s.syncLog)

	s.result = s.result[:0]
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
	schemaDetector    *SchemaDetector
	conflictDetector  *ConflictDetector
	metrics           *SyncMetrics
	failedRowRecorder *FailedRowRecorder
	reconcileKeys     [][]any
	// watermarkVal 本轮已读行的增量字段最大值（原始扫描值，非字符串）。
	// 分片扫描下各片末行不再单调，水位必须跨片取最大；落库仍只在整轮成功后一次（见 doDataSync）。
	watermarkVal any
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
	if watermarkErr := advanceSyncWatermark(srcRes, sec); watermarkErr != nil {
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
						pkWhere := buildPKWhereClause(srcData, target2Src, targetTableMeta, escapeFnForDialect(targetDbConn.Info.Type), targetDbConn.GetDialect().Quoter().QuoteIdent)
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
			// 转换规则失败被计为 skip：必须在这里把错误提出来让本轮失败。
			// 否则表现为"任务成功、但目标列少写了一行/若干行"，比直接写 NULL 更难发现。
			if transformEngine != nil {
				if tErr := transformEngine.Err(); tErr != nil {
					return nil, tErr
				}
			}
			continue
		}
		if transformEngine != nil {
			if tErr := transformEngine.Err(); tErr != nil {
				return nil, tErr
			}
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
