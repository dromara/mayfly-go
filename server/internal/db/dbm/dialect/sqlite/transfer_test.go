package sqlite

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

// sqlite类型系统宽松（TEXT/INTEGER/REAL/BLOB），映射不产生长度/精度截断
func TestSqliteConverter_Mappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeSqlite)
	col := newCol()

	// 字符串族全部归一到TEXT
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCVarchar, col))
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCChar, col))
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCText, col))
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCLongtext, col))
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCEnum, col))
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCJSON, col))

	// 整数族全部归一到Integer（含无符号，Integer为8字节有符号但sqlite动态类型不截断）
	assert.Equal(t, Integer, resolveTarget(t, engine, dbi.TCInt1, col))
	assert.Equal(t, Integer, resolveTarget(t, engine, dbi.TCInt8, col))
	assert.Equal(t, Integer, resolveTarget(t, engine, dbi.TCUnsignedInt8, col))

	// 二进制族归一到Blob
	assert.Equal(t, Blob, resolveTarget(t, engine, dbi.TCBinary, col))
	assert.Equal(t, Blob, resolveTarget(t, engine, dbi.TCLongblob, col))

	// 定点数必须落NUMERIC亲和的numeric/decimal声明（降级为real会丢十进制精确性，
	// 并使后续往强类型库迁移时列类型从decimal退化为double）；浮点仍为Real
	assert.Equal(t, Numeric, resolveTarget(t, engine, dbi.TCNumeric, col))
	assert.Equal(t, Decimal, resolveTarget(t, engine, dbi.TCDecimal, col))

	// 时间
	assert.Equal(t, DateTime, resolveTarget(t, engine, dbi.TCDateTime, col))
	assert.Equal(t, DateTime, resolveTarget(t, engine, dbi.TCTimestamp, col))
}
