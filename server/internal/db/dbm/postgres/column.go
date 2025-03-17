package postgres

import (
	"mayfly-go/internal/db/dbm/dbi"
)

var (
	Bool        = dbi.NewDbDataType("bool", dbi.DTBool).WithCT(dbi.CTBool).WithFixColumn(dbi.ClearNumScale)
	Int2        = dbi.NewDbDataType("int2", dbi.DTInt16).WithCT(dbi.CTInt2).WithFixColumn(dbi.ClearNumScale)
	Int4        = dbi.NewDbDataType("int4", dbi.DTInt32).WithCT(dbi.CTInt4).WithFixColumn(dbi.ClearNumScale)
	Int8        = dbi.NewDbDataType("int8", dbi.DTInt64).WithCT(dbi.CTInt8).WithFixColumn(dbi.ClearNumScale)
	Numeric     = dbi.NewDbDataType("numeric", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	Decimal     = dbi.NewDbDataType("decimal", dbi.DTDecimal).WithCT(dbi.CTDecimal)
	Smallserial = dbi.NewDbDataType("smallserial", dbi.DTInt16).WithCT(dbi.CTInt2)
	Serial      = dbi.NewDbDataType("serial", dbi.DTInt32).WithCT(dbi.CTInt4)
	Bigserial   = dbi.NewDbDataType("bigserial", dbi.DTInt64).WithCT(dbi.CTInt8)
	Largeserial = dbi.NewDbDataType("largeserial", dbi.DTInt64).WithCT(dbi.CTInt8)

	Money = dbi.NewDbDataType("money", dbi.DTString).WithCT(dbi.CTVarchar)

	Char    = dbi.NewDbDataType("char", dbi.DTString).WithCT(dbi.CTChar)
	Nchar   = dbi.NewDbDataType("nchar", dbi.DTString).WithCT(dbi.CTVarchar)
	Varchar = dbi.NewDbDataType("varchar", dbi.DTString).WithCT(dbi.CTVarchar)
	Text    = dbi.NewDbDataType("text", dbi.DTString).WithCT(dbi.CTText).WithFixColumn(dbi.ClearCharMaxLength)
	Json    = dbi.NewDbDataType("json", dbi.DTString).WithCT(dbi.CTJSON).WithFixColumn(dbi.ClearCharMaxLength)
	Bytea   = dbi.NewDbDataType("bytea", dbi.DTString).WithCT(dbi.CTBinary)

	Date      = dbi.NewDbDataType("date", dbi.DTDate).WithCT(dbi.CTDate).WithFixColumn(dbi.ClearCharMaxLength)
	Time      = dbi.NewDbDataType("time", dbi.DTTime).WithCT(dbi.CTTime).WithFixColumn(dbi.ClearCharMaxLength)
	Timestamp = dbi.NewDbDataType("timestamp", dbi.DTDateTime).WithCT(dbi.CTDateTime).WithFixColumn(dbi.ClearCharMaxLength)
)
