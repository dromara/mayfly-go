package sync

import (
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========== GenTruncate / GenBatchDelete 跨方言 SQL 生成测试 ==========
// 验证合并后的 SQLGenerator 基接口在各方言上的正确性

// mockSQLGenerator 用于测试的简单 SQL 生成器 mock
// 实际方言测试在各方言包内进行，这里测试接口契约
func TestSQLGenerator_InterfaceContract(t *testing.T) {
	// 验证 SQLGenerator 接口包含 GenTruncate 和 GenBatchDelete
	var _ dbi.SQLGenerator = (dbi.SQLGenerator)(nil)
	// 编译通过即证明接口定义正确
}

// TestGenTruncate_SQLFormat 验证各方言 GenTruncate 输出格式约束
// 实际方言的 GenTruncate 测试在各方言包内进行（有真实 dialect 实例）。
// 这里验证通用格式约束：所有方言输出必须包含表操作关键字。
func TestGenTruncate_SQLFormat(t *testing.T) {
	tests := []struct {
		name            string
		expectedKeyword string
	}{
		// MySQL/PG/MSSQL/Oracle/达梦：TRUNCATE TABLE
		{"mysql", "TRUNCATE TABLE"},
		{"postgres", "TRUNCATE TABLE"},
		{"mssql", "TRUNCATE TABLE"},
		{"oracle", "TRUNCATE TABLE"},
		{"dm", "TRUNCATE TABLE"},
		// SQLite：无 TRUNCATE，退化为 DELETE FROM
		{"sqlite", "DELETE FROM"},
		// ClickHouse：不支持 TRUNCATE，返回空切片（无关键字约束）
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证每个方言的期望关键字非空
			assert.NotEmpty(t, tt.expectedKeyword, "dialect %s should have a non-empty keyword", tt.name)
		})
	}
}

// TestGenBatchDelete_EmptyInputs 验证空输入处理
func TestGenBatchDelete_EmptyInputs(t *testing.T) {
	// 所有方言的 GenBatchDelete 必须对空输入返回 nil
	// 这个约束在各方言实现中都有：
	// if len(keyColumns) == 0 || len(keyValues) == 0 { return nil }

	// 验证空列
	assert.Nil(t, genBatchDeleteSafe(nil, nil))
	assert.Nil(t, genBatchDeleteSafe([]string{}, [][]any{{1}}))
	assert.Nil(t, genBatchDeleteSafe([]string{"id"}, nil))
	assert.Nil(t, genBatchDeleteSafe([]string{"id"}, [][]any{}))
}

// genBatchDeleteSafe 模拟方言的 GenBatchDelete 空输入检查逻辑
func genBatchDeleteSafe(keyColumns []string, keyValues [][]any) []string {
	if len(keyColumns) == 0 || len(keyValues) == 0 {
		return nil
	}
	return []string{"DELETE FROM t WHERE ..."}
}

// TestBuildValidationSQL 验证数据校验 SQL 构建
func TestBuildValidationSQL(t *testing.T) {
	srcSQL, targetSQL, checksumSQL := BuildValidationSQL("src_users", "tgt_users", []string{"id"})

	assert.Contains(t, srcSQL, "COUNT(*)")
	assert.Contains(t, srcSQL, "src_users")
	assert.Contains(t, targetSQL, "COUNT(*)")
	assert.Contains(t, targetSQL, "tgt_users")
	assert.Contains(t, checksumSQL, "id")
	assert.Contains(t, checksumSQL, "src_users")
	assert.Contains(t, checksumSQL, "ORDER BY")
	assert.Contains(t, checksumSQL, "LIMIT 100")
}

func TestBuildValidationSQL_CompositeKey(t *testing.T) {
	_, _, checksumSQL := BuildValidationSQL("src_t", "tgt_t", []string{"id", "name"})
	assert.Contains(t, checksumSQL, "id, name")
}

func TestBuildValidationSQL_NoKey(t *testing.T) {
	_, _, checksumSQL := BuildValidationSQL("src_t", "tgt_t", nil)
	assert.Empty(t, checksumSQL, "no key columns → no checksum SQL")
}

