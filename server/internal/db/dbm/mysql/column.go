package mysql

import (
	"mayfly-go/internal/db/dbm/dbi"
)

const (
	IndexSubPartKey = "subPart"
)

var (
	// DTStringMysql mysql专用字符串类型：mysql默认模式下反斜杠是转义字符，
	// 若仅转义单引号而不双写反斜杠，含\b、\n、\'等内容的字符串会被静默解释为转义字符导致数据损坏
	DTStringMysql = dbi.DTString.Copy().WithSQLValue(dbi.SQLValueStringEscapeBackslash)

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

	Varchar    = dbi.NewDbDataType("varchar", DTStringMysql).WithCT(dbi.CTVarchar)
	Char       = dbi.NewDbDataType("char", DTStringMysql).WithCT(dbi.CTChar)
	Text       = dbi.NewDbDataType("text", DTStringMysql).WithCT(dbi.CTText).WithFixColumn(dbi.ClearCharMaxLength)
	Mediumtext = dbi.NewDbDataType("mediumtext", DTStringMysql).WithCT(dbi.CTMediumtext).WithFixColumn(dbi.ClearCharMaxLength)
	Longtext   = dbi.NewDbDataType("longtext", DTStringMysql).WithCT(dbi.CTLongtext).WithFixColumn(dbi.ClearCharMaxLength)
	JSON       = dbi.NewDbDataType("json", DTStringMysql).WithCT(dbi.CTJSON).WithFixColumn(dbi.ClearCharMaxLength)

	Datetime  = dbi.NewDbDataType("datetime", dbi.DTDateTime).WithCT(dbi.CTDateTime)
	Date      = dbi.NewDbDataType("date", dbi.DTDate).WithCT(dbi.CTDate)
	Time      = dbi.NewDbDataType("time", dbi.DTTime).WithCT(dbi.CTTime)
	Timestamp = dbi.NewDbDataType("timestamp", dbi.DTDateTime).WithCT(dbi.CTTimestamp)

	Enum = dbi.NewDbDataType("enum", DTStringMysql).WithCT(dbi.CTEnum)
	Set  = dbi.NewDbDataType("set", DTStringMysql).WithCT(dbi.CTVarchar)

	Blob       = dbi.NewDbDataType("blob", dbi.DTBytes).WithCT(dbi.CTBlob).WithFixColumn(dbi.ClearNumScale)
	Mediumblob = dbi.NewDbDataType("mediumblob", dbi.DTBytes).WithCT(dbi.CTMediumblob).WithFixColumn(dbi.ClearNumScale)
	Longblob   = dbi.NewDbDataType("longblob", dbi.DTBytes).WithCT(dbi.CTLongblob).WithFixColumn(dbi.ClearNumScale)
	Binary     = dbi.NewDbDataType("binary", dbi.DTBytes).WithCT(dbi.CTBinary)
	Varbinary  = dbi.NewDbDataType("varbinary", dbi.DTBytes).WithCT(dbi.CTVarbinary)
)
