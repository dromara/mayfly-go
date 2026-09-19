package postgres

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCol() *dbi.Column {
	return &dbi.Column{DataType: "varchar"}
}

func resolveTarget(t *testing.T, engine *dbi.TypeEngine, cat dbi.TypeCategory, col *dbi.Column) *dbi.DbDataType {
	t.Helper()
	result, err := engine.ResolveTarget(cat, col)
	require.NoError(t, err)
	return result
}

func TestPostgresConverter_IntMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypePostgres)
	col := newCol()

	assert.Equal(t, Int2, resolveTarget(t, engine, dbi.TCInt1, col))
	assert.Equal(t, Int2, resolveTarget(t, engine, dbi.TCInt2, col))
	assert.Equal(t, Int4, resolveTarget(t, engine, dbi.TCInt4, col))
	assert.Equal(t, Int8, resolveTarget(t, engine, dbi.TCInt8, col))
	// pg无无符号类型，退化为有符号
	assert.Equal(t, Int2, resolveTarget(t, engine, dbi.TCUnsignedInt1, col))
	// 无符号需升位避免溢出：uint16>int2上限、uint32>int4上限、uint64>int8上限
	assert.Equal(t, Int4, resolveTarget(t, engine, dbi.TCUnsignedInt2, col))
	assert.Equal(t, Int8, resolveTarget(t, engine, dbi.TCUnsignedInt4, col))
	assert.Equal(t, Numeric, resolveTarget(t, engine, dbi.TCUnsignedInt8, col))
	// 整数族清空长度精度（cleanupColumnParams副作用）
	assert.Equal(t, 0, col.NumPrecision)
	assert.Equal(t, 0, col.CharMaxLength)
}

func TestPostgresConverter_RepresentativeMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypePostgres)
	col := newCol()

	// 字符串族：pg用DTStringPreserveSpecial（standard_conforming_strings=on下反斜杠不转义）
	assert.Equal(t, Varchar, resolveTarget(t, engine, dbi.TCVarchar, col))
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCMediumtext, col))
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCLongtext, col))
	// 二进制族归一到bytea
	assert.Equal(t, Bytea, resolveTarget(t, engine, dbi.TCBlob, col))
	assert.Equal(t, Bytea, resolveTarget(t, engine, dbi.TCLongblob, col))
	// 时间族
	assert.Equal(t, Timestamp, resolveTarget(t, engine, dbi.TCDateTime, col))
	assert.Equal(t, Timestamp, resolveTarget(t, engine, dbi.TCTimestamp, col))
	// json
	assert.Equal(t, Json, resolveTarget(t, engine, dbi.TCJSON, col))
}
