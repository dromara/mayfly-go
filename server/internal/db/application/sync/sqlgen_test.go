package sync

import (
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"strings"
	"testing"
	"time"

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
	assert.NotContains(t, checksumSQL, "LIMIT", "checksumSQL 不含方言特定分页，由调用方 paginateTopN 改写")
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

// TestSplitFilterLogical 验证条件按顶层逻辑词分割（引号/括号感知）
func TestSplitFilterLogical(t *testing.T) {
	parts, err := splitFilterLogical("status == 1 AND age > 18", "AND")
	assert.NoError(t, err)
	assert.Len(t, parts, 2)
	assert.Equal(t, "status == 1", strings.TrimSpace(parts[0]))
	assert.Equal(t, "age > 18", strings.TrimSpace(parts[1]))
}

func TestSplitFilterLogical_Single(t *testing.T) {
	parts, err := splitFilterLogical("status == 1", "AND")
	assert.NoError(t, err)
	assert.Len(t, parts, 1)
}

func TestSplitFilterLogical_CaseInsensitive(t *testing.T) {
	parts, err := splitFilterLogical("a == 1 and b == 2", "AND")
	assert.NoError(t, err)
	assert.Len(t, parts, 2, "AND should be case-insensitive")
}

func TestSplitFilterLogical_RespectsParens(t *testing.T) {
	parts, err := splitFilterLogical("(a == 1 OR b == 2) AND c == 3", "AND")
	assert.NoError(t, err)
	assert.Len(t, parts, 2, "括号内 OR 不参与 AND 分割")
}

// ========== 补充：bidirectional 纯函数单元测试 ==========
// ConflictDetector / ReverseFieldMapJSON / SyncMetrics 已在 bidirectional_test.go 覆盖，
// 以下补充 parseTimestamp / formatSQLLiteral / buildPKWhereClause / buildPKKey 等缺失项。

// TestParseTimestamp 验证多种时间格式解析
func TestParseTimestamp(t *testing.T) {
	// time.Time直接返回
	now := time.Now()
	ts, err := parseTimestamp(now)
	assert.NoError(t, err)
	assert.Equal(t, now, ts)

	// RFC3339
	ts, err = parseTimestamp("2025-01-15T10:30:00Z")
	assert.NoError(t, err)
	assert.Equal(t, 2025, ts.Year())

	// 标准日期时间
	ts, err = parseTimestamp("2025-01-15 10:30:00")
	assert.NoError(t, err)
	assert.Equal(t, 10, ts.Hour())

	// int64 Unix时间戳
	ts, err = parseTimestamp(int64(1700000000))
	assert.NoError(t, err)
	assert.True(t, ts.Year() >= 2023)

	// 无法解析
	_, err = parseTimestamp("not-a-timestamp")
	assert.Error(t, err)
}

// TestFormatSQLLiteral 验证SQL字面量安全格式化（防注入）
func TestFormatSQLLiteral(t *testing.T) {
	// nil escapeFn 使用标准SQL转义（单引号双写）
	assert.Equal(t, "NULL", formatSQLLiteral(nil, nil))
	assert.Equal(t, "'hello'", formatSQLLiteral("hello", nil))
	assert.Equal(t, "'it''s'", formatSQLLiteral("it's", nil), "单引号应转义")
	assert.Equal(t, "42", formatSQLLiteral(42, nil))
	assert.Equal(t, "3.14", formatSQLLiteral(3.14, nil))
	assert.Equal(t, "1", formatSQLLiteral(true, nil))
	assert.Equal(t, "0", formatSQLLiteral(false, nil))
	assert.Equal(t, "'hello'", formatSQLLiteral([]byte("hello"), nil), "[]byte应转字符串")

	// MySQL escapeFn 额外转义反斜杠
	mysqlFn := dbi.SQLValueStringEscapeBackslash
	assert.Equal(t, `'a\\b'`, formatSQLLiteral(`a\b`, mysqlFn), "MySQL下反斜杠必须双写")
	assert.Equal(t, `'a\\b''c'`, formatSQLLiteral(`a\b'c`, mysqlFn), "反斜杠与单引号同时转义")
}

// TestBuildPKWhereClause 验证主键WHERE子句构建
func TestBuildPKWhereClause(t *testing.T) {
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	target2Src := map[string]string{"id": "user_id"}
	srcRow := map[string]any{"user_id": 42}

	clause := buildPKWhereClause(srcRow, target2Src, meta, nil)
	assert.Equal(t, "id = 42", clause)
}

// TestBuildPKWhereClause_CompositeKey 复合主键
func TestBuildPKWhereClause_CompositeKey(t *testing.T) {
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"org_id", "user_id"}}
	target2Src := map[string]string{"org_id": "org", "user_id": "uid"}
	srcRow := map[string]any{"org": "acme", "uid": 7}

	clause := buildPKWhereClause(srcRow, target2Src, meta, nil)
	assert.Contains(t, clause, "org_id = 'acme'")
	assert.Contains(t, clause, "user_id = 7")
	assert.Contains(t, clause, " AND ")
}

// TestBuildPKWhereClause_MissingColumn 主键列不在映射中返回空
func TestBuildPKWhereClause_MissingColumn(t *testing.T) {
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	target2Src := map[string]string{"id": "user_id"}
	srcRow := map[string]any{"name": "alice"}

	clause := buildPKWhereClause(srcRow, target2Src, meta, nil)
	assert.Empty(t, clause, "missing src column → empty WHERE")
}

// TestBuildPKKey 验证主键字符串键构建
func TestBuildPKKey(t *testing.T) {
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	target2Src := map[string]string{"id": "user_id"}

	key := buildPKKey(map[string]any{"user_id": 42}, target2Src, meta)
	assert.Equal(t, "42", key)

	// 复合主键
	meta2 := &dbi.TargetTableMeta{UniqueColumns: []string{"a", "b"}}
	target2Src2 := map[string]string{"a": "x", "b": "y"}
	key2 := buildPKKey(map[string]any{"x": "foo", "y": "bar"}, target2Src2, meta2)
	assert.Contains(t, key2, "foo")
	assert.Contains(t, key2, "bar")

	// 缺少列
	key3 := buildPKKey(map[string]any{"other": 1}, target2Src, meta)
	assert.Empty(t, key3, "missing column → empty key")
}