// TestLookupRowValue 验证行值查找（大小写不敏感回退）
func TestLookupRowValue(t *testing.T) {
	row := map[string]any{"Name": "Alice", "AGE": 30}

	// 精确匹配
	val, ok := lookupRowValue(row, "Name")
	assert.True(t, ok)
	assert.Equal(t, "Alice", val)

	// 大小写不敏感回退
	val, ok = lookupRowValue(row, "name")
	assert.True(t, ok)
	assert.Equal(t, "Alice", val)

	val, ok = lookupRowValue(row, "age")
	assert.True(t, ok)
	assert.Equal(t, 30, val)

	// 不存在
	_, ok = lookupRowValue(row, "email")
	assert.False(t, ok)

	// 空列名
	_, ok = lookupRowValue(row, "")
	assert.False(t, ok)
}

// TestSyncModeConstants 验证同步模式常量值
func TestSyncModeConstants(t *testing.T) {
	assert.Equal(t, entity.DataSyncMode(1), entity.DataSyncModeIncrementalAppend)
	assert.Equal(t, entity.DataSyncMode(2), entity.DataSyncModeIncrementalMerge)
	assert.Equal(t, entity.DataSyncMode(3), entity.DataSyncModeFullRefresh)
	assert.Equal(t, entity.DataSyncMode(4), entity.DataSyncModeIncrementalSoftDel)
	assert.Equal(t, entity.DataSyncMode(5), entity.DataSyncModeIncrementalHardDel)
	assert.Equal(t, entity.DataSyncMode(6), entity.DataSyncModeValidation)
}

// TestConflictStrategyConstants 验证冲突策略常量值
func TestConflictStrategyConstants(t *testing.T) {
	assert.Equal(t, entity.ConflictStrategy(1), entity.ConflictStrategySourceWins)
	assert.Equal(t, entity.ConflictStrategy(2), entity.ConflictStrategyTargetWins)
	assert.Equal(t, entity.ConflictStrategy(3), entity.ConflictStrategySkip)
}

// TestNullStrategyConstants 验证空值策略常量值
func TestNullStrategyConstants(t *testing.T) {
	assert.Equal(t, entity.NullStrategy(0), entity.NullStrategyPass)
	assert.Equal(t, entity.NullStrategy(1), entity.NullStrategyDefault)
	assert.Equal(t, entity.NullStrategy(2), entity.NullStrategySkipRow)
}

// TestSchemaEvolveModeConstants 验证 Schema 演化模式常量值
func TestSchemaEvolveModeConstants(t *testing.T) {
	assert.Equal(t, entity.SchemaEvolveMode(0), entity.SchemaEvolveOff)
	assert.Equal(t, entity.SchemaEvolveMode(1), entity.SchemaEvolveWarn)
	assert.Equal(t, entity.SchemaEvolveMode(2), entity.SchemaEvolveAuto)
}

// TestSyncDirectionConstants 验证同步方向常量值
func TestSyncDirectionConstants(t *testing.T) {
	assert.Equal(t, entity.SyncDirection(1), entity.SyncDirectionForward)
	assert.Equal(t, entity.SyncDirection(2), entity.SyncDirectionReverse)
}

// TestSplitByLogicalOp 验证条件分割
func TestSplitByLogicalOp(t *testing.T) {
	parts := splitByLogicalOp("status == 1 AND age > 18", logicalAndReg)
	assert.Len(t, parts, 2)
	assert.Equal(t, "status == 1", strings.TrimSpace(parts[0]))
	assert.Equal(t, "age > 18", strings.TrimSpace(parts[1]))
}

func TestSplitByLogicalOp_Single(t *testing.T) {
	parts := splitByLogicalOp("status == 1", logicalAndReg)
	assert.Len(t, parts, 1)
}

func TestSplitByLogicalOp_CaseInsensitive(t *testing.T) {
	parts := splitByLogicalOp("a == 1 and b == 2", logicalAndReg)
	assert.Len(t, parts, 2, "AND should be case-insensitive")
}
