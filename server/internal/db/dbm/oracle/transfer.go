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
func (c *commonTypeConverter) Numeric(col *dbi.Column) *dbi.DbDataType {
	return NUMBER
}

func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
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
	return DATE
}
func (c *commonTypeConverter) Time(col *dbi.Column) *dbi.DbDataType {
	return TIME
}
func (c *commonTypeConverter) Datetime(col *dbi.Column) *dbi.DbDataType {
	return TIMESTAMP
}
func (c *commonTypeConverter) Timestamp(col *dbi.Column) *dbi.DbDataType {
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
