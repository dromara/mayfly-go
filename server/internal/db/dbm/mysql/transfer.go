package mysql

import "mayfly-go/internal/db/dbm/dbi"

var _ dbi.CommonTypeConverter = (*commonTypeConverter)(nil)

type commonTypeConverter struct {
}

func (c *commonTypeConverter) Varchar(col *dbi.Column) *dbi.DbDataType {
	// 如果字符长度大于16383，则转为text类型
	if col.CharMaxLength > 16383 {
		col.CharMaxLength = 0
		return Text
	}
	return Varchar
}

func (c *commonTypeConverter) Char(col *dbi.Column) *dbi.DbDataType {
	return Char
}
func (c *commonTypeConverter) Text(col *dbi.Column) *dbi.DbDataType {
	col.CharMaxLength = 0
	col.NumPrecision = 0
	return Text
}
func (c *commonTypeConverter) Mediumtext(col *dbi.Column) *dbi.DbDataType {
	col.CharMaxLength = 0
	col.NumPrecision = 0
	return Mediumtext
}
func (c *commonTypeConverter) Longtext(col *dbi.Column) *dbi.DbDataType {
	col.CharMaxLength = 0
	col.NumPrecision = 0
	return Longtext
}

func (c *commonTypeConverter) Bit(col *dbi.Column) *dbi.DbDataType {
	return Bit
}
func (c *commonTypeConverter) Bool(col *dbi.Column) *dbi.DbDataType {
	// mysql无原生布尔类型，惯用tinyint(1)表示
	return Tinyint
}
func (c *commonTypeConverter) Int1(col *dbi.Column) *dbi.DbDataType {
	return Tinyint
}
func (c *commonTypeConverter) Int2(col *dbi.Column) *dbi.DbDataType {
	return Smallint
}
func (c *commonTypeConverter) Int4(col *dbi.Column) *dbi.DbDataType {
	return Int
}
func (c *commonTypeConverter) Int8(col *dbi.Column) *dbi.DbDataType {
	return Bigint
}
func (c *commonTypeConverter) Numeric(col *dbi.Column) *dbi.DbDataType {
	// double不接受精度参数（8.0.17已移除M,D形式），且MySQL的double可无损承载任意浮点/无界数值源列，
	// 故清空源列精度避免生成 double(22) 这类非法/无意义DDL
	dbi.ClearNumPrecision(col)
	return Double
}

// Decimal MySQL的decimal省略精度等价decimal(10,0)，会把源列小数静默截断为整数：
// 源为无界精确数值（pg的numeric、oracle的NUMBER、sqlite声明的numeric）时必须补齐MySQL最大精度(65,30)，
// 同时收敛超出MySQL上限的极端精度（宁可插入时报范围错误，也不静默丢小数）
func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
	dbi.FillUnboundedDecimal(col, 65, 30)
	dbi.ClampDecimalPrecision(col, 65, 30)
	return Decimal
}

func (c *commonTypeConverter) UnsignedInt8(col *dbi.Column) *dbi.DbDataType {
	return UnsignedBigint
}
func (c *commonTypeConverter) UnsignedInt4(col *dbi.Column) *dbi.DbDataType {
	return UnsignedInt
}
func (c *commonTypeConverter) UnsignedInt2(col *dbi.Column) *dbi.DbDataType {
	return UnsignedMediumint
}
func (c *commonTypeConverter) UnsignedInt1(col *dbi.Column) *dbi.DbDataType {
	return UnsignedSmallint
}

func (c *commonTypeConverter) Date(col *dbi.Column) *dbi.DbDataType {
	// MySQL的date不接受任何括号参数，源列残留精度会生成 date(3) 非法DDL
	dbi.ClearNumPrecision(col)
	return Date
}

func (c *commonTypeConverter) Time(col *dbi.Column) *dbi.DbDataType {
	// MySQL的time省略fsp即time(0)，会静默丢失源值小数秒，故按最大fsp(6)归一
	dbi.NormalizeTimeFsp(col, 6)
	return Time
}
func (c *commonTypeConverter) Datetime(col *dbi.Column) *dbi.DbDataType {
	// 同上：MySQL的datetime/timestamp省略fsp即秒精度，异构迁移必须保留源列小数秒，否则数据静默失真
	dbi.NormalizeTimeFsp(col, 6)
	return Datetime
}
func (c *commonTypeConverter) Timestamp(col *dbi.Column) *dbi.DbDataType {
	dbi.NormalizeTimeFsp(col, 6)
	return Timestamp
}

func (c *commonTypeConverter) Binary(col *dbi.Column) *dbi.DbDataType {
	// 源列无长度信息时（如pg bytea无character_maximum_length），mysql "binary"
	// 默认binary(1)会直接截断数据（Data too long），降级为blob
	if col.CharMaxLength <= 0 {
		return Blob
	}
	return Binary
}
func (c *commonTypeConverter) Varbinary(col *dbi.Column) *dbi.DbDataType {
	return Varbinary
}
func (c *commonTypeConverter) Mediumblob(col *dbi.Column) *dbi.DbDataType {
	return Mediumblob
}
func (c *commonTypeConverter) Blob(col *dbi.Column) *dbi.DbDataType {
	return Blob
}
func (c *commonTypeConverter) Longblob(col *dbi.Column) *dbi.DbDataType {
	col.CharMaxLength = 0
	return Longblob
}

func (c *commonTypeConverter) Enum(col *dbi.Column) *dbi.DbDataType {
	return Enum
}
func (c *commonTypeConverter) JSON(col *dbi.Column) *dbi.DbDataType {
	return JSON
}
