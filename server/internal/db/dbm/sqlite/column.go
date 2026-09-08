package sqlite

import "mayfly-go/internal/db/dbm/dbi"

var (
	Integer = dbi.NewDbDataType("integer", dbi.DTInt64).WithCT(dbi.CTInt8)
	Real    = dbi.NewDbDataType("real", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	Text    = dbi.NewDbDataType("text", dbi.DTString).WithCT(dbi.CTText)
	Blob    = dbi.NewDbDataType("blob", dbi.DTBytes).WithCT(dbi.CTBlob)

	DateTime = dbi.NewDbDataType("datetime", dbi.DTDateTime).WithCT(dbi.CTDateTime)
	Date     = dbi.NewDbDataType("date", dbi.DTDate).WithCT(dbi.CTDate)
	Time     = dbi.NewDbDataType("time", dbi.DTTime).WithCT(dbi.CTTime)

	// 以下为SQLite官方文档「列亲和性规则」中常见的类型名别名（sqlite.org/datatype3.html）：
	// SQLite是弱类型库，列声明类型只决定存储亲和性，任意写法（含MySQL/PG风格的类型名）都合法，
	// 但未注册的类型会回退为DefaultDbDataType（CTVarchar），使迁移到强类型库时
	// 整型/日期/数值列被静默改成varchar（列类型失真），故按亲和性逐条注册。
	// 别名与对应亲和性标准类型（Integer/Real/Text等）使用相同的Go数据类型与公共类型，行为保持一致。

	// INTEGER亲和（声明类型含"INT"）：SQLite的INTEGER一律64位，统一映射CTInt8避免迁移后溢出
	Int       = dbi.NewDbDataType("int", dbi.DTInt64).WithCT(dbi.CTInt8)
	Int2      = dbi.NewDbDataType("int2", dbi.DTInt64).WithCT(dbi.CTInt8)
	Int4      = dbi.NewDbDataType("int4", dbi.DTInt64).WithCT(dbi.CTInt8)
	Int8      = dbi.NewDbDataType("int8", dbi.DTInt64).WithCT(dbi.CTInt8)
	Tinyint   = dbi.NewDbDataType("tinyint", dbi.DTInt64).WithCT(dbi.CTInt8)
	Smallint  = dbi.NewDbDataType("smallint", dbi.DTInt64).WithCT(dbi.CTInt8)
	Mediumint = dbi.NewDbDataType("mediumint", dbi.DTInt64).WithCT(dbi.CTInt8)
	Bigint    = dbi.NewDbDataType("bigint", dbi.DTInt64).WithCT(dbi.CTInt8)
	// 官方亲和表中唯一一个含空格的类型名（UNSIGNED BIG INT → INTEGER亲和）
	UnsignedBigint = dbi.NewDbDataType("unsigned big int", dbi.DTInt64).WithCT(dbi.CTInt8)

	// TEXT亲和（声明类型含"CHAR"/"CLOB"/"TEXT"）
	Char     = dbi.NewDbDataType("char", dbi.DTString).WithCT(dbi.CTChar)
	Varchar  = dbi.NewDbDataType("varchar", dbi.DTString).WithCT(dbi.CTVarchar)
	NVarchar = dbi.NewDbDataType("nvarchar", dbi.DTString).WithCT(dbi.CTVarchar)
	NChar    = dbi.NewDbDataType("nchar", dbi.DTString).WithCT(dbi.CTChar)
	Clob     = dbi.NewDbDataType("clob", dbi.DTString).WithCT(dbi.CTText)

	// REAL亲和（声明类型含"REAL"/"FLOA"/"DOUB"）
	Float  = dbi.NewDbDataType("float", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	Float4 = dbi.NewDbDataType("float4", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	Float8 = dbi.NewDbDataType("float8", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	Double = dbi.NewDbDataType("double", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	// 官方亲和表中的 DOUBLE PRECISION（含空格，REAL亲和）
	DoublePrecision = dbi.NewDbDataType("double precision", dbi.DTNumeric).WithCT(dbi.CTNumeric)

	// NUMERIC亲和（其余写法，如金额常用的NUMERIC/DECIMAL）：按精确数值映射，
	// 保留精度与小数位，避免被当作字符串列迁移后丢失数值语义
	Numeric = dbi.NewDbDataType("numeric", dbi.DTDecimal).WithCT(dbi.CTDecimal)
	Decimal = dbi.NewDbDataType("decimal", dbi.DTDecimal).WithCT(dbi.CTDecimal)
	Dec     = dbi.NewDbDataType("dec", dbi.DTDecimal).WithCT(dbi.CTDecimal)
	Fixed   = dbi.NewDbDataType("fixed", dbi.DTDecimal).WithCT(dbi.CTDecimal)

	// 日期时间写法：SQLite按NUMERIC亲和存储，但实际存的是文本/数字日期，
	// 迁移为日期类型才符合使用者预期（TIMESTAMP不含任何亲和关键字，属此类）
	Timestamp = dbi.NewDbDataType("timestamp", dbi.DTDateTime).WithCT(dbi.CTTimestamp)
)
