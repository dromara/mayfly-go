package postgres

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newCol() *dbi.Column {
	return &dbi.Column{DataType: "varchar"}
}

func TestPostgresConverter_IntMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	assert.Equal(t, Int2, c.Int1(col))
	assert.Equal(t, Int2, c.Int2(col))
	assert.Equal(t, Int4, c.Int4(col))
	assert.Equal(t, Int8, c.Int8(col))
	// pg无无符号类型，退化为有符号
	assert.Equal(t, Int2, c.UnsignedInt1(col))
	// 无符号需升位避免溢出：uint16>int2上限、uint32>int4上限、uint64>int8上限
	assert.Equal(t, Int4, c.UnsignedInt2(col))
	assert.Equal(t, Int8, c.UnsignedInt4(col))
	assert.Equal(t, Numeric, c.UnsignedInt8(col))
	// 整数族清空长度精度（WithFixColumn(ClearNumScale)副作用）
	assert.Equal(t, 0, col.NumPrecision)
	assert.Equal(t, 0, col.CharMaxLength)
}

func TestPostgresConverter_RepresentativeMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	// 字符串族：pg用DTStringPreserveSpecial（standard_conforming_strings=on下反斜杠不转义）
	assert.Equal(t, Varchar, c.Varchar(col))
	assert.Equal(t, Text, c.Mediumtext(col))
	assert.Equal(t, Text, c.Longtext(col))
	// 二进制族归一到bytea
	assert.Equal(t, Bytea, c.Blob(col))
	assert.Equal(t, Bytea, c.Longblob(col))
	// 时间族
	assert.Equal(t, Timestamp, c.Datetime(col))
	assert.Equal(t, Timestamp, c.Timestamp(col))
	// json
	assert.Equal(t, Json, c.JSON(col))
}
