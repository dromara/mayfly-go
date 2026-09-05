package mysql

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 测试辅助：构造全新列（converter会原地修改col，每个用例须用新实例）
func newCol() *dbi.Column {
	return &dbi.Column{DataType: "varchar"}
}

// 各方言类型转换器映射矩阵：防止映射指错类型导致迁移数据截断/类型错乱

func TestMysqlConverter_IntMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	assert.Equal(t, Tinyint, c.Int1(col))
	assert.Equal(t, Smallint, c.Int2(col))
	assert.Equal(t, Int, c.Int4(col))
	assert.Equal(t, Bigint, c.Int8(col))
	assert.Equal(t, Bit, c.Bit(col))
	// mysql自身支持无符号，映射到对应无符号类型
	assert.Equal(t, UnsignedSmallint, c.UnsignedInt1(col))
	assert.Equal(t, UnsignedMediumint, c.UnsignedInt2(col))
	assert.Equal(t, UnsignedInt, c.UnsignedInt4(col))
	assert.Equal(t, UnsignedBigint, c.UnsignedInt8(col))
}

func TestMysqlConverter_VarcharMaxLengthBoundary(t *testing.T) {
	c := &commonTypeConverter{}

	// 16383及以内保持varchar
	col := newCol()
	col.CharMaxLength = 16383
	assert.Equal(t, Varchar, c.Varchar(col))
	assert.Equal(t, 16383, col.CharMaxLength)

	// 超过16383转text，并清空长度（text无长度语法）
	col2 := newCol()
	col2.CharMaxLength = 16384
	assert.Equal(t, Text, c.Varchar(col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

func TestMysqlConverter_TextFamilyClearLength(t *testing.T) {
	c := &commonTypeConverter{}

	col := newCol()
	col.CharMaxLength = 100
	col.NumPrecision = 10
	assert.Equal(t, Mediumtext, c.Mediumtext(col))
	assert.Equal(t, 0, col.CharMaxLength)
	assert.Equal(t, 0, col.NumPrecision)

	col2 := newCol()
	col2.CharMaxLength = 100
	assert.Equal(t, Longblob, c.Longblob(col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

func TestMysqlConverter_RepresentativeMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	assert.Equal(t, Date, c.Date(col))
	assert.Equal(t, Datetime, c.Datetime(col))
	assert.Equal(t, JSON, c.JSON(col))
	assert.Equal(t, Enum, c.Enum(col))
	assert.Equal(t, Blob, c.Blob(col))
	assert.Equal(t, Decimal, c.Decimal(col))
}

// 覆盖剩余简单映射（目标方言为mysql时的完整映射矩阵）
func TestMysqlConverter_RemainingMappings(t *testing.T) {
	c := &commonTypeConverter{}

	assert.Equal(t, Char, c.Char(newCol()))
	assert.Equal(t, Double, c.Numeric(newCol()))
	assert.Equal(t, Time, c.Time(newCol()))
	assert.Equal(t, Timestamp, c.Timestamp(newCol()))
	assert.Equal(t, Binary, c.Binary(newCol()))
	assert.Equal(t, Varbinary, c.Varbinary(newCol()))
	assert.Equal(t, Mediumblob, c.Mediumblob(newCol()))
	assert.Equal(t, Longtext, c.Longtext(newCol()))

	// text无长度语法，需清空源长度
	col := newCol()
	col.CharMaxLength = 100
	col.NumPrecision = 10
	assert.Equal(t, Text, c.Text(col))
	assert.Equal(t, 0, col.CharMaxLength)
	assert.Equal(t, 0, col.NumPrecision)
}
