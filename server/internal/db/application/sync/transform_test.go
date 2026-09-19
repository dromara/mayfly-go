package sync

import (
	"mayfly-go/internal/db/domain/entity"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== TransformEngine Tests ==========

func TestNewTransformEngine_EmptyRules(t *testing.T) {
	engine, err := NewTransformEngine("")
	assert.NoError(t, err)
	assert.Nil(t, engine, "empty rules should return nil engine")
}

func TestNewTransformEngine_InvalidJSON(t *testing.T) {
	_, err := NewTransformEngine("not json")
	assert.Error(t, err)
}

func TestNewTransformEngine_EmptyArray(t *testing.T) {
	engine, err := NewTransformEngine("[]")
	assert.NoError(t, err)
	assert.Nil(t, engine, "empty array should return nil engine")
}

func TestTransformEngine_ColumnMapping(t *testing.T) {
	rules := `[{"targetColumn":"name","type":"column","config":{"sourceColumn":"user_name"}}]`
	engine, err := NewTransformEngine(rules)
	require.NoError(t, err)

	srcRow := map[string]any{"user_name": "Alice", "age": 30}
	fieldMap := []map[string]string{{"src": "user_name", "target": "name"}}
	task := &entity.DataSyncTask{}

	result, skip := engine.Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Equal(t, "Alice", result["name"])
}

func TestTransformEngine_ConstantValue(t *testing.T) {
	rules := `[{"targetColumn":"status","type":"constant","config":{"value":"active"}}]`
	engine, err := NewTransformEngine(rules)
	require.NoError(t, err)

	srcRow := map[string]any{"id": 1}
	fieldMap := []map[string]string{{"src": "id", "target": "status"}}
	task := &entity.DataSyncTask{}

	result, skip := engine.Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Equal(t, "active", result["status"])
}

func TestTransformEngine_ExpressionUpper(t *testing.T) {
	rules := `[{"targetColumn":"name","type":"expr","config":{"expression":"UPPER(name)"}}]`
	engine, err := NewTransformEngine(rules)
	require.NoError(t, err)

	srcRow := map[string]any{"name": "alice"}
	fieldMap := []map[string]string{{"src": "name", "target": "name"}}
	task := &entity.DataSyncTask{}

	result, skip := engine.Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Equal(t, "ALICE", result["name"])
}

func TestTransformEngine_ExpressionLower(t *testing.T) {
	rules := `[{"targetColumn":"name","type":"expr","config":{"expression":"LOWER(name)"}}]`
	engine, err := NewTransformEngine(rules)
	require.NoError(t, err)

	srcRow := map[string]any{"name": "ALICE"}
	fieldMap := []map[string]string{{"src": "name", "target": "name"}}
	task := &entity.DataSyncTask{}

	result, skip := engine.Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Equal(t, "alice", result["name"])
}

func TestTransformEngine_ExpressionTrim(t *testing.T) {
	rules := `[{"targetColumn":"name","type":"expr","config":{"expression":"TRIM(name)"}}]`
	engine, err := NewTransformEngine(rules)
	require.NoError(t, err)

	srcRow := map[string]any{"name": "  alice  "}
	fieldMap := []map[string]string{{"src": "name", "target": "name"}}
	task := &entity.DataSyncTask{}

	result, skip := engine.Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Equal(t, "alice", result["name"])
}

func TestTransformEngine_ExpressionConcat(t *testing.T) {
	rules := `[{"targetColumn":"full_name","type":"expr","config":{"expression":"CONCAT(first_name, ' ', last_name)"}}]`
	engine, err := NewTransformEngine(rules)
	require.NoError(t, err)

	srcRow := map[string]any{"first_name": "John", "last_name": "Doe"}
	fieldMap := []map[string]string{{"src": "first_name", "target": "full_name"}}
	task := &entity.DataSyncTask{}

	result, skip := engine.Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Equal(t, "John Doe", result["full_name"])
}

func TestTransformEngine_ExpressionNested(t *testing.T) {
	rules := `[{"targetColumn":"name","type":"expr","config":{"expression":"UPPER(TRIM(name))"}}]`
	engine, err := NewTransformEngine(rules)
	require.NoError(t, err)

	srcRow := map[string]any{"name": "  alice  "}
	fieldMap := []map[string]string{{"src": "name", "target": "name"}}
	task := &entity.DataSyncTask{}

	result, skip := engine.Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Equal(t, "ALICE", result["name"])
}

func TestTransformEngine_ExpressionReplace(t *testing.T) {
	rules := `[{"targetColumn":"phone","type":"expr","config":{"expression":"REPLACE(phone, '-', '')"}}]`
	engine, err := NewTransformEngine(rules)
	require.NoError(t, err)

	srcRow := map[string]any{"phone": "123-456-7890"}
	fieldMap := []map[string]string{{"src": "phone", "target": "phone"}}
	task := &entity.DataSyncTask{}

	result, skip := engine.Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Equal(t, "1234567890", result["phone"])
}

func TestTransformEngine_ExpressionSubstring(t *testing.T) {
	rules := `[{"targetColumn":"code","type":"expr","config":{"expression":"SUBSTRING(code, 1, 3)"}}]`
	engine, err := NewTransformEngine(rules)
	require.NoError(t, err)

	srcRow := map[string]any{"code": "ABCDEFG"}
	fieldMap := []map[string]string{{"src": "code", "target": "code"}}
	task := &entity.DataSyncTask{}

	result, skip := engine.Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Equal(t, "ABC", result["code"])
}

func TestTransformEngine_NilStrategyDefault(t *testing.T) {
	srcRow := map[string]any{"name": nil, "age": 30}
	fieldMap := []map[string]string{{"src": "name", "target": "name"}, {"src": "age", "target": "age"}}
	task := &entity.DataSyncTask{NullStrategy: entity.NullStrategyDefault, NullDefault: "unknown"}

	// nil engine (no transform rules) → default mapping with null strategy
	result, skip := (*TransformEngine)(nil).Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Equal(t, "unknown", result["name"])
	assert.Equal(t, 30, result["age"])
}

func TestTransformEngine_NilStrategySkipRow(t *testing.T) {
	srcRow := map[string]any{"name": nil}
	fieldMap := []map[string]string{{"src": "name", "target": "name"}}
	task := &entity.DataSyncTask{NullStrategy: entity.NullStrategySkipRow}

	result, skip := (*TransformEngine)(nil).Transform(srcRow, fieldMap, task)
	assert.True(t, skip, "should skip row when null strategy is SkipRow")
	assert.Nil(t, result)
}

func TestTransformEngine_NilStrategyPass(t *testing.T) {
	srcRow := map[string]any{"name": nil}
	fieldMap := []map[string]string{{"src": "name", "target": "name"}}
	task := &entity.DataSyncTask{NullStrategy: entity.NullStrategyPass}

	result, skip := (*TransformEngine)(nil).Transform(srcRow, fieldMap, task)
	assert.False(t, skip)
	assert.Nil(t, result["name"], "should keep NULL when strategy is Pass")
}

// ========== FilterEngine Tests ==========

func TestFilterEngine_Nil(t *testing.T) {
	engine := NewFilterEngine("")
	assert.Nil(t, engine, "empty condition should return nil engine")
}

func TestFilterEngine_EqualMatch(t *testing.T) {
	engine := NewFilterEngine("status == 1")
	require.NotNil(t, engine)

	assert.True(t, engine.Match(map[string]any{"status": "1"}))
	assert.False(t, engine.Match(map[string]any{"status": "0"}))
}

func TestFilterEngine_NotEqual(t *testing.T) {
	engine := NewFilterEngine("status != 0")
	require.NotNil(t, engine)

	assert.True(t, engine.Match(map[string]any{"status": "1"}))
	assert.False(t, engine.Match(map[string]any{"status": "0"}))
}

func TestFilterEngine_GreaterThan(t *testing.T) {
	engine := NewFilterEngine("age > 18")
	require.NotNil(t, engine)

	assert.True(t, engine.Match(map[string]any{"age": "20"}))
	assert.False(t, engine.Match(map[string]any{"age": "15"}))
}

func TestFilterEngine_LessThan(t *testing.T) {
	engine := NewFilterEngine("age < 18")
	require.NotNil(t, engine)

	assert.True(t, engine.Match(map[string]any{"age": "15"}))
	assert.False(t, engine.Match(map[string]any{"age": "20"}))
}

func TestFilterEngine_Like(t *testing.T) {
	engine := NewFilterEngine("name LIKE %alice%")
	require.NotNil(t, engine)

	assert.True(t, engine.Match(map[string]any{"name": "bob_alice_carl"}))
	assert.False(t, engine.Match(map[string]any{"name": "bob_carl"}))
}

func TestFilterEngine_AndCondition(t *testing.T) {
	engine := NewFilterEngine("status == 1 AND age > 18")
	require.NotNil(t, engine)

	assert.True(t, engine.Match(map[string]any{"status": "1", "age": "20"}))
	assert.False(t, engine.Match(map[string]any{"status": "1", "age": "15"}))
	assert.False(t, engine.Match(map[string]any{"status": "0", "age": "20"}))
}

func TestFilterEngine_MissingField(t *testing.T) {
	engine := NewFilterEngine("status == 1")
	require.NotNil(t, engine)

	assert.False(t, engine.Match(map[string]any{"name": "alice"}), "missing field should not match")
}

// ========== parseFuncCall Tests ==========

func TestParseFuncCall_Simple(t *testing.T) {
	name, args, err := parseFuncCall("UPPER(name)")
	assert.NoError(t, err)
	assert.Equal(t, "UPPER", name)
	assert.Equal(t, []string{"name"}, args)
}

func TestParseFuncCall_MultipleArgs(t *testing.T) {
	name, args, err := parseFuncCall("CONCAT(a, b, c)")
	assert.NoError(t, err)
	assert.Equal(t, "CONCAT", name)
	assert.Equal(t, []string{"a", " b", " c"}, args)
}

func TestParseFuncCall_Nested(t *testing.T) {
	name, args, err := parseFuncCall("UPPER(TRIM(name))")
	assert.NoError(t, err)
	assert.Equal(t, "UPPER", name)
	assert.Equal(t, []string{"TRIM(name)"}, args)
}

func TestParseFuncCall_NotFunc(t *testing.T) {
	_, _, err := parseFuncCall("just_a_column")
	assert.Error(t, err)
}

// ========== splitArgs Tests ==========

func TestSplitArgs_Empty(t *testing.T) {
	assert.Nil(t, splitArgs(""))
}

func TestSplitArgs_Single(t *testing.T) {
	result := splitArgs("a")
	assert.Equal(t, []string{"a"}, result)
}

func TestSplitArgs_Multiple(t *testing.T) {
	result := splitArgs("a, b, c")
	assert.Equal(t, []string{"a", " b", " c"}, result)
}

func TestSplitArgs_NestedParens(t *testing.T) {
	result := splitArgs("TRIM(x), UPPER(y)")
	assert.Equal(t, []string{"TRIM(x)", " UPPER(y)"}, result)
}
