package sync

import (
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========== SchemaDetector Tests ==========

func TestNewSchemaDetector_Off(t *testing.T) {
	detector := NewSchemaDetector(entity.SchemaEvolveOff)
	assert.Nil(t, detector, "SchemaEvolveOff should return nil detector")
}

func TestNewSchemaDetector_Warn(t *testing.T) {
	detector := NewSchemaDetector(entity.SchemaEvolveWarn)
	assert.NotNil(t, detector)
}

func TestNewSchemaDetector_Auto(t *testing.T) {
	detector := NewSchemaDetector(entity.SchemaEvolveAuto)
	assert.NotNil(t, detector)
}

// TestSchemaDetector_DetectChanges_TargetColumnMissing 检测 fieldMap 映射的目标列不存在于目标表
func TestSchemaDetector_DetectChanges_TargetColumnMissing(t *testing.T) {
	detector := NewSchemaDetector(entity.SchemaEvolveWarn)

	fieldMap := []map[string]string{
		{"src": "id", "target": "id"},
		{"src": "name", "target": "name"},
		{"src": "email", "target": "email"}, // 目标表没有 email 列
	}
	targetColumns := []dbi.Column{
		{ColumnName: "id"},
		{ColumnName: "name"},
	}

	changes := detector.DetectChanges(fieldMap, targetColumns)
	assert.Len(t, changes, 1)
	assert.Equal(t, SchemaChangeTargetColumnMissing, changes[0].Type)
	assert.Equal(t, "email", changes[0].ColumnName)
}

// TestSchemaDetector_DetectChanges_UnmappedTargetColumnsIgnored 目标表有但 fieldMap 未映射的列不算变更
func TestSchemaDetector_DetectChanges_UnmappedTargetColumnsIgnored(t *testing.T) {
	detector := NewSchemaDetector(entity.SchemaEvolveWarn)

	// 用户只映射了 id，目标表有 id + name + age，但用户故意不同步 name 和 age
	fieldMap := []map[string]string{
		{"src": "user_id", "target": "id"},
	}
	targetColumns := []dbi.Column{
		{ColumnName: "id"},
		{ColumnName: "name", ColumnType: "varchar(100)"},
		{ColumnName: "age", ColumnType: "int"},
	}

	changes := detector.DetectChanges(fieldMap, targetColumns)
	assert.Len(t, changes, 0, "unmapped target columns should NOT be reported as schema changes")
}

// TestSchemaDetector_DetectChanges_NoChanges fieldMap 所有目标列都存在
func TestSchemaDetector_DetectChanges_NoChanges(t *testing.T) {
	detector := NewSchemaDetector(entity.SchemaEvolveWarn)

	fieldMap := []map[string]string{
		{"src": "id", "target": "id"},
		{"src": "name", "target": "name"},
	}
	targetColumns := []dbi.Column{
		{ColumnName: "id"},
		{ColumnName: "name"},
	}

	changes := detector.DetectChanges(fieldMap, targetColumns)
	assert.Len(t, changes, 0)
}

// TestSchemaDetector_DetectChanges_CaseInsensitive 大小写不敏感
func TestSchemaDetector_DetectChanges_CaseInsensitive(t *testing.T) {
	detector := NewSchemaDetector(entity.SchemaEvolveWarn)

	fieldMap := []map[string]string{
		{"src": "user_id", "target": "ID"},
		{"src": "user_name", "target": "Name"},
	}
	targetColumns := []dbi.Column{
		{ColumnName: "id"},
		{ColumnName: "name"},
	}

	changes := detector.DetectChanges(fieldMap, targetColumns)
	assert.Len(t, changes, 0, "comparison should be case-insensitive")
}

// TestSchemaDetector_DetectChanges_MultipleMissing 多个目标列缺失
func TestSchemaDetector_DetectChanges_MultipleMissing(t *testing.T) {
	detector := NewSchemaDetector(entity.SchemaEvolveAuto)

	fieldMap := []map[string]string{
		{"src": "id", "target": "id"},
		{"src": "name", "target": "user_name"},   // 目标表没有
		{"src": "email", "target": "user_email"}, // 目标表没有
	}
	targetColumns := []dbi.Column{
		{ColumnName: "id"},
	}

	changes := detector.DetectChanges(fieldMap, targetColumns)
	assert.Len(t, changes, 2)
}

func TestSchemaDetector_HandleChanges_WarnMode(t *testing.T) {
	detector := NewSchemaDetector(entity.SchemaEvolveWarn)

	changes := []SchemaChange{
		{Type: SchemaChangeTargetColumnMissing, ColumnName: "email"},
	}

	skipCols := detector.HandleChanges(nil, changes, "test-task")
	assert.Len(t, skipCols, 0, "warn mode should not skip any columns")
}

func TestSchemaDetector_HandleChanges_AutoMode(t *testing.T) {
	detector := NewSchemaDetector(entity.SchemaEvolveAuto)

	changes := []SchemaChange{
		{Type: SchemaChangeTargetColumnMissing, ColumnName: "email"},
	}

	skipCols := detector.HandleChanges(nil, changes, "test-task")
	assert.Len(t, skipCols, 1, "auto mode should skip missing target columns")
	assert.True(t, skipCols["email"])
}

func TestFilterFieldMapBySchemaChanges(t *testing.T) {
	fieldMap := []map[string]string{
		{"src": "id", "target": "id"},
		{"src": "name", "target": "name"},
		{"src": "email", "target": "email"},
	}

	skipCols := map[string]bool{"email": true}
	filtered := FilterFieldMapBySchemaChanges(fieldMap, skipCols)

	assert.Len(t, filtered, 2)
	assert.Equal(t, "id", filtered[0]["target"])
	assert.Equal(t, "name", filtered[1]["target"])
}

func TestFilterFieldMapBySchemaChanges_EmptySkip(t *testing.T) {
	fieldMap := []map[string]string{
		{"src": "id", "target": "id"},
		{"src": "name", "target": "name"},
	}

	filtered := FilterFieldMapBySchemaChanges(fieldMap, nil)
	assert.Len(t, filtered, 2, "nil skip set should not filter anything")
}

func TestSchemaDetector_NilReceiver(t *testing.T) {
	var detector *SchemaDetector
	changes := detector.DetectChanges([]map[string]string{{"src": "id", "target": "id"}}, []dbi.Column{{ColumnName: "id"}})
	assert.Nil(t, changes, "nil detector should return nil changes")
}
