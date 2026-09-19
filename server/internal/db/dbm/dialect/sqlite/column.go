package sqlite

import "mayfly-go/internal/db/dbm/dbi"

var (
	Integer = dbi.NewDbDataType("integer", dbi.DTInt64).WithCategory(dbi.TCInt8)
	Real    = dbi.NewDbDataType("real", dbi.DTNumeric).WithCategory(dbi.TCNumeric)
	Text    = dbi.NewDbDataType("text", dbi.DTString).WithCategory(dbi.TCText)
	Blob    = dbi.NewDbDataType("blob", dbi.DTBytes).WithCategory(dbi.TCBlob)

	DateTime = dbi.NewDbDataType("datetime", dbi.DTDateTime).WithCategory(dbi.TCDateTime)
	Date     = dbi.NewDbDataType("date", dbi.DTDate).WithCategory(dbi.TCDate)
	Time     = dbi.NewDbDataType("time", dbi.DTTime).WithCategory(dbi.TCTime)

	// 以下为SQLite官方文档「列亲和性规则」中常见的类型名别名（sqlite.org/datatype3.html）：
	// SQLite是弱类型库，列声明类型只决定存储亲和性，任意写法（含MySQL/PG风格的类型名）都合法，
	// 但未注册的类型会回退为DefaultDbDataType（TCVarchar），使迁移到强类型库时
	// 整型/日期/数值列被静默改成varchar（列类型失真），故按亲和性逐条注册。
	// 别名与对应亲和性标准类型（Integer/Real/Text等）使用相同的Go数据类型与公共类型，行为保持一致。

	// INTEGER亲和（声明类型含"INT"）：SQLite的INTEGER一律64位，统一映射CTInt8避免迁移后溢出
	Int       = dbi.NewDbDataType("int", dbi.DTInt64).WithCategory(dbi.TCInt8)
	Int2      = dbi.NewDbDataType("int2", dbi.DTInt64).WithCategory(dbi.TCInt8)
	Int4      = dbi.NewDbDataType("int4", dbi.DTInt64).WithCategory(dbi.TCInt8)
	Int8      = dbi.NewDbDataType("int8", dbi.DTInt64).WithCategory(dbi.TCInt8)
	Tinyint   = dbi.NewDbDataType("tinyint", dbi.DTInt64).WithCategory(dbi.TCInt8)
	Smallint  = dbi.NewDbDataType("smallint", dbi.DTInt64).WithCategory(dbi.TCInt8)
	Mediumint = dbi.NewDbDataType("mediumint", dbi.DTInt64).WithCategory(dbi.TCInt8)
	Bigint    = dbi.NewDbDataType("bigint", dbi.DTInt64).WithCategory(dbi.TCInt8)
	// 官方亲和表中唯一一个含空格的类型名（UNSIGNED BIG INT → INTEGER亲和）
	UnsignedBigint = dbi.NewDbDataType("unsigned big int", dbi.DTInt64).WithCategory(dbi.TCInt8)

	// TEXT亲和（声明类型含"CHAR"/"CLOB"/"TEXT"）
	Char     = dbi.NewDbDataType("char", dbi.DTString).WithCategory(dbi.TCChar)
	Varchar  = dbi.NewDbDataType("varchar", dbi.DTString).WithCategory(dbi.TCVarchar)
	NVarchar = dbi.NewDbDataType("nvarchar", dbi.DTString).WithCategory(dbi.TCVarchar)
	NChar    = dbi.NewDbDataType("nchar", dbi.DTString).WithCategory(dbi.TCChar)
	Clob     = dbi.NewDbDataType("clob", dbi.DTString).WithCategory(dbi.TCText)

	// REAL亲和（声明类型含"REAL"/"FLOA"/"DOUB"）
	Float  = dbi.NewDbDataType("float", dbi.DTNumeric).WithCategory(dbi.TCNumeric)
	Float4 = dbi.NewDbDataType("float4", dbi.DTNumeric).WithCategory(dbi.TCNumeric)
	Float8 = dbi.NewDbDataType("float8", dbi.DTNumeric).WithCategory(dbi.TCNumeric)
	Double = dbi.NewDbDataType("double", dbi.DTNumeric).WithCategory(dbi.TCNumeric)
	// 官方亲和表中的 DOUBLE PRECISION（含空格，REAL亲和）
	DoublePrecision = dbi.NewDbDataType("double precision", dbi.DTNumeric).WithCategory(dbi.TCNumeric)

	// NUMERIC亲和（其余写法，如金额常用的NUMERIC/DECIMAL）：按精确数值映射，
	// 保留精度与小数位，避免被当作字符串列迁移后丢失数值语义
	Numeric = dbi.NewDbDataType("numeric", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Decimal = dbi.NewDbDataType("decimal", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Dec     = dbi.NewDbDataType("dec", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	Fixed   = dbi.NewDbDataType("fixed", dbi.DTDecimal).WithCategory(dbi.TCDecimal)

	// 日期时间写法：SQLite按NUMERIC亲和存储，但实际存的是文本/数字日期，
	// 迁移为日期类型才符合使用者预期（TIMESTAMP不含任何亲和关键字，属此类）
	Timestamp = dbi.NewDbDataType("timestamp", dbi.DTDateTime).WithCategory(dbi.TCTimestamp)
)
