package sqlite

import "mayfly-go/internal/db/dbm/dbi"

var _ dbi.CommonTypeConverter = (*commonTypeConverter)(nil)

type commonTypeConverter struct {
}

func (c *commonTypeConverter) Varchar(col *dbi.Column) *dbi.DbDataType {
	return Text
}

func (c *commonTypeConverter) Char(col *dbi.Column) *dbi.DbDataType {
	return Text
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
	return Integer
}
func (c *commonTypeConverter) Bool(col *dbi.Column) *dbi.DbDataType {
	return Integer
}
func (c *commonTypeConverter) Int1(col *dbi.Column) *dbi.DbDataType {
	return Integer
}
func (c *commonTypeConverter) Int2(col *dbi.Column) *dbi.DbDataType {
	return Integer
}
func (c *commonTypeConverter) Int4(col *dbi.Column) *dbi.DbDataType {
	return Integer
}
func (c *commonTypeConverter) Int8(col *dbi.Column) *dbi.DbDataType {
	return Integer
}

// Numeric/Decimal 必须落到 NUMERIC 亲和的 numeric/decimal 声明，不能降级为 real：
// REAL亲和会强制把定点数以IEEE-754浮点存储（金额类值失去十进制精确性，整数值还会多出.0），
// 且再往强类型库迁移时列类型会从decimal一路退化成double/float；SQLite官方亲和规则正是把
// NUMERIC/DECIMAL作为定点数的推荐声明（整数值还能以INTEGER存储）。两者同为字符串Valuer，
// 读写行为不变，仅修正DDL的声明类型
func (c *commonTypeConverter) Numeric(col *dbi.Column) *dbi.DbDataType {
	return Numeric
}

func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
	return Decimal
}

func (c *commonTypeConverter) UnsignedInt8(col *dbi.Column) *dbi.DbDataType {
	return Integer
}
func (c *commonTypeConverter) UnsignedInt4(col *dbi.Column) *dbi.DbDataType {
	return Integer
}
func (c *commonTypeConverter) UnsignedInt2(col *dbi.Column) *dbi.DbDataType {
	return Integer
}
func (c *commonTypeConverter) UnsignedInt1(col *dbi.Column) *dbi.DbDataType {
	return Integer
}

// SQLite是动态类型库，日期时间仅以声明类型决定亲和性，DDL不支持任何精度参数，
// 故必须抹掉源列残留的精度/小数位，避免生成 datetime(6) 这类无意义且不合法的列声明
func (c *commonTypeConverter) Date(col *dbi.Column) *dbi.DbDataType {
	dbi.ClearNumPrecision(col)
	return Date
}
func (c *commonTypeConverter) Time(col *dbi.Column) *dbi.DbDataType {
	dbi.ClearNumPrecision(col)
	return Time
}
func (c *commonTypeConverter) Datetime(col *dbi.Column) *dbi.DbDataType {
	dbi.ClearNumPrecision(col)
	return DateTime
}
func (c *commonTypeConverter) Timestamp(col *dbi.Column) *dbi.DbDataType {
	dbi.ClearNumPrecision(col)
	return DateTime
}

func (c *commonTypeConverter) Binary(col *dbi.Column) *dbi.DbDataType {
	return Blob
}
func (c *commonTypeConverter) Varbinary(col *dbi.Column) *dbi.DbDataType {
	return Blob
}
func (c *commonTypeConverter) Mediumblob(col *dbi.Column) *dbi.DbDataType {
	return Blob
}
func (c *commonTypeConverter) Blob(col *dbi.Column) *dbi.DbDataType {
	return Blob
}
func (c *commonTypeConverter) Longblob(col *dbi.Column) *dbi.DbDataType {
	return Blob
}

func (c *commonTypeConverter) Enum(col *dbi.Column) *dbi.DbDataType {
	return Text
}
func (c *commonTypeConverter) JSON(col *dbi.Column) *dbi.DbDataType {
	return Text
}
