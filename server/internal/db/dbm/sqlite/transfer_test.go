package sqlite

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newCol() *dbi.Column {
	return &dbi.Column{DataType: "varchar"}
}

// sqlite类型系统宽松（TEXT/INTEGER/REAL/BLOB），映射不产生长度/精度截断
func TestSqliteConverter_Mappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	// 字符串族全部归一到TEXT
	assert.Equal(t, Text, c.Varchar(col))
	assert.Equal(t, Text, c.Char(col))
	assert.Equal(t, Text, c.Text(col))
	assert.Equal(t, Text, c.Longtext(col))
	assert.Equal(t, Text, c.Enum(col))
	assert.Equal(t, Text, c.JSON(col))

	// 整数族全部归一到Integer（含无符号，Integer为8字节有符号但sqlite动态类型不截断）
	assert.Equal(t, Integer, c.Int1(col))
	assert.Equal(t, Integer, c.Int8(col))
	assert.Equal(t, Integer, c.UnsignedInt8(col))

	// 二进制族归一到Blob
	assert.Equal(t, Blob, c.Binary(col))
	assert.Equal(t, Blob, c.Longblob(col))

	// 定点数必须落NUMERIC亲和的numeric/decimal声明（降级为real会丢十进制精确性，
	// 并使后续往强类型库迁移时列类型从decimal退化为double）；浮点仍为Real
	assert.Equal(t, Numeric, c.Numeric(col))
	assert.Equal(t, Decimal, c.Decimal(col))

	// 时间
	assert.Equal(t, DateTime, c.Datetime(col))
	assert.Equal(t, DateTime, c.Timestamp(col))
}
