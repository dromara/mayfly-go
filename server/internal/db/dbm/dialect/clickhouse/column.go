package clickhouse

import (
	"mayfly-go/internal/db/dbm/dbi"
)

// ClickHouse data types
var (
	// Numeric types
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

	// String types
	// ClickHouse字符串字面量与mysql同为反斜杠转义语义（' \\ \n \r \t \0 \b等），
	// 若反斜杠原样写入，含\\、\n等内容的字符串/JSON会被静默解释为转义字符导致数据损坏
	String      = dbi.NewDbDataType("String", DTStringCh).WithCategory(dbi.TCVarchar)
	FixedString = dbi.NewDbDataType("FixedString", DTStringCh).WithCategory(dbi.TCChar)

	// Date and time types
	DateTime   = dbi.NewDbDataType("DateTime", dbi.DTDateTime).WithCategory(dbi.TCDateTime)
	DateTime64 = dbi.NewDbDataType("DateTime64", dbi.DTDateTime).WithCategory(dbi.TCDateTime)
	Date       = dbi.NewDbDataType("Date", dbi.DTDate).WithCategory(dbi.TCDate)
	Date32     = dbi.NewDbDataType("Date32", dbi.DTDate).WithCategory(dbi.TCDate)

	// Other types
	UUID = dbi.NewDbDataType("UUID", DTStringCh).WithCategory(dbi.TCVarchar)
	IPv4 = dbi.NewDbDataType("IPv4", DTStringCh).WithCategory(dbi.TCVarchar)
	IPv6 = dbi.NewDbDataType("IPv6", DTStringCh).WithCategory(dbi.TCVarchar)
	Bool = dbi.NewDbDataType("Bool", dbi.DTBool).WithCategory(dbi.TCBool)

	// Decimal types
	Decimal    = dbi.NewDbDataType("Decimal", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Decimal32  = dbi.NewDbDataType("Decimal32", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Decimal64  = dbi.NewDbDataType("Decimal64", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Decimal128 = dbi.NewDbDataType("Decimal128", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Decimal256 = dbi.NewDbDataType("Decimal256", dbi.DTDecimal).WithCategory(dbi.TCDecimal)

	// Enum types
	Enum8  = dbi.NewDbDataType("Enum8", DTStringCh).WithCategory(dbi.TCEnum)
	Enum16 = dbi.NewDbDataType("Enum16", DTStringCh).WithCategory(dbi.TCEnum)

	// Complex types（Array/Tuple/Map等以字符串字面量输出，同样遵循CH转义语义）
	Array                   = dbi.NewDbDataType("Array", DTStringCh).WithCategory(dbi.TCVarchar)
	Tuple                   = dbi.NewDbDataType("Tuple", DTStringCh).WithCategory(dbi.TCVarchar)
	Map                     = dbi.NewDbDataType("Map", DTStringCh).WithCategory(dbi.TCVarchar)
	Nested                  = dbi.NewDbDataType("Nested", DTStringCh).WithCategory(dbi.TCVarchar)
	AggregateFunction       = dbi.NewDbDataType("AggregateFunction", DTStringCh).WithCategory(dbi.TCVarchar)
	SimpleAggregateFunction = dbi.NewDbDataType("SimpleAggregateFunction", DTStringCh).WithCategory(dbi.TCVarchar)

	// Special types
	LowCardinality = dbi.NewDbDataType("LowCardinality", DTStringCh).WithCategory(dbi.TCVarchar)
	Nullable       = dbi.NewDbDataType("Nullable", DTStringCh).WithCategory(dbi.TCVarchar)
)

// DTStringCh ClickHouse专用字符串类型：CH字符串字面量与mysql同为反斜杠转义语义，
// 复用mysql转义规则（'双写、反斜杠双写），否则含\\、\n等内容的字符串/JSON会静默损坏
var DTStringCh = dbi.DTString.Copy().WithSQLValue(dbi.SQLValueStringEscapeBackslash)

// Get all ClickHouse data types as a map for easy lookup
func GetAllClickHouseDataTypes() map[string]*dbi.DbDataType {
	return map[string]*dbi.DbDataType{
		"UInt8": UInt8, "UInt16": UInt16, "UInt32": UInt32, "UInt64": UInt64,
		"Int8": Int8, "Int16": Int16, "Int32": Int32, "Int64": Int64,
		"Float32": Float32, "Float64": Float64,
		"String": String, "FixedString": FixedString,
		"DateTime": DateTime, "DateTime64": DateTime64, "Date": Date, "Date32": Date32,
		"UUID": UUID, "IPv4": IPv4, "IPv6": IPv6, "Bool": Bool,
		"Decimal": Decimal, "Decimal32": Decimal32, "Decimal64": Decimal64,
		"Decimal128": Decimal128, "Decimal256": Decimal256,
		"Enum8": Enum8, "Enum16": Enum16,
		"Array": Array, "Tuple": Tuple, "Map": Map, "Nested": Nested,
		"AggregateFunction": AggregateFunction, "SimpleAggregateFunction": SimpleAggregateFunction,
		"LowCardinality": LowCardinality, "Nullable": Nullable,
	}
}
