package mysql

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试辅助：构造全新列（resolver会原地修改col，每个用例须用新实例）
func newCol() *dbi.Column {
	return &dbi.Column{DataType: "varchar"}
}

func resolveTarget(t *testing.T, engine *dbi.TypeEngine, cat dbi.TypeCategory, col *dbi.Column) *dbi.DbDataType {
	t.Helper()
	result, err := engine.ResolveTarget(cat, col)
	require.NoError(t, err)
	return result
}

// 各方言类型转换器映射矩阵：防止映射指错类型导致迁移数据截断/类型错乱

func TestMysqlConverter_IntMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeMysql)
	col := newCol()

	assert.Equal(t, Tinyint, resolveTarget(t, engine, dbi.TCInt1, col))
	assert.Equal(t, Smallint, resolveTarget(t, engine, dbi.TCInt2, col))
	assert.Equal(t, Int, resolveTarget(t, engine, dbi.TCInt4, col))
	assert.Equal(t, Bigint, resolveTarget(t, engine, dbi.TCInt8, col))
	assert.Equal(t, Bit, resolveTarget(t, engine, dbi.TCBit, col))
	// mysql自身支持无符号，映射到对应无符号类型
	assert.Equal(t, UnsignedSmallint, resolveTarget(t, engine, dbi.TCUnsignedInt1, col))
	assert.Equal(t, UnsignedMediumint, resolveTarget(t, engine, dbi.TCUnsignedInt2, col))
	assert.Equal(t, UnsignedInt, resolveTarget(t, engine, dbi.TCUnsignedInt4, col))
	assert.Equal(t, UnsignedBigint, resolveTarget(t, engine, dbi.TCUnsignedInt8, col))
}

func TestMysqlConverter_VarcharMaxLengthBoundary(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeMysql)

	// 16383及以内保持varchar
	col := newCol()
	col.CharMaxLength = 16383
	assert.Equal(t, Varchar, resolveTarget(t, engine, dbi.TCVarchar, col))
	assert.Equal(t, 16383, col.CharMaxLength)

	// 超过16383转text，并清空长度（text无长度语法）
	col2 := newCol()
	col2.CharMaxLength = 16384
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCVarchar, col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

func TestMysqlConverter_TextFamilyClearLength(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeMysql)

	col := newCol()
	col.CharMaxLength = 100
	col.NumPrecision = 10
	assert.Equal(t, Mediumtext, resolveTarget(t, engine, dbi.TCMediumtext, col))
	assert.Equal(t, 0, col.CharMaxLength)
	assert.Equal(t, 0, col.NumPrecision)

	col2 := newCol()
	col2.CharMaxLength = 100
	assert.Equal(t, Longblob, resolveTarget(t, engine, dbi.TCLongblob, col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

func TestMysqlConverter_RepresentativeMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeMysql)
	col := newCol()

	assert.Equal(t, Date, resolveTarget(t, engine, dbi.TCDate, col))
	assert.Equal(t, Datetime, resolveTarget(t, engine, dbi.TCDateTime, col))
	assert.Equal(t, JSON, resolveTarget(t, engine, dbi.TCJSON, col))
	assert.Equal(t, Enum, resolveTarget(t, engine, dbi.TCEnum, col))
	assert.Equal(t, Blob, resolveTarget(t, engine, dbi.TCBlob, col))
	assert.Equal(t, Decimal, resolveTarget(t, engine, dbi.TCDecimal, col))
}

// 覆盖剩余简单映射（目标方言为mysql时的完整映射矩阵）
func TestMysqlConverter_RemainingMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeMysql)

	assert.Equal(t, Char, resolveTarget(t, engine, dbi.TCChar, newCol()))
	assert.Equal(t, Double, resolveTarget(t, engine, dbi.TCNumeric, newCol()))
	assert.Equal(t, Time, resolveTarget(t, engine, dbi.TCTime, newCol()))
	assert.Equal(t, Timestamp, resolveTarget(t, engine, dbi.TCTimestamp, newCol()))
	assert.Equal(t, Varbinary, resolveTarget(t, engine, dbi.TCVarbinary, newCol()))
	// binary无长度信息时降级为blob（mysql "binary"默认binary(1)会截断数据）
	assert.Equal(t, Blob, resolveTarget(t, engine, dbi.TCBinary, newCol()))
	lenCol := newCol()
	lenCol.CharMaxLength = 16
	assert.Equal(t, Binary, resolveTarget(t, engine, dbi.TCBinary, lenCol))
	assert.Equal(t, Mediumblob, resolveTarget(t, engine, dbi.TCMediumblob, newCol()))
	assert.Equal(t, Longtext, resolveTarget(t, engine, dbi.TCLongtext, newCol()))

	// text无长度语法，需清空源长度
	col := newCol()
	col.CharMaxLength = 100
	col.NumPrecision = 10
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCText, col))
	assert.Equal(t, 0, col.CharMaxLength)
	assert.Equal(t, 0, col.NumPrecision)
}
