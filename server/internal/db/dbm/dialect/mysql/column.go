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

	Bit       = dbi.NewDbDataType("bit", dbi.DTBit).WithCategory(dbi.TCBit)
	Tinyint   = dbi.NewDbDataType("tinyint", dbi.DTInt8).WithCategory(dbi.TCInt1).WithFixColumn(dbi.ClearNumScale)
	Smallint  = dbi.NewDbDataType("smallint", dbi.DTInt16).WithCategory(dbi.TCInt2).WithFixColumn(dbi.ClearNumScale)
	Mediumint = dbi.NewDbDataType("mediumint", dbi.DTInt32).WithCategory(dbi.TCInt4).WithFixColumn(dbi.ClearNumScale)
	Int       = dbi.NewDbDataType("int", dbi.DTInt32).WithCategory(dbi.TCInt4).WithFixColumn(dbi.ClearNumScale)
	Bigint    = dbi.NewDbDataType("bigint", dbi.DTInt64).WithCategory(dbi.TCInt8).WithFixColumn(dbi.ClearNumScale)

	UnsignedBigint    = dbi.NewDbDataType("unsigned bigint", dbi.DTUint64).WithCategory(dbi.TCUnsignedInt8).WithFixColumn(dbi.ClearNumScale)
	UnsignedInt       = dbi.NewDbDataType("unsigned int", dbi.DTUint64).WithCategory(dbi.TCUnsignedInt4).WithFixColumn(dbi.ClearNumScale)
	UnsignedMediumint = dbi.NewDbDataType("unsigned mediumint", dbi.DTInt64).WithCategory(dbi.TCUnsignedInt4).WithFixColumn(dbi.ClearNumScale)
	UnsignedSmallint  = dbi.NewDbDataType("unsigned smallint", dbi.DTInt32).WithCategory(dbi.TCUnsignedInt2).WithFixColumn(dbi.ClearNumScale)
	// unsigned tinyint最大255，无更大无符号1字节类型，以DTInt16承载避免溢出
	UnsignedTinyint = dbi.NewDbDataType("unsigned tinyint", dbi.DTInt16).WithCategory(dbi.TCUnsignedInt1).WithFixColumn(dbi.ClearNumScale)

	Decimal = dbi.NewDbDataType("decimal", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Double  = dbi.NewDbDataType("double", dbi.DTNumeric).WithCategory(dbi.TCNumeric).WithFixColumn(dbi.ClearNumPrecision)
	Float   = dbi.NewDbDataType("float", dbi.DTNumeric).WithCategory(dbi.TCNumeric)

	Varchar    = dbi.NewDbDataType("varchar", DTStringMysql).WithCategory(dbi.TCVarchar)
	Char       = dbi.NewDbDataType("char", DTStringMysql).WithCategory(dbi.TCChar)
	Text       = dbi.NewDbDataType("text", DTStringMysql).WithCategory(dbi.TCText).WithFixColumn(dbi.ClearCharMaxLength)
	Mediumtext = dbi.NewDbDataType("mediumtext", DTStringMysql).WithCategory(dbi.TCMediumtext).WithFixColumn(dbi.ClearCharMaxLength)
	Longtext   = dbi.NewDbDataType("longtext", DTStringMysql).WithCategory(dbi.TCLongtext).WithFixColumn(dbi.ClearCharMaxLength)
	JSON       = dbi.NewDbDataType("json", DTStringMysql).WithCategory(dbi.TCJSON).WithFixColumn(dbi.ClearCharMaxLength)

	Datetime  = dbi.NewDbDataType("datetime", dbi.DTDateTime).WithCategory(dbi.TCDateTime)
	Date      = dbi.NewDbDataType("date", dbi.DTDate).WithCategory(dbi.TCDate)
	Time      = dbi.NewDbDataType("time", dbi.DTTime).WithCategory(dbi.TCTime)
	Timestamp = dbi.NewDbDataType("timestamp", dbi.DTDateTime).WithCategory(dbi.TCTimestamp)

	Enum = dbi.NewDbDataType("enum", DTStringMysql).WithCategory(dbi.TCEnum)
	Set  = dbi.NewDbDataType("set", DTStringMysql).WithCategory(dbi.TCVarchar)

	Blob = dbi.NewDbDataType("blob", DTBytesMysql).WithCategory(dbi.TCBlob).WithFixColumn(dbi.ClearNumScale)
	// Tinyblob 必须显式注册：未注册时落入Default（string通道），dump链路中二进制值会以
	// 文本字面量而非X'...'十六进制字面量输出，字节被静默翻倍失真（大数据量IT实测抓出）
	Tinyblob   = dbi.NewDbDataType("tinyblob", DTBytesMysql).WithCategory(dbi.TCBlob).WithFixColumn(dbi.ClearNumScale)
	Mediumblob = dbi.NewDbDataType("mediumblob", DTBytesMysql).WithCategory(dbi.TCMediumblob).WithFixColumn(dbi.ClearNumScale)
	Longblob   = dbi.NewDbDataType("longblob", DTBytesMysql).WithCategory(dbi.TCLongblob).WithFixColumn(dbi.ClearNumScale)
	Binary     = dbi.NewDbDataType("binary", DTBytesMysql).WithCategory(dbi.TCBinary)
	Varbinary  = dbi.NewDbDataType("varbinary", DTBytesMysql).WithCategory(dbi.TCVarbinary)
)
