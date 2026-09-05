package dm

import (
	"mayfly-go/internal/db/dbm/dbi"
	mysql "mayfly-go/internal/db/dbm/mysql" // 触发mysql源类型注册
	"testing"

	"github.com/stretchr/testify/assert"
)

func newCol() *dbi.Column {
	return &dbi.Column{DataType: "VARCHAR"}
}

func dmTargetDialect() dbi.Dialect {
	meta := dbi.GetMeta(DbTypeDM)
	conn := &dbi.DbConn{Info: &dbi.DbInfo{Type: DbTypeDM, Meta: meta}}
	return conn.GetDialect()
}

func TestDmConverter_IntMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	assert.Equal(t, TINYINT, c.Int1(col))
	assert.Equal(t, SMALLINT, c.Int2(col))
	assert.Equal(t, INTEGER, c.Int4(col))
	assert.Equal(t, BIGINT, c.Int8(col))
	assert.Equal(t, BIT, c.Bit(col))
	// dm无无符号类型，退化为有符号；无符号小整型统一升格INT防溢出
	assert.Equal(t, INT, c.UnsignedInt1(col))
	assert.Equal(t, INT, c.UnsignedInt2(col))
	assert.Equal(t, INT, c.UnsignedInt4(col))
	assert.Equal(t, BIGINT, c.UnsignedInt8(col))
	// 整数族必须清空长度精度（clearLength副作用）
	assert.Equal(t, 0, col.CharMaxLength)
	assert.Equal(t, 0, col.NumPrecision)
}

// varchar超长边界：dm VARCHAR上限32767，超长必须转TEXT
func TestDmConverter_VarcharMaxLengthBoundary(t *testing.T) {
	c := &commonTypeConverter{}

	col := newCol()
	col.CharMaxLength = 32767
	assert.Equal(t, VARCHAR, c.Varchar(col))

	col2 := newCol()
	col2.CharMaxLength = 32768
	assert.Equal(t, TEXT, c.Varchar(col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

func TestDmConverter_RepresentativeMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	assert.Equal(t, LONGVARCHAR, c.Longtext(col))
	assert.Equal(t, TEXT, c.Text(col))
	assert.Equal(t, BLOB, c.Blob(col))
	assert.Equal(t, BLOB, c.Longblob(col))
	assert.Equal(t, VARCHAR, c.Enum(col))
	assert.Equal(t, VARCHAR, c.JSON(col))
	assert.Equal(t, DECIMAL, c.Decimal(col))
}

// 覆盖剩余映射（目标方言为dm时的完整映射矩阵）
func TestDmConverter_RemainingMappings(t *testing.T) {
	c := &commonTypeConverter{}

	assert.Equal(t, CHAR, c.Char(newCol()))
	assert.Equal(t, NUMBER, c.Numeric(newCol()))
	assert.Equal(t, BLOB, c.Binary(newCol()))
	assert.Equal(t, BLOB, c.Varbinary(newCol()))
	assert.Equal(t, BLOB, c.Mediumblob(newCol()))

	// 时间族映射及clearLength副作用（时间类型无长度语法）
	col := newCol()
	col.CharMaxLength = 100
	col.NumPrecision = 10
	assert.Equal(t, DATE, c.Date(col))
	assert.Equal(t, TIME, c.Time(col))
	assert.Equal(t, DATETIME, c.Datetime(col))
	assert.Equal(t, TIMESTAMP, c.Timestamp(col))
	assert.Equal(t, 0, col.CharMaxLength)
	assert.Equal(t, 0, col.NumPrecision)

	// mediumtext归一化为text并清空长度（text无长度语法）
	col2 := newCol()
	col2.CharMaxLength = 500
	assert.Equal(t, TEXT, c.Mediumtext(col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

// 端到端回归：mysql 超长varchar 迁移至 dm 的完整链路（历史上曾生成非法DDL VARCHAR(50000)）
func TestDmConvToTargetDbColumn_FromMysql(t *testing.T) {
	dbi.GetMeta(mysql.DbTypeMysql)
	dialect := dmTargetDialect()

	longVarchar := &dbi.Column{DataType: "varchar", CharMaxLength: 50000}
	err := dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeDM, dialect, longVarchar)
	assert.NoError(t, err)
	assert.Equal(t, "TEXT", longVarchar.DataType)
	assert.Equal(t, "TEXT", longVarchar.GetColumnType())

	// 正常长度保留
	normalVarchar := &dbi.Column{DataType: "varchar", CharMaxLength: 50}
	err = dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeDM, dialect, normalVarchar)
	assert.NoError(t, err)
	assert.Equal(t, "VARCHAR(50)", normalVarchar.GetColumnType())
}
