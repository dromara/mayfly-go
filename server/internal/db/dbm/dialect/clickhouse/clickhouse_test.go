package clickhouse

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClickHouseMetaRegistration(t *testing.T) {
	// Test that ClickHouse meta is registered
	meta := dbi.GetBackend(DbTypeClickHouse)
	assert.NotNil(t, meta, "ClickHouse meta should be registered")

	// Test that ClickHouse dialect can be obtained
	dialect := dbi.GetDialect(DbTypeClickHouse)
	assert.NotNil(t, dialect, "ClickHouse dialect should be available")

	// Test that ClickHouse has data types registered via TypeEngine
	engine := dbi.GetTypeEngine(DbTypeClickHouse)
	assert.NotNil(t, engine, "ClickHouse type engine should be registered")
	dataTypes := engine.Types()
	assert.Greater(t, len(dataTypes), 0, "ClickHouse should have data types")
}

func TestClickHouseDataTypes(t *testing.T) {
	// Test specific data types
	assert.NotNil(t, UInt8, "UInt8 should be defined")
	assert.NotNil(t, UInt16, "UInt16 should be defined")
	assert.NotNil(t, UInt32, "UInt32 should be defined")
	assert.NotNil(t, UInt64, "UInt64 should be defined")
	assert.NotNil(t, Int8, "Int8 should be defined")
	assert.NotNil(t, Int16, "Int16 should be defined")
	assert.NotNil(t, Int32, "Int32 should be defined")
	assert.NotNil(t, Int64, "Int64 should be defined")
	assert.NotNil(t, Float32, "Float32 should be defined")
	assert.NotNil(t, Float64, "Float64 should be defined")
	assert.NotNil(t, String, "String should be defined")
	assert.NotNil(t, FixedString, "FixedString should be defined")
	assert.NotNil(t, DateTime, "DateTime should be defined")
	assert.NotNil(t, DateTime64, "DateTime64 should be defined")
	assert.NotNil(t, Date, "Date should be defined")
	assert.NotNil(t, Date32, "Date32 should be defined")
	assert.NotNil(t, UUID, "UUID should be defined")
	assert.NotNil(t, IPv4, "IPv4 should be defined")
	assert.NotNil(t, IPv6, "IPv6 should be defined")
	assert.NotNil(t, Bool, "Bool should be defined")
	assert.NotNil(t, Decimal, "Decimal should be defined")
	assert.NotNil(t, Decimal32, "Decimal32 should be defined")
	assert.NotNil(t, Decimal64, "Decimal64 should be defined")
	assert.NotNil(t, Decimal128, "Decimal128 should be defined")
	assert.NotNil(t, Decimal256, "Decimal256 should be defined")
	assert.NotNil(t, Enum8, "Enum8 should be defined")
	assert.NotNil(t, Enum16, "Enum16 should be defined")
	assert.NotNil(t, Array, "Array should be defined")
	assert.NotNil(t, Tuple, "Tuple should be defined")
	assert.NotNil(t, Map, "Map should be defined")
	assert.NotNil(t, Nested, "Nested should be defined")
	assert.NotNil(t, AggregateFunction, "AggregateFunction should be defined")
	assert.NotNil(t, SimpleAggregateFunction, "SimpleAggregateFunction should be defined")
	assert.NotNil(t, LowCardinality, "LowCardinality should be defined")
	assert.NotNil(t, Nullable, "Nullable should be defined")
}

func TestClickHouseTypeEngineRules(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeClickHouse)
	column := &dbi.Column{}

	// Test conversion of common types via TypeEngine
	result, err := engine.ResolveTarget(dbi.TCInt1, column)
	assert.NoError(t, err)
	assert.Equal(t, Int8, result, "TCInt1 should convert to Int8")

	result, err = engine.ResolveTarget(dbi.TCInt2, column)
	assert.NoError(t, err)
	assert.Equal(t, Int16, result, "TCInt2 should convert to Int16")

	result, err = engine.ResolveTarget(dbi.TCInt4, column)
	assert.NoError(t, err)
	assert.Equal(t, Int32, result, "TCInt4 should convert to Int32")

	result, err = engine.ResolveTarget(dbi.TCInt8, column)
	assert.NoError(t, err)
	assert.Equal(t, Int64, result, "TCInt8 should convert to Int64")

	result, err = engine.ResolveTarget(dbi.TCVarchar, column)
	assert.NoError(t, err)
	assert.Equal(t, String, result, "TCVarchar should convert to String")

	result, err = engine.ResolveTarget(dbi.TCChar, column)
	// clickhouse不使用FixedString承载：必须显式指定长度且\0定长填充会改变数据语义
	assert.NoError(t, err)
	assert.Equal(t, String, result, "TCChar should convert to String")

	result, err = engine.ResolveTarget(dbi.TCDate, column)
	assert.NoError(t, err)
	assert.Equal(t, Date, result, "TCDate should convert to Date")

	result, err = engine.ResolveTarget(dbi.TCDateTime, column)
	assert.NoError(t, err)
	assert.Equal(t, DateTime, result, "TCDateTime should convert to DateTime")

	result, err = engine.ResolveTarget(dbi.TCBool, column)
	assert.NoError(t, err)
	assert.Equal(t, Bool, result, "TCBool should convert to Bool")

	result, err = engine.ResolveTarget(dbi.TCDecimal, column)
	assert.NoError(t, err)
	assert.Equal(t, Decimal, result, "TCDecimal should convert to Decimal")
}
