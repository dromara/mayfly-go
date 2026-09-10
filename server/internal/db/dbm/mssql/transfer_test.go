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

// 字符列边界：异构目标统一落nvarchar（Unicode）；超长（>4000字符）或长度未知（<=0）转nvarchar(max)
func TestMssqlConverter_VarcharMaxLengthBoundary(t *testing.T) {
	c := &commonTypeConverter{}

	// 4000字符为nvarchar(n)上限，边界内保真保留长度
	col := newCol()
	col.CharMaxLength = 4000
	assert.Equal(t, Nvarchar, c.Varchar(col))
	assert.Equal(t, 4000, col.CharMaxLength)

	// 4001起只能落nvarchar(max)：不得截断为nvarchar(4000)（那会静默丢掉第4001个字符往后的内容）
	col2 := newCol()
	col2.CharMaxLength = 4001
	assert.Equal(t, NvarcharMax, c.Varchar(col2))
	assert.Equal(t, 0, col2.CharMaxLength)

	// 源列长度远超nvarchar(n)上限（如mysql varchar(8000)/varchar(65535)）时同样只能落nvarchar(max)
	col3 := newCol()
	col3.CharMaxLength = 8000
	assert.Equal(t, NvarcharMax, c.Varchar(col3))
	col4 := newCol()
	col4.CharMaxLength = 65535
	assert.Equal(t, NvarcharMax, c.Varchar(col4))

	// 长度未知（pg/sqlite的不限长varchar）必须落nvarchar(max)：SQL Server的nvarchar不写长度
	// 即等价nvarchar(1)，会静默截断所有超长值且无任何报错（与bytesType对CharMaxLength<=0的处理一致）
	col5 := newCol()
	assert.Equal(t, NvarcharMax, c.Varchar(col5))
	col6 := newCol()
	col6.CharMaxLength = -1 // SQL Server对max列回报-1，防御性同样归为不限长
	assert.Equal(t, NvarcharMax, c.Varchar(col6))
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

// 大文本族清空长度并落nvarchar(max)（源text长度无意义，且varchar/text受库代码页限制不保真非ASCII）
func TestMssqlConverter_TextClearLength(t *testing.T) {
	c := &commonTypeConverter{}

	col := newCol()
	col.CharMaxLength = 500
	assert.Equal(t, NvarcharMax, c.Text(col))
	assert.Equal(t, 0, col.CharMaxLength)
}

func TestMssqlConverter_RepresentativeMappings(t *testing.T) {
	c := &commonTypeConverter{}
	col := newCol()

	assert.Equal(t, Date, c.Date(col))
	// datetime2而非datetime：datetime标度固定3.33ms且无精度语法，无法表达异构源的微秒小数秒
	assert.Equal(t, Datetime2, c.Datetime(col))
	assert.Equal(t, Datetime2, c.Timestamp(col))
	// enum元数据回报的列长为各枚举值最大字节数（>0），按有界字符列处理
	enumCol := newCol()
	enumCol.CharMaxLength = 20
	assert.Equal(t, Nvarchar, c.Enum(enumCol))
	assert.Equal(t, NvarcharMax, c.JSON(col))
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

	// 定长列保持定长语义（带长度时）；长度未知时不得落nchar（等价nchar(1)，会静默截断）
	charCol := newCol()
	charCol.CharMaxLength = 2
	assert.Equal(t, Nchar, c.Char(charCol))
	assert.Equal(t, NvarcharMax, c.Char(newCol()))
	// CTNumeric（浮点/无约束数值）落float而非numeric：numeric不带精度等价numeric(18,0)会截断小数
	assert.Equal(t, Float, c.Numeric(newCol()))
	assert.Equal(t, Time, c.Time(newCol()))

	// mediumtext/longtext同样落nvarchar(max)并清空长度
	col := newCol()
	col.CharMaxLength = 500
	assert.Equal(t, NvarcharMax, c.Mediumtext(col))
	assert.Equal(t, 0, col.CharMaxLength)

	col2 := newCol()
	col2.CharMaxLength = 500
	assert.Equal(t, NvarcharMax, c.Longtext(col2))
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

	// varchar(8001) → nvarchar(max)（nvarchar(n)上限4000字符）
	longVarchar := &dbi.Column{DataType: "varchar", CharMaxLength: 8001}
	err = dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeMssql, dialect, longVarchar)
	assert.NoError(t, err)
	assert.Equal(t, "nvarchar(max)", longVarchar.DataType)
	assert.Equal(t, "nvarchar(max)", longVarchar.GetColumnType())

	// varchar(50) → nvarchar(50) 保留字符长度
	normalVarchar := &dbi.Column{DataType: "varchar", CharMaxLength: 50}
	err = dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeMssql, dialect, normalVarchar)
	assert.NoError(t, err)
	assert.Equal(t, "nvarchar", normalVarchar.DataType)
	assert.Equal(t, "nvarchar(50)", normalVarchar.GetColumnType())

	// 值转SQL接线：异构目标列类型名必须能在该方言类型注册表中命中，
	// 否则GenInsert按名取类型会回退默认字符串类型（跨方言守卫见 dbm.TestConverterOutputTypesRegistered）
	for _, cc := range []*dbi.Column{blobCol, longVarchar, normalVarchar} {
		resolved := dbi.GetDbDataType(DbTypeMssql, cc.DataType)
		assert.NotSame(t, dbi.DefaultDbDataType, resolved, "类型[%s]未注册，值转SQL将静默回退默认字符串类型", cc.DataType)
		assert.Equal(t, cc.DataType, resolved.Name, "类型[%s]命中了错误的注册类型", cc.DataType)
	}
}
