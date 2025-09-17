package mysql

import (
	"mayfly-go/internal/db/dbm/dbi"
)

const (
	IndexSubPartKey = "subPart"
)

var (
	Bit       = dbi.NewDbDataType("bit", dbi.DTBit).WithCT(dbi.CTBit)
	Tinyint   = dbi.NewDbDataType("tinyint", dbi.DTInt8).WithCT(dbi.CTInt1).WithFixColumn(dbi.ClearNumScale)
	Smallint  = dbi.NewDbDataType("smallint", dbi.DTInt16).WithCT(dbi.CTInt2).WithFixColumn(dbi.ClearNumScale)
	Mediumint = dbi.NewDbDataType("mediumint", dbi.DTInt32).WithCT(dbi.CTInt4).WithFixColumn(dbi.ClearNumScale)
	Int       = dbi.NewDbDataType("int", dbi.DTInt32).WithCT(dbi.CTInt4).WithFixColumn(dbi.ClearNumScale)
	Bigint    = dbi.NewDbDataType("bigint", dbi.DTInt64).WithCT(dbi.CTInt8).WithFixColumn(dbi.ClearNumScale)

	UnsignedBigint    = dbi.NewDbDataType("unsigned bigint", dbi.DTUint64).WithCT(dbi.CTUnsignedInt8).WithFixColumn(dbi.ClearNumScale)
	UnsignedInt       = dbi.NewDbDataType("unsigned int", dbi.DTUint64).WithCT(dbi.CTUnsignedInt4).WithFixColumn(dbi.ClearNumScale)
	UnsignedSmallint  = dbi.NewDbDataType("unsigned smallint", dbi.DTInt32).WithCT(dbi.CTUnsignedInt2).WithFixColumn(dbi.ClearNumScale)
	UnsignedMediumint = dbi.NewDbDataType("unsigned mediumint", dbi.DTInt64).WithCT(dbi.CTUnsignedInt4).WithFixColumn(dbi.ClearNumScale)

	Decimal = dbi.NewDbDataType("decimal", dbi.DTDecimal).WithCT(dbi.CTDecimal)
	Double  = dbi.NewDbDataType("double", dbi.DTNumeric).WithCT(dbi.CTNumeric).WithFixColumn(dbi.ClearNumPrecision)
	Float   = dbi.NewDbDataType("float", dbi.DTNumeric).WithCT(dbi.CTNumeric)

	Varchar    = dbi.NewDbDataType("varchar", dbi.DTString).WithCT(dbi.CTVarchar)
	Char       = dbi.NewDbDataType("char", dbi.DTString).WithCT(dbi.CTChar)
	Text       = dbi.NewDbDataType("text", dbi.DTString).WithCT(dbi.CTText).WithFixColumn(dbi.ClearCharMaxLength)
	Mediumtext = dbi.NewDbDataType("mediumtext", dbi.DTString).WithCT(dbi.CTMediumtext).WithFixColumn(dbi.ClearCharMaxLength)
	Longtext   = dbi.NewDbDataType("longtext", dbi.DTString).WithCT(dbi.CTLongtext).WithFixColumn(dbi.ClearCharMaxLength)
	JSON       = dbi.NewDbDataType("json", dbi.DTString).WithCT(dbi.CTJSON).WithFixColumn(dbi.ClearCharMaxLength)

	Datetime  = dbi.NewDbDataType("datetime", dbi.DTDateTime).WithCT(dbi.CTDateTime)
	Date      = dbi.NewDbDataType("date", dbi.DTDate).WithCT(dbi.CTDate)
	Time      = dbi.NewDbDataType("time", dbi.DTTime).WithCT(dbi.CTTime)
	Timestamp = dbi.NewDbDataType("timestamp", dbi.DTDateTime).WithCT(dbi.CTTimestamp)

	Enum = dbi.NewDbDataType("enum", dbi.DTString).WithCT(dbi.CTEnum)
	Set  = dbi.NewDbDataType("set", dbi.DTString).WithCT(dbi.CTVarchar)

	Blob       = dbi.NewDbDataType("blob", dbi.DTBytes).WithCT(dbi.CTBlob).WithFixColumn(dbi.ClearNumScale)
	Mediumblob = dbi.NewDbDataType("mediumblob", dbi.DTBytes).WithCT(dbi.CTMediumblob).WithFixColumn(dbi.ClearNumScale)
	Longblob   = dbi.NewDbDataType("longblob", dbi.DTBytes).WithCT(dbi.CTLongblob).WithFixColumn(dbi.ClearNumScale)
	Binary     = dbi.NewDbDataType("binary", dbi.DTBytes).WithCT(dbi.CTBinary)
	Varbinary  = dbi.NewDbDataType("varbinary", dbi.DTBytes).WithCT(dbi.CTVarbinary)
)
