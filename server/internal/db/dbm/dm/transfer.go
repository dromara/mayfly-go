package dm

import "mayfly-go/internal/db/dbm/dbi"

var _ dbi.CommonTypeConverter = (*commonTypeConverter)(nil)

type commonTypeConverter struct {
}

func (c *commonTypeConverter) Varchar(col *dbi.Column) *dbi.DbDataType {
	return VARCHAR
}

func (c *commonTypeConverter) Char(col *dbi.Column) *dbi.DbDataType {
	return CHAR
}
func (c *commonTypeConverter) Text(col *dbi.Column) *dbi.DbDataType {
	return TEXT
}
func (c *commonTypeConverter) Mediumtext(col *dbi.Column) *dbi.DbDataType {
	return TEXT
}
func (c *commonTypeConverter) Longtext(col *dbi.Column) *dbi.DbDataType {
	return LONGVARCHAR
}

func (c *commonTypeConverter) Bit(col *dbi.Column) *dbi.DbDataType {
	clearLength(col)
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
func (c *commonTypeConverter) Numeric(col *dbi.Column) *dbi.DbDataType {
	return NUMBER
}

func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
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
