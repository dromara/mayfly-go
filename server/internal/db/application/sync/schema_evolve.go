package sync

import (
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/logx"
	"strings"
)

// SchemaChangeType Schema 变更类型
type SchemaChangeType string

const (
	// SchemaChangeTargetColumnMissing 目标表缺少 fieldMap 映射所需的列。
	SchemaChangeTargetColumnMissing SchemaChangeType = "TARGET_COLUMN_MISSING"
	// SchemaChangeSourceColumnMissing 源查询结果缺少 fieldMap 映射所需的源列。
	// 若源表删除了某列或 DataSql 未包含该列，同步会静默产生 NULL 值。
	SchemaChangeSourceColumnMissing SchemaChangeType = "SOURCE_COLUMN_MISSING"
)

// SchemaChange 记录单个 Schema 变更
type SchemaChange struct {
	Type       SchemaChangeType `json:"type"`
	ColumnName string           `json:"columnName"`
	OldValue   string           `json:"oldValue,omitempty"`
	NewValue   string           `json:"newValue,omitempty"`
}

// SchemaDetector Schema 变更检测器。
// 通过对比源表/目标表的列信息，检测结构差异并按 SchemaEvolveMode 处理。
type SchemaDetector struct {
	mode entity.SchemaEvolveMode
}

// NewSchemaDetector 创建检测器。mode=0 时返回 nil（不检测）。
func NewSchemaDetector(mode entity.SchemaEvolveMode) *SchemaDetector {
	if mode == entity.SchemaEvolveOff {
		return nil
	}
	return &SchemaDetector{mode: mode}
}

// DetectChanges 检测 fieldMap 映射的目标列是否在目标表中存在。
func (d *SchemaDetector) DetectChanges(fieldMap []map[string]string, targetColumns []dbi.Column) []SchemaChange {
	if d == nil {
		return nil
	}
	return d.DetectChangesWithSource(fieldMap, nil, targetColumns)
}

// DetectChangesWithSource 同时检测源列和目标列的 Schema 变化。
// srcColumns 为源查询结果的列名列表（可为 nil，表示不检测源列）。
func (d *SchemaDetector) DetectChangesWithSource(fieldMap []map[string]string, srcColumns []string, targetColumns []dbi.Column) []SchemaChange {
	if d == nil {
		return nil
	}

	// 构建源列名集合（小写）
	srcColSet := make(map[string]bool, len(srcColumns))
	for _, col := range srcColumns {
		srcColSet[strings.ToLower(col)] = true
	}

	// 构建目标列名集合（小写）
	targetColMap := make(map[string]dbi.Column, len(targetColumns))
	for _, col := range targetColumns {
		targetColMap[strings.ToLower(col.ColumnName)] = col
	}

	var changes []SchemaChange

	for _, fm := range fieldMap {
		srcCol := fm["src"]
		targetCol := fm["target"]

		// 检测源列是否存在
		if srcCol != "" && len(srcColSet) > 0 {
			if !srcColSet[strings.ToLower(srcCol)] {
				changes = append(changes, SchemaChange{
					Type:       SchemaChangeSourceColumnMissing,
					ColumnName: srcCol,
					NewValue:   "mapped in fieldMap but not found in source query result",
				})
			}
		}

		// 检测目标列是否存在
		if targetCol != "" {
			if _, ok := targetColMap[strings.ToLower(targetCol)]; !ok {
				changes = append(changes, SchemaChange{
					Type:       SchemaChangeTargetColumnMissing,
					ColumnName: targetCol,
					NewValue:   "mapped in fieldMap but not found in target table",
				})
			}
		}
	}

	return changes
}

// HandleChanges 处理检测到的 Schema 变更。
// 根据 mode 决定是仅告警还是自动适配（跳过缺失列映射）。
// 返回应跳过的目标列名集合（auto 模式下目标表不存在的列应跳过，避免 INSERT 报错）。
func (d *SchemaDetector) HandleChanges(_ func(string, ...any), changes []SchemaChange, taskName string) map[string]bool {
	if d == nil || len(changes) == 0 {
		return nil
	}

	skipColumns := make(map[string]bool)

	for _, change := range changes {
		switch d.mode {
		case entity.SchemaEvolveWarn:
			logx.Warnf("[schema-evolve] task [%s] detected %s on column [%s]: %s",
				taskName, change.Type, change.ColumnName, change.NewValue)

		case entity.SchemaEvolveAuto:
			logx.Warnf("[schema-evolve] task [%s] auto-adapting %s on column [%s]: %s",
				taskName, change.Type, change.ColumnName, change.NewValue)
			// 自动适配：目标表不存在的列 → 跳过该列映射
			if change.Type == SchemaChangeTargetColumnMissing {
				skipColumns[strings.ToLower(change.ColumnName)] = true
			}
		}
	}

	return skipColumns
}

// FilterFieldMapBySchemaChanges 根据 Schema 检测结果过滤字段映射。
// 在 auto 模式下，跳过目标表中不存在的列映射。
func FilterFieldMapBySchemaChanges(fieldMap []map[string]string, skipColumns map[string]bool) []map[string]string {
	if len(skipColumns) == 0 {
		return fieldMap
	}
	filtered := make([]map[string]string, 0, len(fieldMap))
	for _, fm := range fieldMap {
		if skipColumns[strings.ToLower(fm["target"])] {
			continue
		}
		filtered = append(filtered, fm)
	}
	return filtered
}
