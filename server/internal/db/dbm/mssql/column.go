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

// normalizeMssqlColumnType 按sys.columns回报的max_length归一列类型名与字符长度：
//   - max_length = -1 表示 (max) 不限长形态（varchar(max)/nvarchar(max)/varbinary(max)，最大2GB），
//     必须把“(max)”还入类型名，否则直接拼会生成varchar(-1)非法DDL；
//   - nchar/nvarchar的max_length是字节数，而DDL与各方言列长以字符计，不换算会使迁入异构库时长度翻倍；
//   - text/ntext/xml回报16（LOB指针大小）或无意义值，由其FixColumn清空。
//
// 两者顺序不能倒：Go整型除法向零截断使 -1/2 == 0，先换算会让nvarchar(max)退化为“长度0的nvarchar”，
// 同方言DDL拼成nvarchar(1)（所有超长值静默截断且无报错），异构迁移则因取不到长度被兜底为varchar(255)
func normalizeMssqlColumnType(dataType string, charMaxLength int) (string, int) {
	if charMaxLength < 0 {
		switch dataType {
		case "varchar", "nvarchar", "varbinary":
			dataType += "(max)"
		}
		return dataType, 0
	}
	if dataType == "nchar" || dataType == "nvarchar" {
		return dataType, charMaxLength / 2
	}
	return dataType, charMaxLength
}

// mssqlSQLValueString 字符串值转SQL：必须输出N'...'Unicode字面量。
// SQL Server对裸'...'字面量按**数据库排序规则的代码页**解释（本机实测：默认SQL_Latin1_General_CP1_CI_AS库下
// INSERT INTO t(v) VALUES('中文😀abc')落入NVARCHAR列读回'????abc'——LEN仍为7而看不出丢数据），
// 而N'...'以Unicode解析，后才与列类型无关地保真；此行为与驱动无关（dump/导入均以语句文本下发）
func mssqlSQLValueString(val any) string {
	if val == nil {
		return dbi.NULL
	}
	// SQLValueString对非nil入参必然返回带引号字面量，直接前缀N
	return "N" + dbi.SQLValueString(val)
}

var (
	// DTBytesMssql mssql专用二进制类型：hex值以mssql二进制字面量0x...保真还原
	DTBytesMssql = dbi.DTBytes.Copy().WithSQLValue(mssqlSQLValueBytes)

	// DTStringMssql mssql专用字符串类型：字面量带N前缀（见mssqlSQLValueString），
	// 否则异构迁移/备份恢复中的非ASCII文本会被静默替换为'?'且无任何报错
	DTStringMssql = dbi.DTString.Copy().WithSQLValue(mssqlSQLValueString)

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
	// 字符类类型统一用DTStringMssql（N'...'字面量）：SQL Server的varchar/text受库代码页限制，
	// 能存下的非ASCII文本也必须以Unicode字面量下发才不会在解析阶段被替换为'?'
	Char    = dbi.NewDbDataType("char", DTStringMssql).WithCT(dbi.CTVarchar)
	Varchar = dbi.NewDbDataType("varchar", DTStringMssql).WithCT(dbi.CTVarchar)
	// mssql的varchar(n)上限8000字节，超长文本需用varchar(max)（最大2GB）
	//
	// (max)/text/ntext/xml属无限长字符类型（上限2GB），必须归CTLongtext而非CTVarchar：
	// SQL Server对max列回报sys.columns.max_length = -1，归CTVarchar会使异构目标取不到长度，
	// 被mysql的sqlgen兜底为varchar(255)（见mysql/sqlgen.go），超长文本迁入时报Data too long或静默截断
	// （本机mssql→mysql IT实测）；与oracle/dm将CLOB/TEXT/LONGVARCHAR归CTText/CTLongtext的做法同构
	VarcharMax  = dbi.NewDbDataType("varchar(max)", DTStringMssql).WithCT(dbi.CTLongtext).WithFixColumn(dbi.ClearNumPrecision)
	NvarcharMax = dbi.NewDbDataType("nvarchar(max)", DTStringMssql).WithCT(dbi.CTLongtext).WithFixColumn(dbi.ClearNumPrecision)
	Text        = dbi.NewDbDataType("text", DTStringMssql).WithCT(dbi.CTLongtext).WithFixColumn(dbi.ClearNumPrecision)
	Nchar       = dbi.NewDbDataType("nchar", DTStringMssql).WithCT(dbi.CTVarchar)
	Nvarchar    = dbi.NewDbDataType("nvarchar", DTStringMssql).WithCT(dbi.CTVarchar)
	Ntext       = dbi.NewDbDataType("ntext", DTStringMssql).WithCT(dbi.CTLongtext).WithFixColumn(dbi.ClearNumPrecision)
	Binary      = dbi.NewDbDataType("binary", DTBytesMssql).WithCT(dbi.CTBinary)
	Varbinary   = dbi.NewDbDataType("varbinary", DTBytesMssql).WithCT(dbi.CTBinary)
	// mssql的varbinary(n)上限8000字节，大对象需用varbinary(max)（上限2GB，同属无限长类型，理由见上文）
	VarbinaryMax     = dbi.NewDbDataType("varbinary(max)", DTBytesMssql).WithCT(dbi.CTLongblob).WithFixColumn(dbi.ClearNumPrecision)
	Cursor           = dbi.NewDbDataType("cursor", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Rowversion       = dbi.NewDbDataType("rowversion", DTBytesMssql).WithCT(dbi.CTBinary).WithFixColumn(dbi.ClearNumPrecision)
	Hierarchyid      = dbi.NewDbDataType("hierarchyid", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Uniqueidentifier = dbi.NewDbDataType("uniqueidentifier", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Sql_variant      = dbi.NewDbDataType("sql_variant", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Xml              = dbi.NewDbDataType("xml", dbi.DTString).WithCT(dbi.CTLongtext).WithFixColumn(dbi.ClearNumPrecision)
	Table            = dbi.NewDbDataType("table", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Geometry         = dbi.NewDbDataType("geometry", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
	Geography        = dbi.NewDbDataType("geography", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)
)
