package mssql

import "mayfly-go/internal/db/dbm/dbi"

var _ dbi.CommonTypeConverter = (*commonTypeConverter)(nil)

type commonTypeConverter struct {
}

// varcharType 字符类异构目标列统一落Unicode类型：SQL Server的varchar/char/text按库排序规则的代码页
// 存储与解析（本机IT实例为SQL_Latin1_General_CP1_CI_AS），utf8mb4源库的中文/emoji迁入会被静默替换为'?'
// 且LEN不变、无任何报错；nvarchar以UTF-16存储，才是MySQL utf8/utf8mb4的等价类型
func varcharType(col *dbi.Column) *dbi.DbDataType {
	// nvarchar(n)的n是字符数（上限4000），与源列长度语义一致；超长或无长度信息用nvarchar(max)
	// （CharMaxLength<=0对应pg/sqlite的不限长varchar，而SQL Server的nvarchar不带长度即等价
	// nvarchar(1)，会静默截断所有超长值且无任何报错，与bytesType的处理保持一致）
	if col.CharMaxLength <= 0 || col.CharMaxLength > 4000 {
		col.CharMaxLength = 0
		return NvarcharMax
	}
	return Nvarchar
}

func (c *commonTypeConverter) Varchar(col *dbi.Column) *dbi.DbDataType {
	return varcharType(col)
}

func (c *commonTypeConverter) Char(col *dbi.Column) *dbi.DbDataType {
	// 定长列保持定长语义，但同样必须Unicode化；长度未知时不能落nchar（等价nchar(1)），改用nvarchar(max)承载
	if col.CharMaxLength <= 0 || col.CharMaxLength > 4000 {
		col.CharMaxLength = 0
		return NvarcharMax
	}
	return Nchar
}

// textType 大文本归nvarchar(max)：mssql的text/ntext已被废弃（微软声明后续版本移除），
// 且text受代码页限制不保真非ASCII；nvarchar(max)与其同为大对象字符类型且语义更完整
func textType(col *dbi.Column) *dbi.DbDataType {
	// text类型无长度语法，清空源长度避免生成非法DDL
	col.CharMaxLength = 0
	return NvarcharMax
}

func (c *commonTypeConverter) Text(col *dbi.Column) *dbi.DbDataType {
	return textType(col)
}
func (c *commonTypeConverter) Mediumtext(col *dbi.Column) *dbi.DbDataType {
	return textType(col)
}
func (c *commonTypeConverter) Longtext(col *dbi.Column) *dbi.DbDataType {
	return textType(col)
}

func (c *commonTypeConverter) Bit(col *dbi.Column) *dbi.DbDataType {
	return Bit
}
func (c *commonTypeConverter) Bool(col *dbi.Column) *dbi.DbDataType {
	return Bit
}
func (c *commonTypeConverter) Int1(col *dbi.Column) *dbi.DbDataType {
	return Tinyint
}
func (c *commonTypeConverter) Int2(col *dbi.Column) *dbi.DbDataType {
	return Smallint
}
func (c *commonTypeConverter) Int4(col *dbi.Column) *dbi.DbDataType {
	return Int
}
func (c *commonTypeConverter) Int8(col *dbi.Column) *dbi.DbDataType {
	return Bigint
}

// Numeric CTNumeric代指近似数值（float/real/double precision）与无约束数值，
// 而SQL Server的numeric不声明精度即等价numeric(18,0)（静默截断小数），故浮点类源列必须落float；
// 同时清空源精度（各家库的精度单位不一致，如pg以bit计、mssql以位数计，沿用会生成错误宽度）
func (c *commonTypeConverter) Numeric(col *dbi.Column) *dbi.DbDataType {
	dbi.ClearNumPrecision(col)
	return Float
}

// Decimal SQL Server的decimal不声明精度即等价decimal(18,0)，无约束的源精确数值必须补齐最大精度避免静默截断小数；
// 精度取(38,19)：38为SQL Server上限，19位小数可使整数部分仍容纳int64（最大19位数字）
func (c *commonTypeConverter) Decimal(col *dbi.Column) *dbi.DbDataType {
	dbi.FillUnboundedDecimal(col, 38, 19)
	dbi.ClampDecimalPrecision(col, 38, 19)
	return Decimal
}

func (c *commonTypeConverter) UnsignedInt8(col *dbi.Column) *dbi.DbDataType {
	return Bigint
}
func (c *commonTypeConverter) UnsignedInt4(col *dbi.Column) *dbi.DbDataType {
	return Int
}
func (c *commonTypeConverter) UnsignedInt2(col *dbi.Column) *dbi.DbDataType {
	return Smallint
}
func (c *commonTypeConverter) UnsignedInt1(col *dbi.Column) *dbi.DbDataType {
	return Tinyint
}

func (c *commonTypeConverter) Date(col *dbi.Column) *dbi.DbDataType {
	// SQL Server的date不接受精度参数，源列残留精度会生成 date(3) 非法DDL
	dbi.ClearNumPrecision(col)
	return Date
}
func (c *commonTypeConverter) Time(col *dbi.Column) *dbi.DbDataType {
	// time不声明fsp时默认7位，但为与datetime2保持一致的归一策略，未知时显式声明最大fsp
	dbi.NormalizeTimeFsp(col, 7)
	return Time
}

// Datetime/Timestamp 目标类型用datetime2：mssql的datetime标度固定为3.33ms且不支持精度参数，
// 异构源（pg timestamp(6)、sqlite小数秒文本）迁入会静默丢失/无法表达小数秒；datetime2是微软推荐的替代类型
func (c *commonTypeConverter) Datetime(col *dbi.Column) *dbi.DbDataType {
	dbi.NormalizeTimeFsp(col, 7)
	return Datetime2
}
func (c *commonTypeConverter) Timestamp(col *dbi.Column) *dbi.DbDataType {
	dbi.NormalizeTimeFsp(col, 7)
	return Datetime2
}

func (c *commonTypeConverter) Binary(col *dbi.Column) *dbi.DbDataType {
	return bytesType(col)
}
func (c *commonTypeConverter) Varbinary(col *dbi.Column) *dbi.DbDataType {
	return bytesType(col)
}
func (c *commonTypeConverter) Mediumblob(col *dbi.Column) *dbi.DbDataType {
	return bytesType(col)
}
func (c *commonTypeConverter) Blob(col *dbi.Column) *dbi.DbDataType {
	return bytesType(col)
}
func (c *commonTypeConverter) Longblob(col *dbi.Column) *dbi.DbDataType {
	return bytesType(col)
}

// bytesType 二进制类型统一归一化：varbinary(n)上限8000且binary为定长填充（会pad 0x00），
// 源无长度或超长时用varbinary(max)承载，避免binary(1)默认长度或非法DDL
func bytesType(col *dbi.Column) *dbi.DbDataType {
	if col.CharMaxLength <= 0 || col.CharMaxLength > 8000 {
		col.CharMaxLength = 0
		return VarbinaryMax
	}
	return Varbinary
}

func (c *commonTypeConverter) Enum(col *dbi.Column) *dbi.DbDataType {
	return varcharType(col)
}
func (c *commonTypeConverter) JSON(col *dbi.Column) *dbi.DbDataType {
	// JSON文本可含任意Unicode，且mssql无原生JSON类型，用nvarchar(max)承载
	return textType(col)
}
