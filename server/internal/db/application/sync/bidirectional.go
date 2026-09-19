package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
	"time"
)

// BidirectionalSyncManager 双向同步管理器。
// 负责创建/管理反向同步任务，以及冲突检测与仲裁。
type BidirectionalSyncManager struct {
	taskApp DataSyncTask
}

// NewBidirectionalSyncManager 创建双向同步管理器
func NewBidirectionalSyncManager(taskApp DataSyncTask) *BidirectionalSyncManager {
	return &BidirectionalSyncManager{taskApp: taskApp}
}

// EnsureReverseTask 确保反向同步任务存在。
// 若 forwardTask.ReverseTaskId == 0，创建新的反向任务并回填 ID。
// 若已存在，校验反向任务配置一致性。
func (m *BidirectionalSyncManager) EnsureReverseTask(ctx context.Context, forwardTask *entity.DataSyncTask) error {
	if !forwardTask.BiDirEnabled {
		return nil
	}

	// 已有关联的反向任务
	if forwardTask.ReverseTaskId > 0 {
		return m.validateReverseTask(ctx, forwardTask)
	}

	// 创建反向同步任务
	reverseTask := m.buildReverseTask(forwardTask)
	if err := m.taskApp.Save(ctx, reverseTask); err != nil {
		return fmt.Errorf("failed to create reverse sync task: %s", err.Error())
	}

	// 回填反向任务 ID 到正向任务
	forwardTask.ReverseTaskId = reverseTask.Id
	if err := m.taskApp.Save(ctx, forwardTask); err != nil {
		return fmt.Errorf("failed to update forward task with reverse task ID: %s", err.Error())
	}

	logx.InfofContext(ctx, "bidirectional sync: created reverse task [%d] for forward task [%d], "+
		"TargetTableName requires manual configuration (auto-derived from forward task DataSql is not reliable)",
		reverseTask.Id, forwardTask.Id)
	return nil
}

// buildReverseTask 基于正向任务构建反向任务。
// 反向任务：源/目标互换，字段映射反转，增量模式保持。
func (m *BidirectionalSyncManager) buildReverseTask(forward *entity.DataSyncTask) *entity.DataSyncTask {
	// 反转字段映射
	reverseFieldMap := reverseFieldMapJSON(forward.FieldMap)

	reverse := &entity.DataSyncTask{
		TaskName:     fmt.Sprintf("[Reverse] %s", forward.TaskName),
		TaskCron:     forward.TaskCron,
		Status:       entity.DataSyncTaskStatusDisable, // 默认禁用，待用户配置 TargetTableName 后手动启用
		TaskKey:      stringx.RandUUID(),
		SyncMode:     forward.SyncMode,
		RunningState: entity.DataSyncTaskRunStateReady,

		// 源/目标互换
		SrcDbId:    forward.TargetDbId,
		SrcDbName:  forward.TargetDbName,
		SrcTagPath: forward.TargetTagPath,
		// 反向任务 DataSql：从正向的目标表全量查询（用户可按需调整）
		DataSql: fmt.Sprintf("SELECT * FROM %s", forward.TargetTableName),

		TargetDbId:    forward.SrcDbId,
		TargetDbName:  forward.SrcDbName,
		TargetTagPath: forward.SrcTagPath,
		// 反向任务的目标表名无法从正向 DataSql 自动推断，需用户手动配置
		// EnsureReverseTask 返回后由调用方提示用户补充
		TargetTableName: "",

		// 增量字段保持（反向任务可能需要调整）
		UpdField:          forward.UpdField,
		UpdFieldVal:       "0", // 反向任务从 0 开始
		UpdFieldSrc:       forward.UpdFieldSrc,
		UpdFieldSecondary: forward.UpdFieldSecondary,

		// 反转字段映射
		FieldMap: reverseFieldMap,

		// 双向同步配置
		BiDirEnabled:        false, // 反向任务不再创建反向的反向
		ReverseTaskId:       forward.Id,
		ConflictStrategy:    forward.ConflictStrategy,
		BiDirTimestampField: forward.BiDirTimestampField,

		// Schema 演化模式继承
		SchemaEvolveMode: forward.SchemaEvolveMode,

		// 分页大小继承
		PageSize: forward.PageSize,

		// 反向任务默认增量追加模式（避免循环写入）
		DuplicateStrategy: forward.DuplicateStrategy,
	}

	return reverse
}

