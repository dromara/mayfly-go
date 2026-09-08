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
func (c *commonTypeConverter) Bool(col *dbi.Column) *dbi.DbDataType {
	return Bool
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

// Numeric pg的numeric不声明精度即代表任意精度，因此源列无精度约束时必须保持不带括号，
// 而不能沿用其他库的默认精度（会反而限制目标列）
func (c *commonTypeConverter) Numeric(col *dbi.Column) *dbi.DbDataType {
	dbi.ClearNumPrecision(col)
	return Numeric
}

// Decimal 同理：无精度时pg的numeric比任何固定精度都能无损承载源值；带精度时收敛到pg上限
func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
	if col.NumPrecision <= 0 {
		dbi.ClearNumPrecision(col)
	} else {
		dbi.ClampDecimalPrecision(col, 1000, 1000)
	}
	return Decimal
}

func (c *commonTypeConverter) UnsignedInt8(col *dbi.Column) *dbi.DbDataType {
	// uint64最大值(18446744073709551615)超过pg int8上限(9223372036854775807)，转numeric避免溢出截断
	return Numeric
}
func (c *commonTypeConverter) UnsignedInt4(col *dbi.Column) *dbi.DbDataType {
	// uint32最大值(4294967295)超过pg int4上限(2147483647)，需升位为int8
	return Int8
}
func (c *commonTypeConverter) UnsignedInt2(col *dbi.Column) *dbi.DbDataType {
	// uint16最大值(65535)超过pg int2上限(32767)，需升位为int4
	return Int4
}
func (c *commonTypeConverter) UnsignedInt1(col *dbi.Column) *dbi.DbDataType {
	return Int2
}

func (c *commonTypeConverter) Date(col *dbi.Column) *dbi.DbDataType {
	// pg的date不接受任何括号参数，源列残留精度会生成 date(3) 非法DDL
	dbi.ClearNumPrecision(col)
	return Date
}
func (c *commonTypeConverter) Time(col *dbi.Column) *dbi.DbDataType {
	// pg的time/timestamp不声明fsp时即最大精度(6)，故只需收敛超限精度，不能主动补齐
	dbi.ClampTimeFsp(col, 6)
	return Time
}
func (c *commonTypeConverter) Datetime(col *dbi.Column) *dbi.DbDataType {
	dbi.ClampTimeFsp(col, 6)
	return Timestamp
}
func (c *commonTypeConverter) Timestamp(col *dbi.Column) *dbi.DbDataType {
	dbi.ClampTimeFsp(col, 6)
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
