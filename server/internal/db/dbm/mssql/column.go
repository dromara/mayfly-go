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

// fixMssqlTimeFsp SQL Server的datetime2/time/datetimeoffset在sys.columns中：precision是字节数（23/16），
// scale才是小数秒位数，而DDL写的是类型名(小数秒)。不换算会生成datetime2(23,7)这类非法DDL，
// 使同库结构迁移与迁移到SQL文件直接失败
func fixMssqlTimeFsp(column *dbi.Column) {
	column.NumPrecision, column.NumScale = column.NumScale, 0
	column.CharMaxLength = 0
}

// fixMssqlDecimalLength SQL Server的sys.columns对numeric/decimal/float回报的max_length是**存储字节数**
// （numeric(18,2)为9、numeric(38,5)为17、float为8），而GetColumnType优先用字符长度拼接DDL参数，
// 不则清就会拼出numeric(9)（等价numeric(9,0)）——迁移到任何库都会把所有小数静默舍入为整数；
// float同理拼成float(8)，而SQL Server的float(n<=24)实为4字节单精度，精度直接减半。
// 数值的DDL参数只能是(精度,小数位)，故清空字符长度，保留precision/scale
func fixMssqlDecimalLength(column *dbi.Column) {
	column.CharMaxLength = 0
}

var (
	// DTBytesMssql mssql专用二进制类型：hex值以mssql二进制字面量0x...保真还原
	DTBytesMssql = dbi.DTBytes.Copy().WithSQLValue(mssqlSQLValueBytes)

	// 以下类型在SQL Server不接受任何类型修饰符，而sys.columns会为它们回报无意义的precision/scale
	// （int为10、bit为1、text为16字节指针长、datetime为23,3），残留会被GetColumnType拼成
	// int(10)、text(16)、datetime(23,3)这类非法DDL，故统一清空
	Bigint   = dbi.NewDbDataType("bigint", dbi.DTInt64).WithCT(dbi.CTInt8).WithFixColumn(dbi.ClearNumPrecision)
	Int      = dbi.NewDbDataType("int", dbi.DTInt32).WithCT(dbi.CTInt4).WithFixColumn(dbi.ClearNumPrecision)
	Smallint = dbi.NewDbDataType("smallint", dbi.DTInt16).WithCT(dbi.CTInt2).WithFixColumn(dbi.ClearNumPrecision)
	Tinyint  = dbi.NewDbDataType("tinyint", dbi.DTInt8).WithCT(dbi.CTInt1).WithFixColumn(dbi.ClearNumPrecision)
	Bit      = dbi.NewDbDataType("bit", dbi.DTBit).WithCT(dbi.CTBit).WithFixColumn(dbi.ClearNumPrecision)
	// SQL Server的numeric与decimal完全同义（精确数值），必须归CTDecimal：归CTNumeric会使
	// numeric(18,2)迁入MySQL时退化为double而静默丢小数精度
	// 以下两个类型的max_length为存储字节数，必须走precision/scale取精度（见fixMssqlDecimalLength）
	Numeric    = dbi.NewDbDataType("numeric", dbi.DTNumeric).WithCT(dbi.CTDecimal).WithFixColumn(fixMssqlDecimalLength)
	Decimal    = dbi.NewDbDataType("decimal", dbi.DTDecimal).WithCT(dbi.CTDecimal).WithFixColumn(fixMssqlDecimalLength)
	Smallmoney = dbi.NewDbDataType("smallmoney", dbi.DTDecimal).WithCT(dbi.CTDecimal).WithFixColumn(dbi.ClearNumPrecision)
	Money      = dbi.NewDbDataType("money", dbi.DTDecimal).WithCT(dbi.CTDecimal).WithFixColumn(dbi.ClearNumPrecision)
	// float(n)合法（n为有效数字位数），float保持原样但需清掉字节长度；real不接受参数且回报precision=24，必须清空
	Float = dbi.NewDbDataType("float", dbi.DTNumeric).WithCT(dbi.CTNumeric).WithFixColumn(fixMssqlDecimalLength)
	// real是4字节近似浮点数（float(24)的同义词），旧版归CTVarchar会使异构迁移把它静默建成字符串列
	Real           = dbi.NewDbDataType("real", dbi.DTNumeric).WithCT(dbi.CTNumeric).WithFixColumn(dbi.ClearNumPrecision)
	Date           = dbi.NewDbDataType("date", dbi.DTDate).WithCT(dbi.CTDate).WithFixColumn(dbi.ClearNumPrecision)
	Datetimeoffset = dbi.NewDbDataType("datetimeoffset", dbi.DTDateTime).WithCT(dbi.CTDateTime).WithFixColumn(fixMssqlTimeFsp)
	Datetime2      = dbi.NewDbDataType("datetime2", dbi.DTDateTime).WithCT(dbi.CTDateTime).WithFixColumn(fixMssqlTimeFsp)
	Smalldatetime  = dbi.NewDbDataType("smalldatetime", dbi.DTDateTime).WithCT(dbi.CTDateTime).WithFixColumn(dbi.ClearNumPrecision)
	Datetime       = dbi.NewDbDataType("datetime", dbi.DTDateTime).WithCT(dbi.CTDateTime).WithFixColumn(dbi.ClearNumPrecision)
	Time           = dbi.NewDbDataType("time", dbi.DTTime).WithCT(dbi.CTTime).WithFixColumn(fixMssqlTimeFsp)
	Char           = dbi.NewDbDataType("char", dbi.DTString).WithCT(dbi.CTVarchar)
	Varchar        = dbi.NewDbDataType("varchar", dbi.DTString).WithCT(dbi.CTVarchar)
	// mssql的varchar(n)上限8000字节，超长文本需用varchar(max)（最大2GB）
	VarcharMax  = dbi.NewDbDataType("varchar(max)", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	NvarcharMax = dbi.NewDbDataType("nvarchar(max)", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Text        = dbi.NewDbDataType("text", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Nchar       = dbi.NewDbDataType("nchar", dbi.DTString).WithCT(dbi.CTVarchar)
	Nvarchar    = dbi.NewDbDataType("nvarchar", dbi.DTString).WithCT(dbi.CTVarchar)
	Ntext       = dbi.NewDbDataType("ntext", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Binary      = dbi.NewDbDataType("binary", DTBytesMssql).WithCT(dbi.CTBinary)
	Varbinary   = dbi.NewDbDataType("varbinary", DTBytesMssql).WithCT(dbi.CTBinary)
	// mssql的varbinary(n)上限8000字节，大对象需用varbinary(max)
	VarbinaryMax     = dbi.NewDbDataType("varbinary(max)", DTBytesMssql).WithCT(dbi.CTBinary).WithFixColumn(dbi.ClearNumPrecision)
	Cursor           = dbi.NewDbDataType("cursor", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Rowversion       = dbi.NewDbDataType("rowversion", DTBytesMssql).WithCT(dbi.CTBinary).WithFixColumn(dbi.ClearNumPrecision)
	Hierarchyid      = dbi.NewDbDataType("hierarchyid", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Uniqueidentifier = dbi.NewDbDataType("uniqueidentifier", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Sql_variant      = dbi.NewDbDataType("sql_variant", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Xml              = dbi.NewDbDataType("xml", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Table            = dbi.NewDbDataType("table", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Geometry         = dbi.NewDbDataType("geometry", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Geography        = dbi.NewDbDataType("geography", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
)
