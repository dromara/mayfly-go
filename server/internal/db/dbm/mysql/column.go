package mysql

import (
	"fmt"

	"mayfly-go/internal/db/dbm/dbi"
)

const (
	IndexSubPartKey = "subPart"
)

// mysqlSQLValueBytes 二进制值转SQL：hex编码输出X'...'保真还原，非hex按mysql转义规则处理
func mysqlSQLValueBytes(val any) string {
	if val == nil {
		return dbi.NULL
	}
	if strVal, ok := val.(string); ok && dbi.IsHexString(strVal) {
		return fmt.Sprintf("X'%s'", strVal)
	}
	return dbi.SQLValueStringEscapeBackslash(val)
}

var (
	// DTStringMysql mysql专用字符串类型：mysql默认模式下反斜杠是转义字符，
	// 若仅转义单引号而不双写反斜杠，含\b、\n、\'等内容的字符串会被静默解释为转义字符导致数据损坏
	DTStringMysql = dbi.DTString.Copy().WithSQLValue(dbi.SQLValueStringEscapeBackslash)

	// DTBytesMysql mysql专用二进制类型：迁移链路中二进制数据经ValuerBytes回读为hex编码字符串，
	// 输出X'...'保真还原二进制；非hex值（调用方直接构造的字符串）按mysql转义规则处理
	DTBytesMysql = dbi.DTBytes.Copy().WithSQLValue(mysqlSQLValueBytes)

	Bit       = dbi.NewDbDataType("bit", dbi.DTBit).WithCT(dbi.CTBit)
	Tinyint   = dbi.NewDbDataType("tinyint", dbi.DTInt8).WithCT(dbi.CTInt1).WithFixColumn(dbi.ClearNumScale)
	Smallint  = dbi.NewDbDataType("smallint", dbi.DTInt16).WithCT(dbi.CTInt2).WithFixColumn(dbi.ClearNumScale)
	Mediumint = dbi.NewDbDataType("mediumint", dbi.DTInt32).WithCT(dbi.CTInt4).WithFixColumn(dbi.ClearNumScale)
	Int       = dbi.NewDbDataType("int", dbi.DTInt32).WithCT(dbi.CTInt4).WithFixColumn(dbi.ClearNumScale)
	Bigint    = dbi.NewDbDataType("bigint", dbi.DTInt64).WithCT(dbi.CTInt8).WithFixColumn(dbi.ClearNumScale)

	UnsignedBigint    = dbi.NewDbDataType("unsigned bigint", dbi.DTUint64).WithCT(dbi.CTUnsignedInt8).WithFixColumn(dbi.ClearNumScale)
	UnsignedInt       = dbi.NewDbDataType("unsigned int", dbi.DTUint64).WithCT(dbi.CTUnsignedInt4).WithFixColumn(dbi.ClearNumScale)
	UnsignedMediumint = dbi.NewDbDataType("unsigned mediumint", dbi.DTInt64).WithCT(dbi.CTUnsignedInt4).WithFixColumn(dbi.ClearNumScale)
	UnsignedSmallint  = dbi.NewDbDataType("unsigned smallint", dbi.DTInt32).WithCT(dbi.CTUnsignedInt2).WithFixColumn(dbi.ClearNumScale)
	// unsigned tinyint最大255，无更大无符号1字节类型，以DTInt16承载避免溢出
	UnsignedTinyint = dbi.NewDbDataType("unsigned tinyint", dbi.DTInt16).WithCT(dbi.CTUnsignedInt1).WithFixColumn(dbi.ClearNumScale)

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

	Blob       = dbi.NewDbDataType("blob", DTBytesMysql).WithCT(dbi.CTBlob).WithFixColumn(dbi.ClearNumScale)
	Mediumblob = dbi.NewDbDataType("mediumblob", DTBytesMysql).WithCT(dbi.CTMediumblob).WithFixColumn(dbi.ClearNumScale)
	Longblob   = dbi.NewDbDataType("longblob", DTBytesMysql).WithCT(dbi.CTLongblob).WithFixColumn(dbi.ClearNumScale)
	Binary     = dbi.NewDbDataType("binary", DTBytesMysql).WithCT(dbi.CTBinary)
	Varbinary  = dbi.NewDbDataType("varbinary", DTBytesMysql).WithCT(dbi.CTVarbinary)
)
