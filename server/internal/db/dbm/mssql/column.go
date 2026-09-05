package mssql

import (
	"fmt"

	"mayfly-go/internal/db/dbm/dbi"
)

// mssqlSQLValueBytes 二进制值转SQL：hex编码输出mssql二进制字面量0x...保真还原，非hex按字符串转义处理
func mssqlSQLValueBytes(val any) string {
	if val == nil {
		return dbi.NULL
	}
	if strVal, ok := val.(string); ok && dbi.IsHexString(strVal) {
		return fmt.Sprintf("0x%s", strVal)
	}
	return dbi.SQLValueString(val)
}

var (
	// DTBytesMssql mssql专用二进制类型：hex值以mssql二进制字面量0x...保真还原
	DTBytesMssql = dbi.DTBytes.Copy().WithSQLValue(mssqlSQLValueBytes)

	Bigint           = dbi.NewDbDataType("bigint", dbi.DTInt64).WithCT(dbi.CTInt8)
	Numeric          = dbi.NewDbDataType("numeric", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	Bit              = dbi.NewDbDataType("bit", dbi.DTBit).WithCT(dbi.CTBit)
	Smallint         = dbi.NewDbDataType("smallint", dbi.DTInt16).WithCT(dbi.CTInt2)
	Decimal          = dbi.NewDbDataType("decimal", dbi.DTDecimal).WithCT(dbi.CTDecimal)
	Smallmoney       = dbi.NewDbDataType("smallmoney", dbi.DTDecimal).WithCT(dbi.CTDecimal)
	Int              = dbi.NewDbDataType("int", dbi.DTInt32).WithCT(dbi.CTInt4)
	Tinyint          = dbi.NewDbDataType("tinyint", dbi.DTInt8).WithCT(dbi.CTInt1)
	Money            = dbi.NewDbDataType("money", dbi.DTDecimal).WithCT(dbi.CTDecimal)
	Float            = dbi.NewDbDataType("float", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	Real             = dbi.NewDbDataType("real", dbi.DTString).WithCT(dbi.CTVarchar)
	Date             = dbi.NewDbDataType("date", dbi.DTDate).WithCT(dbi.CTDate)
	Datetimeoffset   = dbi.NewDbDataType("datetimeoffset", dbi.DTDateTime).WithCT(dbi.CTDateTime)
	Datetime2        = dbi.NewDbDataType("datetime2", dbi.DTDateTime).WithCT(dbi.CTDateTime)
	Smalldatetime    = dbi.NewDbDataType("smalldatetime", dbi.DTDateTime).WithCT(dbi.CTDateTime)
	Datetime         = dbi.NewDbDataType("datetime", dbi.DTDateTime).WithCT(dbi.CTDateTime)
	Time             = dbi.NewDbDataType("time", dbi.DTTime).WithCT(dbi.CTTime)
	Char             = dbi.NewDbDataType("char", dbi.DTString).WithCT(dbi.CTVarchar)
	Varchar          = dbi.NewDbDataType("varchar", dbi.DTString).WithCT(dbi.CTVarchar)
	// mssql的varchar(n)上限8000字节，超长文本需用varchar(max)（最大2GB）
	VarcharMax = dbi.NewDbDataType("varchar(max)", dbi.DTString).WithCT(dbi.CTVarchar)
	NvarcharMax = dbi.NewDbDataType("nvarchar(max)", dbi.DTString).WithCT(dbi.CTVarchar)
	Text             = dbi.NewDbDataType("text", dbi.DTString).WithCT(dbi.CTVarchar)
	Nchar            = dbi.NewDbDataType("nchar", dbi.DTString).WithCT(dbi.CTVarchar)
	Nvarchar         = dbi.NewDbDataType("nvarchar", dbi.DTString).WithCT(dbi.CTVarchar)
	Ntext            = dbi.NewDbDataType("ntext", dbi.DTString).WithCT(dbi.CTVarchar)
	Binary           = dbi.NewDbDataType("binary", DTBytesMssql).WithCT(dbi.CTBinary)
	Varbinary        = dbi.NewDbDataType("varbinary", DTBytesMssql).WithCT(dbi.CTBinary)
	// mssql的varbinary(n)上限8000字节，大对象需用varbinary(max)
	VarbinaryMax = dbi.NewDbDataType("varbinary(max)", DTBytesMssql).WithCT(dbi.CTBinary)
	Cursor           = dbi.NewDbDataType("cursor", dbi.DTString).WithCT(dbi.CTVarchar)
	Rowversion       = dbi.NewDbDataType("rowversion", DTBytesMssql).WithCT(dbi.CTBinary)
	Hierarchyid      = dbi.NewDbDataType("hierarchyid", dbi.DTString).WithCT(dbi.CTVarchar)
	Uniqueidentifier = dbi.NewDbDataType("uniqueidentifier", dbi.DTString).WithCT(dbi.CTVarchar)
	Sql_variant      = dbi.NewDbDataType("sql_variant", dbi.DTString).WithCT(dbi.CTVarchar)
	Xml              = dbi.NewDbDataType("xml", dbi.DTString).WithCT(dbi.CTVarchar)
	Table            = dbi.NewDbDataType("table", dbi.DTString).WithCT(dbi.CTVarchar)
	Geometry         = dbi.NewDbDataType("geometry", dbi.DTString).WithCT(dbi.CTVarchar)
	Geography        = dbi.NewDbDataType("geography", dbi.DTString).WithCT(dbi.CTVarchar)
)
