package postgres

import "mayfly-go/internal/db/dbm/dbi"

var _ dbi.CommonTypeConverter = (*commonTypeConverter)(nil)

type commonTypeConverter struct {
}

func (c *commonTypeConverter) Varchar(col *dbi.Column) *dbi.DbDataType {
	return Varchar
}

func (c *commonTypeConverter) Char(col *dbi.Column) *dbi.DbDataType {
	return Char
}
func (c *commonTypeConverter) Text(col *dbi.Column) *dbi.DbDataType {
	return Text
}
func (c *commonTypeConverter) Mediumtext(col *dbi.Column) *dbi.DbDataType {
	return Text
}
func (c *commonTypeConverter) Longtext(col *dbi.Column) *dbi.DbDataType {
	return Text
}

func (c *commonTypeConverter) Bit(col *dbi.Column) *dbi.DbDataType {
	return Int2
}
func (c *commonTypeConverter) Int1(col *dbi.Column) *dbi.DbDataType {
	return Int2
}
func (c *commonTypeConverter) Int2(col *dbi.Column) *dbi.DbDataType {
	return Int2
}
func (c *commonTypeConverter) Int4(col *dbi.Column) *dbi.DbDataType {
	return Int4
}
func (c *commonTypeConverter) Int8(col *dbi.Column) *dbi.DbDataType {
	return Int8
}
func (c *commonTypeConverter) Numeric(col *dbi.Column) *dbi.DbDataType {
	return Numeric
}

func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
	return Decimal
}

func (c *commonTypeConverter) UnsignedInt8(col *dbi.Column) *dbi.DbDataType {
	return Int8
}
func (c *commonTypeConverter) UnsignedInt4(col *dbi.Column) *dbi.DbDataType {
	return Int4
}
func (c *commonTypeConverter) UnsignedInt2(col *dbi.Column) *dbi.DbDataType {
	return Int2
}
func (c *commonTypeConverter) UnsignedInt1(col *dbi.Column) *dbi.DbDataType {
	return Int2
}

func (c *commonTypeConverter) Date(col *dbi.Column) *dbi.DbDataType {
	return Date
}
func (c *commonTypeConverter) Time(col *dbi.Column) *dbi.DbDataType {
	return Time
}
func (c *commonTypeConverter) Datetime(col *dbi.Column) *dbi.DbDataType {
	return Timestamp
}
func (c *commonTypeConverter) Timestamp(col *dbi.Column) *dbi.DbDataType {
	return Timestamp
}

func (c *commonTypeConverter) Binary(col *dbi.Column) *dbi.DbDataType {
	return Bytea
}
func (c *commonTypeConverter) Varbinary(col *dbi.Column) *dbi.DbDataType {
	return Bytea
}
func (c *commonTypeConverter) Mediumblob(col *dbi.Column) *dbi.DbDataType {
	return Bytea
}
func (c *commonTypeConverter) Blob(col *dbi.Column) *dbi.DbDataType {
	return Bytea
}
func (c *commonTypeConverter) Longblob(col *dbi.Column) *dbi.DbDataType {
	return Bytea
}

func (c *commonTypeConverter) Enum(col *dbi.Column) *dbi.DbDataType {
	return Varchar
}
func (c *commonTypeConverter) JSON(col *dbi.Column) *dbi.DbDataType {
	return Json
}
