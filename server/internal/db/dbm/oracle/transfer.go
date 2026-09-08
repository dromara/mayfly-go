package oracle

import "mayfly-go/internal/db/dbm/dbi"

var _ dbi.CommonTypeConverter = (*commonTypeConverter)(nil)

type commonTypeConverter struct {
}

func (c *commonTypeConverter) Varchar(col *dbi.Column) *dbi.DbDataType {
	// VARCHAR2(n)上限4000字节（extended默认关闭），超长转CLOB承载，避免非法DDL
	if col.CharMaxLength > 4000 {
		col.CharMaxLength = 0
		return CLOB
	}
	return VARCHAR2
}

func (c *commonTypeConverter) Char(col *dbi.Column) *dbi.DbDataType {
	return CHAR
}
func (c *commonTypeConverter) Text(col *dbi.Column) *dbi.DbDataType {
	// NVARCHAR2无长度时默认长度为1，长文本必插入失败，改用无长度约束的CLOB
	return lobType(col)
}
func (c *commonTypeConverter) Mediumtext(col *dbi.Column) *dbi.DbDataType {
	return lobType(col)
}
func (c *commonTypeConverter) Longtext(col *dbi.Column) *dbi.DbDataType {
	return lobType(col)
}

// lobType 大文本统一归一化：CLOB无长度概念，需清空源长度避免生成CLOB(n)非法DDL
func lobType(col *dbi.Column) *dbi.DbDataType {
	col.CharMaxLength = 0
	return CLOB
}

func (c *commonTypeConverter) Bit(col *dbi.Column) *dbi.DbDataType {
	return BIT
}
func (c *commonTypeConverter) Bool(col *dbi.Column) *dbi.DbDataType {
	return BIT
}
func (c *commonTypeConverter) Int1(col *dbi.Column) *dbi.DbDataType {
	return TINYINT
}
func (c *commonTypeConverter) Int2(col *dbi.Column) *dbi.DbDataType {
	return SMALLINT
}
func (c *commonTypeConverter) Int4(col *dbi.Column) *dbi.DbDataType {
	return INTEGER
}
func (c *commonTypeConverter) Int8(col *dbi.Column) *dbi.DbDataType {
	return BIGINT
}

// Numeric Oracle的NUMBER不带精度即无约束精确数值，可无损承载任意浮点/数值源列；
// 但带精度时NUMBER(p)相当于整数语义（标度0），沿用浮点源的精度会静默截断小数，故必须清空
func (c *commonTypeConverter) Numeric(col *dbi.Column) *dbi.DbDataType {
	dbi.ClearNumPrecision(col)
	return NUMBER
}

// Decimal NUMBER精度上限为38位，异构源（如mysql decimal(65,30)、pg numeric(200,30)）超限必须收敛，
// 否则生成非法DDL；源列为无精度约束的精确数值（pg的numeric、oracle自己的NUMBER、sqlite的numeric）时，
// 必须落为不带参数的NUMBER（Oracle语义下即任意精度），若沿用不带参数的DECIMAL会被当作NUMBER(38,0)
// 而静默截断所有小数
func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
	if col.NumPrecision <= 0 {
		dbi.ClearNumPrecision(col)
		return NUMBER
	}
	dbi.ClampDecimalPrecision(col, 38, 38)
	return DECIMAL
}

func (c *commonTypeConverter) UnsignedInt8(col *dbi.Column) *dbi.DbDataType {
	return BIGINT
}
func (c *commonTypeConverter) UnsignedInt4(col *dbi.Column) *dbi.DbDataType {
	return INT
}
func (c *commonTypeConverter) UnsignedInt2(col *dbi.Column) *dbi.DbDataType {
	return INT
}
func (c *commonTypeConverter) UnsignedInt1(col *dbi.Column) *dbi.DbDataType {
	return INT
}

func (c *commonTypeConverter) Date(col *dbi.Column) *dbi.DbDataType {
	// Oracle的DATE不接受精度参数且仅秒精度，源列残留精度会生成 DATE(3) 非法DDL
	dbi.ClearNumPrecision(col)
	return DATE
}
func (c *commonTypeConverter) Time(col *dbi.Column) *dbi.DbDataType {
	dbi.ClearNumPrecision(col)
	return TIME
}
func (c *commonTypeConverter) Datetime(col *dbi.Column) *dbi.DbDataType {
	// TIMESTAMP(n)合法且上限为9位，只需收敛超限精度；无精度时保留不带括号的TIMESTAMP（默认6位）
	dbi.ClampTimeFsp(col, 9)
	return TIMESTAMP
}
func (c *commonTypeConverter) Timestamp(col *dbi.Column) *dbi.DbDataType {
	dbi.ClampTimeFsp(col, 9)
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
	return NVARCHAR2
}
func (c *commonTypeConverter) JSON(col *dbi.Column) *dbi.DbDataType {
	return NVARCHAR2
}