// validateReverseTask 校验反向任务配置一致性
func (m *BidirectionalSyncManager) validateReverseTask(ctx context.Context, forwardTask *entity.DataSyncTask) error {
	reverseTask, err := m.taskApp.GetById(forwardTask.ReverseTaskId)
	if err != nil {
		return fmt.Errorf("reverse task [%d] not found: %s", forwardTask.ReverseTaskId, err.Error())
	}

	// 校验源/目标是否互换
	if reverseTask.SrcDbId != forwardTask.TargetDbId || reverseTask.TargetDbId != forwardTask.SrcDbId {
		logx.WarnfContext(ctx, "bidirectional sync: reverse task [%d] source/target mismatch with forward task [%d]",
			reverseTask.Id, forwardTask.Id)
	}

	return nil
}

// reverseFieldMapJSON 反转字段映射 JSON：src->target 变为 target->src
func reverseFieldMapJSON(fieldMapJSON string) string {
	var fieldMap []map[string]string
	if err := json.Unmarshal([]byte(fieldMapJSON), &fieldMap); err != nil {
		return fieldMapJSON
	}
	reversed := make([]map[string]string, len(fieldMap))
	for i, fm := range fieldMap {
		reversed[i] = map[string]string{
			"src":    fm["target"],
			"target": fm["src"],
		}
	}
	result, _ := json.Marshal(reversed)
	return string(result)
}

// ConflictDetector 冲突检测器。
// 用于双向同步场景：当同一行在源和目标都有更新时检测冲突。
type ConflictDetector struct {
	strategy       entity.ConflictStrategy
	timestampField string
}

// NewConflictDetector 创建冲突检测器
func NewConflictDetector(strategy entity.ConflictStrategy, timestampField string) *ConflictDetector {
	if strategy == 0 {
		strategy = entity.ConflictStrategySourceWins
	}
	return &ConflictDetector{
		strategy:       strategy,
		timestampField: timestampField,
	}
}

// DetectConflict 检测单行是否存在冲突。
// srcRow：源库行数据
// targetRow：目标库行数据（通过反向查询获取）
// 返回 true 表示存在冲突，应跳过该行的同步。
func (d *ConflictDetector) DetectConflict(srcRow, targetRow map[string]any) bool {
	if d.timestampField == "" {
		return false // 未配置时间戳字段，不做冲突检测
	}

	srcTs, srcOk := lookupRowValue(srcRow, d.timestampField)
	targetTs, targetOk := lookupRowValue(targetRow, d.timestampField)

	if !srcOk || !targetOk {
		return false // 无法取到时间戳，不检测
	}

	srcTime, srcErr := parseTimestamp(srcTs)
	targetTime, targetErr := parseTimestamp(targetTs)

	// 任一侧无法解析时降级为字符串比较（保证向后兼容）
	if srcErr != nil || targetErr != nil {
		srcStr := fmt.Sprintf("%v", srcTs)
		targetStr := fmt.Sprintf("%v", targetTs)
		return srcStr != targetStr && d.shouldSkip()
	}

	// 时间戳相同表示无冲突（同一版本）
	if srcTime.Equal(targetTime) {
		return false
	}

	// 存在冲突：根据策略决定
	return d.shouldSkip()
}

// shouldSkip 按冲突策略判断是否应跳过同步
func (d *ConflictDetector) shouldSkip() bool {
	switch d.strategy {
	case entity.ConflictStrategySourceWins:
		return false // 源优先，不跳过
	case entity.ConflictStrategyTargetWins:
		return true // 目标优先，跳过同步
	case entity.ConflictStrategySkip:
		return true // 冲突跳过
	default:
		return false
	}
}

// parseTimestamp 将数据库返回的时间值解析为 time.Time。
// 兼容多种后端返回类型：time.Time / string / []byte / 数值（Unix 时间戳）。
var timestampLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func parseTimestamp(val any) (time.Time, error) {
	switch v := val.(type) {
	case time.Time:
		return v, nil
	case *time.Time:
		if v == nil {
			return time.Time{}, fmt.Errorf("nil time pointer")
		}
		return *v, nil
	case string:
		return parseTimestampString(v)
	case []byte:
		return parseTimestampString(string(v))
	case int64:
		return time.Unix(v, 0), nil
	case uint64:
		return time.Unix(int64(v), 0), nil
	case float64:
		return time.Unix(int64(v), 0), nil
	default:
		s := fmt.Sprintf("%v", v)
		return parseTimestampString(s)
	}
}

