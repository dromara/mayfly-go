package mssql

import "mayfly-go/internal/db/dbm/dbi"

var _ dbi.CommonTypeConverter = (*commonTypeConverter)(nil)

type commonTypeConverter struct {
}

func (c *commonTypeConverter) Varchar(col *dbi.Column) *dbi.DbDataType {
	// varchar(n)上限8000字节，超长转varchar(max)避免非法DDL
	if col.CharMaxLength > 8000 {
		col.CharMaxLength = 0
		return VarcharMax
	}
	return Varchar
}

func (c *commonTypeConverter) Char(col *dbi.Column) *dbi.DbDataType {
	return Char
}
func (c *commonTypeConverter) Text(col *dbi.Column) *dbi.DbDataType {
	// text类型无长度语法，清空长度避免生成text(n)非法DDL
	col.CharMaxLength = 0
	return Text
}
func (c *commonTypeConverter) Mediumtext(col *dbi.Column) *dbi.DbDataType {
	// 与Text对齐：text无长度语法，清空源长度避免生成text(n)非法DDL
	col.CharMaxLength = 0
	return Text
}
func (c *commonTypeConverter) Longtext(col *dbi.Column) *dbi.DbDataType {
	col.CharMaxLength = 0
	return Text
}

func (c *commonTypeConverter) Bit(col *dbi.Column) *dbi.DbDataType {
	return Bit
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
	return Numeric
}

func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
	return Decimal
}

func (c *commonTypeConverter) UnsignedInt8(col *dbi.Column) *dbi.DbDataType {
	return Bigint
}
func (c *commonTypeConverter) UnsignedInt4(col *dbi.Column) *dbi.DbDataType {
	return Int
}
func (c *commonTypeConverter) UnsignedInt2(col *dbi.Column) *dbi.DbDataType {
	return Smallint
}
func (c *commonTypeConverter) UnsignedInt1(col *dbi.Column) *dbi.DbDataType {
	return Tinyint
}

func (c *commonTypeConverter) Date(col *dbi.Column) *dbi.DbDataType {
	return Date
}
func (c *commonTypeConverter) Time(col *dbi.Column) *dbi.DbDataType {
	return Time
}
func (c *commonTypeConverter) Datetime(col *dbi.Column) *dbi.DbDataType {
	return Datetime
}
func (c *commonTypeConverter) Timestamp(col *dbi.Column) *dbi.DbDataType {
	return Datetime
}

func (c *commonTypeConverter) Binary(col *dbi.Column) *dbi.DbDataType {
	return bytesType(col)
}
func (c *commonTypeConverter) Varbinary(col *dbi.Column) *dbi.DbDataType {
	return bytesType(col)
}
func (c *commonTypeConverter) Mediumblob(col *dbi.Column) *dbi.DbDataType {
	return bytesType(col)
}
func (c *commonTypeConverter) Blob(col *dbi.Column) *dbi.DbDataType {
	return bytesType(col)
}
func (c *commonTypeConverter) Longblob(col *dbi.Column) *dbi.DbDataType {
	return bytesType(col)
}

// bytesType 二进制类型统一归一化：varbinary(n)上限8000且binary为定长填充（会pad 0x00），
// 源无长度或超长时用varbinary(max)承载，避免binary(1)默认长度或非法DDL
func bytesType(col *dbi.Column) *dbi.DbDataType {
	if col.CharMaxLength <= 0 || col.CharMaxLength > 8000 {
		col.CharMaxLength = 0
		return VarbinaryMax
	}
	return Varbinary
}

func (c *commonTypeConverter) Enum(col *dbi.Column) *dbi.DbDataType {
	return Varchar
}
func (c *commonTypeConverter) JSON(col *dbi.Column) *dbi.DbDataType {
	return Text
}
