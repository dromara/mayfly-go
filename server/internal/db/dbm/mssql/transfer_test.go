package mssql

import (
	"mayfly-go/internal/db/dbm/dbi"
	mysql "mayfly-go/internal/db/dbm/mysql" // 触发mysql源类型注册
	"testing"

	"github.com/stretchr/testify/assert"
)

func newCol() *dbi.Column {
	return &dbi.Column{DataType: "varchar"}
}

// mssql目标dialect（伪连接，无真实数据库）
func mssqlTargetDialect() dbi.Dialect {
	meta := dbi.GetMeta(DbTypeMssql)
	conn := &dbi.DbConn{Info: &dbi.DbInfo{Type: DbTypeMssql, Meta: meta}}
	return conn.GetDialect()
}

// 整数族映射：防止映射指错类型导致迁移数据截断
func TestMssqlConverter_IntMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	assert.Equal(t, Tinyint, c.Int1(col))
	assert.Equal(t, Smallint, c.Int2(col))
	assert.Equal(t, Int, c.Int4(col))
	assert.Equal(t, Bigint, c.Int8(col))
	assert.Equal(t, Bit, c.Bit(col))
	// mssql无无符号类型，退化为容量更大的有符号类型
	assert.Equal(t, Tinyint, c.UnsignedInt1(col))
	assert.Equal(t, Smallint, c.UnsignedInt2(col))
	assert.Equal(t, Int, c.UnsignedInt4(col))
	assert.Equal(t, Bigint, c.UnsignedInt8(col))
}

// varchar超长边界：varchar(n)上限8000字节，超长必须转varchar(max)
func TestMssqlConverter_VarcharMaxLengthBoundary(t *testing.T) {
	c := &commonTypeConverter{}

	col := newCol()
	col.CharMaxLength = 8000
	assert.Equal(t, Varchar, c.Varchar(col))
	assert.Equal(t, 8000, col.CharMaxLength)

	col2 := newCol()
	col2.CharMaxLength = 8001
	assert.Equal(t, VarcharMax, c.Varchar(col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

// 二进制类型归一化：无长度或超长转varbinary(max)，避免binary(1)默认长度或binary(n)>8000非法DDL
func TestMssqlConverter_BytesTypeBoundary(t *testing.T) {
	c := &commonTypeConverter{}

	// 源blob无长度（最常见）→ varbinary(max)
	col := newCol()
	assert.Equal(t, VarbinaryMax, c.Longblob(col))

	// 超过8000 → varbinary(max)
	col2 := newCol()
	col2.CharMaxLength = 16777215
	assert.Equal(t, VarbinaryMax, c.Blob(col2))
	assert.Equal(t, 0, col2.CharMaxLength)

	// 有长度且不超限 → varbinary(n)（保真，不用定长binary避免pad 0x00）
	col3 := newCol()
	col3.CharMaxLength = 16
	assert.Equal(t, Varbinary, c.Binary(col3))
	assert.Equal(t, Varbinary, c.Varbinary(col3))

	// blob族全部走归一化
	col4 := newCol()
	assert.Equal(t, VarbinaryMax, c.Mediumblob(col4))
	col5 := newCol()
	assert.Equal(t, VarbinaryMax, c.Varbinary(col5))
}

// text族清空长度（text类型无长度语法）
func TestMssqlConverter_TextClearLength(t *testing.T) {
	c := &commonTypeConverter{}

	col := newCol()
	col.CharMaxLength = 500
	assert.Equal(t, Text, c.Text(col))
	assert.Equal(t, 0, col.CharMaxLength)
}

func TestMssqlConverter_RepresentativeMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	assert.Equal(t, Date, c.Date(col))
	// datetime2而非datetime：datetime标度固定3.33ms且无精度语法，无法表达异构源的微秒小数秒
	assert.Equal(t, Datetime2, c.Datetime(col))
	assert.Equal(t, Datetime2, c.Timestamp(col))
	assert.Equal(t, Varchar, c.Enum(col))
	assert.Equal(t, Text, c.JSON(col))
	assert.Equal(t, Decimal, c.Decimal(col))
	// 无精度约束的源列必须补齐(38,19)：mssql的decimal不声明精度即decimal(18,0)，会静默截断小数
	unbounded := newCol()
	assert.Equal(t, Decimal, c.Decimal(unbounded))
	assert.Equal(t, 38, unbounded.NumPrecision)
	assert.Equal(t, 19, unbounded.NumScale)

	// datetime2保留并收敛源列小数秒精度（mssql上限7），且时间类型不得残留标度
	timeCol := newCol()
	timeCol.NumPrecision, timeCol.NumScale = 3, 2
	assert.Equal(t, Datetime2, c.Datetime(timeCol))
	assert.Equal(t, 3, timeCol.NumPrecision)
	assert.Equal(t, 0, timeCol.NumScale, "时间类型的标度无意义，必须清零避免拼出datetime2(3,2)")
	assert.Equal(t, Datetime2, c.Timestamp(&dbi.Column{NumPrecision: 9}))
	dateCol := newCol()
	dateCol.NumPrecision = 3
	c.Date(dateCol)
	assert.Equal(t, 0, dateCol.NumPrecision, "date不接受精度参数，必须清空")
}

// 覆盖剩余简单映射（目标方言为mssql时的完整映射矩阵）
func TestMssqlConverter_RemainingMappings(t *testing.T) {
	c := &commonTypeConverter{}

	assert.Equal(t, Char, c.Char(newCol()))
	// CTNumeric（浮点/无约束数值）落float而非numeric：numeric不带精度等价numeric(18,0)会截断小数
	assert.Equal(t, Float, c.Numeric(newCol()))
	assert.Equal(t, Time, c.Time(newCol()))

	// mediumtext/longtext同样归一化为text并清空长度（text无长度语法）
	col := newCol()
	col.CharMaxLength = 500
	assert.Equal(t, Text, c.Mediumtext(col))
	assert.Equal(t, 0, col.CharMaxLength)

	col2 := newCol()
	col2.CharMaxLength = 500
	assert.Equal(t, Text, c.Longtext(col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

// 端到端回归：mysql longtext/超长varchar 迁移至 mssql 的完整链路（源类型注册→CT匹配→converter→DataType替换）
func TestMssqlConvToTargetDbColumn_FromMysql(t *testing.T) {
	dbi.GetMeta(mysql.DbTypeMysql)
	dialect := mssqlTargetDialect()

	// longblob（无长度，CTBinary）→ varbinary(max)，历史上曾生成 binary(1) 导致插入必失败
	blobCol := &dbi.Column{DataType: "longblob"}
	err := dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeMssql, dialect, blobCol)
	assert.NoError(t, err)
	assert.Equal(t, "varbinary(max)", blobCol.DataType)
	assert.Equal(t, "varbinary(max)", blobCol.GetColumnType())

	// varchar(8001) → varchar(max)，历史上曾生成非法DDL varchar(8001)
	longVarchar := &dbi.Column{DataType: "varchar", CharMaxLength: 8001}
	err = dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeMssql, dialect, longVarchar)
	assert.NoError(t, err)
	assert.Equal(t, "varchar(max)", longVarchar.DataType)
	assert.Equal(t, "varchar(max)", longVarchar.GetColumnType())

	// varchar(50) → varchar(50) 正常保留长度
	normalVarchar := &dbi.Column{DataType: "varchar", CharMaxLength: 50}
	err = dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeMssql, dialect, normalVarchar)
	assert.NoError(t, err)
	assert.Equal(t, "varchar", normalVarchar.DataType)
	assert.Equal(t, "varchar(50)", normalVarchar.GetColumnType())
}
