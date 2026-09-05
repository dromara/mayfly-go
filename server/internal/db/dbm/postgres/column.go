package postgres

import (
	"fmt"

	"mayfly-go/internal/db/dbm/dbi"
)

// pgSQLValueBytes 二进制值转SQL：hex编码输出postgres bytea标准hex格式'\x...'保真还原，
// 非hex值保留特殊字符的字符串字面量处理
func pgSQLValueBytes(val any) string {
	if val == nil {
		return dbi.NULL
	}
	if strVal, ok := val.(string); ok && dbi.IsHexString(strVal) {
		return fmt.Sprintf("'\\x%s'", strVal)
	}
	return dbi.SQLValuePreserveSpecialChars(val)
}

var (
	// DTBytesPg postgres专用二进制类型：hex值以bytea标准hex格式保真还原
	DTBytesPg = dbi.DTBytes.Copy().WithSQLValue(pgSQLValueBytes)

	Bool        = dbi.NewDbDataType("bool", dbi.DTString).WithCT(dbi.CTBool).WithFixColumn(dbi.ClearNumScale)
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

	Char    = dbi.NewDbDataType("char", dbi.DTStringPreserveSpecial).WithCT(dbi.CTChar)
	Nchar   = dbi.NewDbDataType("nchar", dbi.DTStringPreserveSpecial).WithCT(dbi.CTVarchar)
	Varchar = dbi.NewDbDataType("varchar", dbi.DTStringPreserveSpecial).WithCT(dbi.CTVarchar)
	Text    = dbi.NewDbDataType("text", dbi.DTStringPreserveSpecial).WithCT(dbi.CTText).WithFixColumn(dbi.ClearCharMaxLength)
	Json    = dbi.NewDbDataType("json", dbi.DTStringPreserveSpecial).WithCT(dbi.CTJSON).WithFixColumn(dbi.ClearCharMaxLength)
	Jsonb   = dbi.NewDbDataType("jsonb", dbi.DTStringPreserveSpecial).WithCT(dbi.CTJSON).WithFixColumn(dbi.ClearCharMaxLength)
	Bytea   = dbi.NewDbDataType("bytea", DTBytesPg).WithCT(dbi.CTBinary)

	Date      = dbi.NewDbDataType("date", dbi.DTDate).WithCT(dbi.CTDate).WithFixColumn(dbi.ClearCharMaxLength)
	Time      = dbi.NewDbDataType("time", dbi.DTTime).WithCT(dbi.CTTime).WithFixColumn(dbi.ClearCharMaxLength)
	Timestamp = dbi.NewDbDataType("timestamp", dbi.DTDateTime).WithCT(dbi.CTDateTime).WithFixColumn(dbi.ClearCharMaxLength)
)
