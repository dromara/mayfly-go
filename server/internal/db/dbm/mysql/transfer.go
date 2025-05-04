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
	return Double
}

func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
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
	return Date
}
func (c *commonTypeConverter) Time(col *dbi.Column) *dbi.DbDataType {
	return Time
}
func (c *commonTypeConverter) Datetime(col *dbi.Column) *dbi.DbDataType {
	return Datetime
}
func (c *commonTypeConverter) Timestamp(col *dbi.Column) *dbi.DbDataType {
	return Timestamp
}

func (c *commonTypeConverter) Binary(col *dbi.Column) *dbi.DbDataType {
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
