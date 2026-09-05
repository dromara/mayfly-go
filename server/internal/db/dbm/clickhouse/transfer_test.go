package clickhouse

import (
	"mayfly-go/internal/db/dbm/dbi"
	mysql "mayfly-go/internal/db/dbm/mysql" // 触发mysql源类型注册
	"testing"

	"github.com/stretchr/testify/assert"
)

func newCol() *dbi.Column {
	return &dbi.Column{DataType: "varchar"}
}

func clickhouseTargetDialect() dbi.Dialect {
	meta := dbi.GetMeta(DbTypeClickHouse)
	conn := &dbi.DbConn{Info: &dbi.DbInfo{Type: DbTypeClickHouse, Meta: meta}}
	return conn.GetDialect()
}

func TestClickHouseConverter_IntMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	assert.Equal(t, Int8, c.Int1(col))
	assert.Equal(t, Int16, c.Int2(col))
	assert.Equal(t, Int32, c.Int4(col))
	assert.Equal(t, Int64, c.Int8(col))
	// 无符号映射回归：unsigned bigint必须为UInt64（曾误映射UInt8导致大于255的值插入失败）
	assert.Equal(t, UInt8, c.UnsignedInt1(col))
	assert.Equal(t, UInt16, c.UnsignedInt2(col))
	assert.Equal(t, UInt32, c.UnsignedInt4(col))
	assert.Equal(t, UInt64, c.UnsignedInt8(col))
}

func TestClickHouseConverter_RepresentativeMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	// 字符串族归一到String（无长度限制，无截断风险）
	assert.Equal(t, String, c.Varchar(col))
	assert.Equal(t, String, c.Longtext(col))
	// clickhouse不使用FixedString承载：必须显式指定长度且\0定长填充会改变数据语义
	assert.Equal(t, String, c.Char(col))
	// 二进制以String承载
	assert.Equal(t, String, c.Blob(col))
	assert.Equal(t, String, c.Longblob(col))
	// 时间
	assert.Equal(t, Date, c.Date(col))
	assert.Equal(t, DateTime, c.Datetime(col))
	assert.Equal(t, DateTime, c.Timestamp(col))
	// 其他
	assert.Equal(t, Float64, c.Numeric(col))
	assert.Equal(t, Bool, c.Bool(col))
	assert.Equal(t, String, c.JSON(col))
}

// 剩余映射矩阵（目标方言为clickhouse时），含clearLength副作用断言
func TestClickHouseConverter_RemainingMappings(t *testing.T) {
	c := &commonTypeConverter{}

	assert.Equal(t, String, c.Mediumtext(newCol()))
	assert.Equal(t, String, c.Text(newCol()))
	assert.Equal(t, String, c.Binary(newCol()))
	assert.Equal(t, String, c.Varbinary(newCol()))
	assert.Equal(t, String, c.Mediumblob(newCol()))
	// Enum8必须显式声明枚举值集合，源枚举定义不可知，退化为String
	assert.Equal(t, String, c.Enum(newCol()))

	// clickhouse类型不携带长度/精度，必须清空源长度信息
	col := newCol()
	col.CharMaxLength = 100
	col.NumPrecision = 10
	col.NumScale = 2
	assert.Equal(t, String, c.Varchar(col))
	assert.Equal(t, 0, col.CharMaxLength)
	assert.Equal(t, 0, col.NumPrecision)
	assert.Equal(t, 0, col.NumScale)

	// Decimal必须保留精度（Decimal(P,S)参数化）
	dec := newCol()
	dec.NumPrecision = 12
	dec.NumScale = 4
	assert.Equal(t, Decimal, c.Decimal(dec))
	assert.Equal(t, 12, dec.NumPrecision)
	assert.Equal(t, 4, dec.NumScale)
}

// 端到端回归：mysql各类列迁移至clickhouse的完整链路（源类型注册→CT匹配→converter→长度清理）
func TestClickHouseConvToTargetDbColumn_FromMysql(t *testing.T) {
	dbi.GetMeta(mysql.DbTypeMysql)
	dialect := clickhouseTargetDialect()

	// varchar(50) → String，且无长度残留（避免生成非法的String(50)）
	varcharCol := &dbi.Column{DataType: "varchar", CharMaxLength: 50}
	err := dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeClickHouse, dialect, varcharCol)
	assert.NoError(t, err)
	assert.Equal(t, "String", varcharCol.DataType)
	assert.Equal(t, "String", varcharCol.GetColumnType())

	// unsigned smallint → UInt16（mysql源无符号类型名为"unsigned xxx"风格）
	usmallCol := &dbi.Column{DataType: "unsigned smallint"}
	err = dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeClickHouse, dialect, usmallCol)
	assert.NoError(t, err)
	assert.Equal(t, "UInt16", usmallCol.DataType)

	// unsigned bigint → UInt64（无符号bigint回归）
	ubigCol := &dbi.Column{DataType: "unsigned bigint"}
	err = dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeClickHouse, dialect, ubigCol)
	assert.NoError(t, err)
	assert.Equal(t, "UInt64", ubigCol.DataType)

	// decimal(12,4) → Decimal并保留精度
	decCol := &dbi.Column{DataType: "decimal", NumPrecision: 12, NumScale: 4}
	err = dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeClickHouse, dialect, decCol)
	assert.NoError(t, err)
	assert.Equal(t, "Decimal", decCol.DataType)
	assert.Equal(t, "Decimal(12,4)", decCol.GetColumnType())
}