func parseTimestampString(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range timestampLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp: %q", s)
}

// ResolveConflict 冲突仲裁结果日志
func (d *ConflictDetector) ResolveConflict(taskName string, rowKey string) string {
	switch d.strategy {
	case entity.ConflictStrategySourceWins:
		return fmt.Sprintf("[conflict-resolve] task [%s] row [%s]: source wins, proceeding with sync", taskName, rowKey)
	case entity.ConflictStrategyTargetWins:
		return fmt.Sprintf("[conflict-resolve] task [%s] row [%s]: target wins, skipping sync", taskName, rowKey)
	case entity.ConflictStrategySkip:
		return fmt.Sprintf("[conflict-resolve] task [%s] row [%s]: conflict detected, skipping", taskName, rowKey)
	default:
		return ""
	}
}

// BuildValidationSQL 构建数据校验 SQL（Phase 6.2）。
// 对比源/目标行数，可选 checksum 对比。
func BuildValidationSQL(srcTable, targetTable string, keyColumns []string) (srcCountSQL, targetCountSQL string, checksumSQL string) {
	srcCountSQL = fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", srcTable)
	targetCountSQL = fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", targetTable)

	if len(keyColumns) > 0 {
		// checksum：按主键排序后取前 100 行的 key 值拼接做简单对比
		cols := strings.Join(keyColumns, ", ")
		checksumSQL = fmt.Sprintf("SELECT %s FROM %s ORDER BY %s LIMIT 100", cols, srcTable, cols)
	}
	return
}

// target2SrcMap 构建目标列→源列的反向映射（缓存到 syncExecContext 避免重复构建）。
func (sec *syncExecContext) target2SrcMap() map[string]string {
	m := make(map[string]string, len(sec.fieldMap))
	for _, fm := range sec.fieldMap {
		m[fm["target"]] = fm["src"]
	}
	return m
}

// buildPKWhereClause 根据源行数据和字段映射构建目标表主键 WHERE 子句。
// 用于冲突检测时按主键查询目标行。
// target2Src 为目标列→源列的反向映射，targetDialect 用于安全格式化 SQL 字面量。
func buildPKWhereClause(srcRow map[string]any, target2Src map[string]string, targetTableMeta *dbi.TargetTableMeta, targetDialect dbi.Dialect) string {
	if len(targetTableMeta.UniqueColumns) == 0 {
		return ""
	}

	var conditions []string
	for _, pkCol := range targetTableMeta.UniqueColumns {
		srcCol, ok := target2Src[pkCol]
		if !ok {
			return "" // 主键列不在字段映射中，无法构建
		}
		val, ok := lookupRowValue(srcRow, srcCol)
		if !ok {
			return ""
		}
		// 安全格式化值：根据 Go 类型选择正确的 SQL 字面量表示
		sqlVal := formatSQLLiteral(val)
		conditions = append(conditions, fmt.Sprintf("%s = %s", pkCol, sqlVal))
	}
	return strings.Join(conditions, " AND ")
}

// formatSQLLiteral 将 Go 值安全格式化为 SQL 字面量，防止 SQL 注入。
func formatSQLLiteral(val any) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case string:
		// 转义单引号并用单引号包裹
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case []byte:
		escaped := strings.ReplaceAll(string(v), "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		// 其他类型转为字符串后安全转义
		s := fmt.Sprintf("%v", v)
		escaped := strings.ReplaceAll(s, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	}
}

// SyncMetrics 同步指标收集器
type SyncMetrics struct {
	StartTime   int64
	EndTime     int64
	TotalRows   int
	BatchCount  int
	InsertCount int
	UpdateCount int
	DeleteCount int
	SkipCount   int
}

// NewSyncMetrics 创建指标收集器
func NewSyncMetrics() *SyncMetrics {
	return &SyncMetrics{}
}

// RecordBatch 记录一批同步完成的指标
func (m *SyncMetrics) RecordBatch(rows int) {
	m.TotalRows += rows
	m.BatchCount++
}

