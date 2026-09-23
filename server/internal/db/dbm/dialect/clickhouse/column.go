package clickhouse

import (
	"mayfly-go/internal/db/dbm/dbi"
)

// ClickHouse 数据类型清单
var (
	// 数值类型
	UInt8  = dbi.NewDbDataType("UInt8", dbi.DTInt8).WithCategory(dbi.TCInt1)
	UInt16 = dbi.NewDbDataType("UInt16", dbi.DTInt16).WithCategory(dbi.TCInt2)
	UInt32 = dbi.NewDbDataType("UInt32", dbi.DTInt32).WithCategory(dbi.TCInt4)
	UInt64 = dbi.NewDbDataType("UInt64", dbi.DTInt64).WithCategory(dbi.TCInt8)
	Int8   = dbi.NewDbDataType("Int8", dbi.DTInt8).WithCategory(dbi.TCInt1)
	Int16  = dbi.NewDbDataType("Int16", dbi.DTInt16).WithCategory(dbi.TCInt2)
	Int32  = dbi.NewDbDataType("Int32", dbi.DTInt32).WithCategory(dbi.TCInt4)
	Int64  = dbi.NewDbDataType("Int64", dbi.DTInt64).WithCategory(dbi.TCInt8)

	Float32 = dbi.NewDbDataType("Float32", dbi.DTNumeric).WithCategory(dbi.TCNumeric)
	Float64 = dbi.NewDbDataType("Float64", dbi.DTNumeric).WithCategory(dbi.TCNumeric)

	// 字符串类型
	// ClickHouse字符串字面量与mysql同为反斜杠转义语义（' \\ \n \r \t \0 \b等），
	// 若反斜杠原样写入，含\\、\n等内容的字符串/JSON会被静默解释为转义字符导致数据损坏
	String      = dbi.NewDbDataType("String", DTStringCh).WithCategory(dbi.TCVarchar)
	FixedString = dbi.NewDbDataType("FixedString", DTStringCh).WithCategory(dbi.TCChar)

	// 日期时间类型
	DateTime   = dbi.NewDbDataType("DateTime", dbi.DTDateTime).WithCategory(dbi.TCDateTime)
	DateTime64 = dbi.NewDbDataType("DateTime64", dbi.DTDateTime).WithCategory(dbi.TCDateTime)
	Date       = dbi.NewDbDataType("Date", dbi.DTDate).WithCategory(dbi.TCDate)
	Date32     = dbi.NewDbDataType("Date32", dbi.DTDate).WithCategory(dbi.TCDate)

	// 其他类型
	UUID = dbi.NewDbDataType("UUID", DTStringCh).WithCategory(dbi.TCVarchar)
	IPv4 = dbi.NewDbDataType("IPv4", DTStringCh).WithCategory(dbi.TCVarchar)
	IPv6 = dbi.NewDbDataType("IPv6", DTStringCh).WithCategory(dbi.TCVarchar)
	Bool = dbi.NewDbDataType("Bool", dbi.DTBool).WithCategory(dbi.TCBool)

	// 定点数类型
	Decimal    = dbi.NewDbDataType("Decimal", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Decimal32  = dbi.NewDbDataType("Decimal32", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Decimal64  = dbi.NewDbDataType("Decimal64", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Decimal128 = dbi.NewDbDataType("Decimal128", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Decimal256 = dbi.NewDbDataType("Decimal256", dbi.DTDecimal).WithCategory(dbi.TCDecimal)

	// 枚举类型
	Enum8  = dbi.NewDbDataType("Enum8", DTStringCh).WithCategory(dbi.TCEnum)
	Enum16 = dbi.NewDbDataType("Enum16", DTStringCh).WithCategory(dbi.TCEnum)

	// 复杂类型（Array/Tuple/Map等以字符串字面量输出，同样遵循CH转义语义）
	Array                   = dbi.NewDbDataType("Array", DTStringCh).WithCategory(dbi.TCVarchar)
	Tuple                   = dbi.NewDbDataType("Tuple", DTStringCh).WithCategory(dbi.TCVarchar)
	Map                     = dbi.NewDbDataType("Map", DTStringCh).WithCategory(dbi.TCVarchar)
	Nested                  = dbi.NewDbDataType("Nested", DTStringCh).WithCategory(dbi.TCVarchar)
	AggregateFunction       = dbi.NewDbDataType("AggregateFunction", DTStringCh).WithCategory(dbi.TCVarchar)
	SimpleAggregateFunction = dbi.NewDbDataType("SimpleAggregateFunction", DTStringCh).WithCategory(dbi.TCVarchar)

	// 特殊类型
	LowCardinality = dbi.NewDbDataType("LowCardinality", DTStringCh).WithCategory(dbi.TCVarchar)
	Nullable       = dbi.NewDbDataType("Nullable", DTStringCh).WithCategory(dbi.TCVarchar)
)

// DTStringCh ClickHouse专用字符串类型：CH字符串字面量与mysql同为反斜杠转义语义，
// 复用mysql转义规则（'双写、反斜杠双写），否则含\\、\n等内容的字符串/JSON会静默损坏
var DTStringCh = dbi.DTString.Copy().WithSQLValue(dbi.SQLValueStringEscapeBackslash)
