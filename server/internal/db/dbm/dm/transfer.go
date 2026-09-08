package dm

import "mayfly-go/internal/db/dbm/dbi"

var _ dbi.CommonTypeConverter = (*commonTypeConverter)(nil)

type commonTypeConverter struct {
}

func (c *commonTypeConverter) Varchar(col *dbi.Column) *dbi.DbDataType {
	// dm VARCHAR上限32767，超长转TEXT承载，避免非法DDL
	if col.CharMaxLength > 32767 {
		col.CharMaxLength = 0
		return TEXT
	}
	return VARCHAR
}

func (c *commonTypeConverter) Char(col *dbi.Column) *dbi.DbDataType {
	return CHAR
}
func (c *commonTypeConverter) Text(col *dbi.Column) *dbi.DbDataType {
	// text无长度语法，清空源长度避免生成text(n)非法DDL
	col.CharMaxLength = 0
	return TEXT
}
func (c *commonTypeConverter) Mediumtext(col *dbi.Column) *dbi.DbDataType {
	col.CharMaxLength = 0
	return TEXT
}
func (c *commonTypeConverter) Longtext(col *dbi.Column) *dbi.DbDataType {
	col.CharMaxLength = 0
	return LONGVARCHAR
}

func (c *commonTypeConverter) Bit(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return BIT
}
func (c *commonTypeConverter) Bool(col *dbi.Column) *dbi.DbDataType {
	return BIT
}
func (c *commonTypeConverter) Int1(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return TINYINT
}
func (c *commonTypeConverter) Int2(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return SMALLINT
}
func (c *commonTypeConverter) Int4(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return INTEGER
}
func (c *commonTypeConverter) Int8(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return BIGINT
}

// Numeric 达梦的NUMBER不声明精度即无约束精确数值，可无损承载任意浮点/数值源列；
// 带精度时会被当作整数语义而静默截断浮点小数，故必须清空源精度
func (c *commonTypeConverter) Numeric(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	col.NumScale = 0
	return NUMBER
}

// Decimal NUMBER精度上限为38位，异构源超限精度必须收敛，否则生成非法DDL；
// 达梦的DECIMAL/NUMBER省略精度即等价NUMBER(38,0)，无约束的源精确数值必须显式补齐精度与小数位，
// 否则目标表的所有小数会被静默截断（宁可插入时报范围错误，也不静默丢小数）
func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
	dbi.FillUnboundedDecimal(col, 38, 19)
	dbi.ClampDecimalPrecision(col, 38, 38)
	return DECIMAL
}

func (c *commonTypeConverter) UnsignedInt8(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return BIGINT
}
func (c *commonTypeConverter) UnsignedInt4(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return INT
}
func (c *commonTypeConverter) UnsignedInt2(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return INT
}
func (c *commonTypeConverter) UnsignedInt1(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return INT
}

func (c *commonTypeConverter) Date(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return DATE
}
func (c *commonTypeConverter) Time(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return TIME
}
func (c *commonTypeConverter) Datetime(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return DATETIME
}
func (c *commonTypeConverter) Timestamp(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
	return TIMESTAMP
}

func (c *commonTypeConverter) Binary(col *dbi.Column) *dbi.DbDataType {
	return BLOB
}
func (c *commonTypeConverter) Varbinary(col *dbi.Column) *dbi.DbDataType {
	return BLOB
}
func (c *commonTypeConverter) Mediumblob(col *dbi.Column) *dbi.DbDataType {
	return BLOB
}
func (c *commonTypeConverter) Blob(col *dbi.Column) *dbi.DbDataType {
	return BLOB
}
func (c *commonTypeConverter) Longblob(col *dbi.Column) *dbi.DbDataType {
	return BLOB
}

func (c *commonTypeConverter) Enum(col *dbi.Column) *dbi.DbDataType {
	return VARCHAR
}
func (c *commonTypeConverter) JSON(col *dbi.Column) *dbi.DbDataType {
	return VARCHAR
}

func clearLength(col *dbi.Column) {
	col.CharMaxLength = 0
	col.NumPrecision = 0
}