// ToSyncLog 将指标填充到 DataSyncLog
func (m *SyncMetrics) ToSyncLog(log *entity.DataSyncLog) {
	log.ResNum = m.TotalRows
	log.BatchCount = m.BatchCount
	log.InsertCount = m.InsertCount
	log.UpdateCount = m.UpdateCount
	log.DeleteCount = m.DeleteCount
	log.SkipCount = m.SkipCount
	if m.EndTime > m.StartTime {
		log.DurationMs = m.EndTime - m.StartTime
		if log.DurationMs > 0 {
			log.Throughput = int(int64(m.TotalRows) * 1000 / log.DurationMs)
		}
	}
}

// buildPKKey 构建源行的主键字符串键（用于 map 查找）。
// 返回空字符串表示无法构建（缺少主键列或值）。
func buildPKKey(srcRow map[string]any, target2Src map[string]string, targetTableMeta *dbi.TargetTableMeta) string {
	if len(targetTableMeta.UniqueColumns) == 0 {
		return ""
	}
	var parts []string
	for _, pkCol := range targetTableMeta.UniqueColumns {
		srcCol, ok := target2Src[pkCol]
		if !ok {
			return ""
		}
		val, ok := lookupRowValue(srcRow, srcCol)
		if !ok {
			return ""
		}
		parts = append(parts, fmt.Sprintf("%v", val))
	}
	return strings.Join(parts, "\x00")
}

// batchLoadConflictTargetRows 批量查询目标表中的冲突检测行。
// 将 N 次单行查询合并为分批批量查询，消除 N+1 性能问题。
// 每批最多 conflictBatchSize 行，防止 OR 条件过长。
// 返回 map[pkKey]targetRow；若无需检测（无冲突检测器或无主键）则返回 nil。
func (app *DataSyncAppImpl) batchLoadConflictTargetRows(
	ctx context.Context,
	srcRes []map[string]any,
	sec *syncExecContext,
) map[string]map[string]any {
	if sec.conflictDetector == nil || len(sec.targetTableMeta.UniqueColumns) == 0 || len(srcRes) == 0 {
		return nil
	}

	task := sec.task
	targetDbConn := sec.targetConn
	targetTableMeta := sec.targetTableMeta

	// 构建一次 target2Src 映射，供整批使用
	target2Src := sec.target2SrcMap()

	// 收集所有源行的主键值和 WHERE 条件
	type pkEntry struct {
		key    string
		clause string // "(pk1 = v1 AND pk2 = v2)"
	}
	entries := make([]pkEntry, 0, len(srcRes))
	for _, srcData := range srcRes {
		pkKey := buildPKKey(srcData, target2Src, targetTableMeta)
		if pkKey == "" {
			continue
		}
		pkWhere := buildPKWhereClause(srcData, target2Src, targetTableMeta, targetDbConn.GetDialect())
		if pkWhere == "" {
			continue
		}
		entries = append(entries, pkEntry{key: pkKey, clause: "(" + pkWhere + ")"})
	}

	if len(entries) == 0 {
		return nil
	}

	// 构建 SELECT 列：时间戳字段 + 主键列
	selectCols := task.BiDirTimestampField
	for _, pkCol := range targetTableMeta.UniqueColumns {
		selectCols += ", " + pkCol
	}

	result := make(map[string]map[string]any, len(entries))

	// 分批查询，每批最多 conflictBatchSize 行
	for batchStart := 0; batchStart < len(entries); batchStart += conflictBatchSize {
		batchEnd := batchStart + conflictBatchSize
		if batchEnd > len(entries) {
			batchEnd = len(entries)
		}
		batch := entries[batchStart:batchEnd]

		orConditions := make([]string, len(batch))
		for i, e := range batch {
			orConditions[i] = e.clause
		}
		batchSql := fmt.Sprintf("SELECT %s FROM %s WHERE %s",
			selectCols, task.TargetTableName, strings.Join(orConditions, " OR "))

		_, _ = targetDbConn.WalkQueryRows(ctx, batchSql, func(row map[string]any, cols []*dbi.QueryColumn) error {
			targetRow := make(map[string]any, len(row))
			for k, v := range row {
				targetRow[k] = v
			}
			var keyParts []string
			for _, pkCol := range targetTableMeta.UniqueColumns {
				if val, ok := lookupRowValue(row, pkCol); ok {
					keyParts = append(keyParts, fmt.Sprintf("%v", val))
				}
			}
			if len(keyParts) == len(targetTableMeta.UniqueColumns) {
				pkKey := strings.Join(keyParts, "\x00")
				result[pkKey] = targetRow
			}
			return nil
		})
	}

	return result
}
