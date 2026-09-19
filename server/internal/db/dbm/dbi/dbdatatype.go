package dbi

// GetDbDataType 按类型名查找方言的类型注册表，未命中返回 DefaultDbDataType。
// 委托给 TypeEngine，单一数据源。
func GetDbDataType(dbType DbType, databaseColumnType string) *DbDataType {
	if engine := GetTypeEngine(dbType); engine != nil {
		if dt, ok := engine.ResolveType(databaseColumnType); ok {
			return dt
		}
	}
	return DefaultDbDataType
}

var DefaultDbDataType = NewDbDataType("string", DTString).WithCategory(TCVarchar)

// 数据库对应的数据类型
type DbDataType struct {
	Name string //  类型名

	DataType *DataType // 数据类型

	fixColumnFunc func(column *Column) // 修复字段长度、精度等, 如mysql text会返回长度，需要将其置为0等

	/** 以下为异构数据迁移同步使用，可不赋值，无值则不支持迁移同步 */

	category TypeCategory // 对应的类型类别
}

// WithFixColumn 修复列信息函数，用于修复字段长度、精度等
func (ct *DbDataType) WithFixColumn(fixColumnFunc func(column *Column)) *DbDataType {
	ct.fixColumnFunc = fixColumnFunc
	return ct
}

// WithCategory 设置类型类别，主要用于异构数据库迁移同步时进行类型转换使用
func (ct *DbDataType) WithCategory(category TypeCategory) *DbDataType {
	ct.category = category
	return ct
}

// Category 返回类型类别
func (ct *DbDataType) Category() TypeCategory {
	return ct.category
}

// FixColumn 使用修复列信息函数进行列信息修复
func (ct *DbDataType) FixColumn(column *Column) {
	if ct.fixColumnFunc != nil {
		ct.fixColumnFunc(column)
	}
}

func NewDbDataType(name string, dataType *DataType) *DbDataType {
	return &DbDataType{
		Name:     name,
		DataType: dataType,
	}
}

// 列修正族语义对照（WithFixColumn注册用，各函数清零的字段集互不相同，误用即DDL参数失真）：
//
//	                 CharMaxLength  NumPrecision  NumScale
//	ClearCharMaxLength      0            0           -
//	ClearCharLength         0            -           -
//	ClearNumScale           0            -           0
//	ClearNumPrecision       0            0           0
//
// 归属说明：本族服务于「源端元数据归一」（各方言metadata读取自家目录后立即修正，
// 如mysql的text回报伪长度）；「目标端适配」（精度钳制/补齐/fsp归一）由各方言
// backend.go 的类型转换规则调用 FillUnboundedDecimal/ClampDecimalPrecision/
// NormalizeTimeFsp/ClampTimeFsp 完成，两条管线正交，勿混用。
// 方言一次性怪癖（如mssql把fsp存在max_length列需字段对调）直接用自定义闭包注册，
// 不必强塞进共享函数族——闭包载体是有意为之：怪癖不可枚举，声明式策略表反而制造双机制。

// ClearCharMaxLength 清空字符长度与数值精度（text/blob等类型元数据携带的冗余长度）
func ClearCharMaxLength(column *Column) {
	column.CharMaxLength = 0
	column.NumPrecision = 0
}

// ClearCharLength 仅清空字符长度：时间类列的元数据不含字符长度但含小数秒精度（存于NumPrecision），
// 使用ClearCharMaxLength会一并抹掉精度，使异构迁移后小数秒静默丢失
func ClearCharLength(column *Column) {
	column.CharMaxLength = 0
}

// ClearNumScale 清空小数位与字符长度
func ClearNumScale(column *Column) {
	column.NumScale = 0
	column.CharMaxLength = 0
}

// ClearNumPrecision 清空精度、小数位与字符长度
func ClearNumPrecision(column *Column) {
	column.NumScale = 0
	column.NumPrecision = 0
	column.CharMaxLength = 0
}

// FillUnboundedDecimal 源列为无精度约束的精确数值（如pg的numeric、oracle的NUMBER、sqlite声明的numeric）时，
// 按目标库上限补齐精度与小数位。
// MySQL的decimal省略精度等价decimal(10,0)、SQL Server的numeric默认(18,0)，若沿用无精度的源列，
// 目标表建出来后所有小数都会被静默截断（结构迁移后数据失真且无报错），故必须补齐为尽可能宽的精度
func FillUnboundedDecimal(column *Column, maxPrecision, maxScale int) {
	if column == nil || column.NumPrecision > 0 {
		return
	}
	column.NumPrecision = maxPrecision
	column.NumScale = maxScale
}

// ClampDecimalPrecision 将精度与小数位收敛到目标库支持范围内，并保证小数位不超过精度。
// 源库精度上限高于目标库时（如mysql decimal(65,30) -> oracle NUMBER(38,x)），不收敛会直接生成非法DDL
func ClampDecimalPrecision(column *Column, maxPrecision, maxScale int) {
	if column == nil || column.NumPrecision <= 0 {
		return
	}
	if column.NumScale < 0 {
		column.NumScale = 0
	}
	if column.NumScale > maxScale {
		column.NumScale = maxScale
	}
	if column.NumPrecision > maxPrecision {
		column.NumPrecision = maxPrecision
	}
	if column.NumScale > column.NumPrecision {
		column.NumScale = column.NumPrecision
	}
}

// NormalizeTimeFsp 归一化时间类列的小数秒精度（fsp），用于目标库支持且需要显式声明fsp的情况：
//   - NumScale对时间类型无语义，必须清零，否则会被GetColumnType拼成 datetime(6,2) 这类非法DDL
//   - 源库未提供fsp时按目标库最大fsp补齐：MySQL的datetime省略fsp即datetime(0)，源值的小数秒会在迁移后静默丢失
//   - 源fsp大于目标库上限时收敛到上限（如mssql datetime2(7) -> mysql datetime(6)）
func NormalizeTimeFsp(column *Column, maxFsp int) {
	if column == nil {
		return
	}
	column.NumScale = 0
	if column.NumPrecision <= 0 || column.NumPrecision > maxFsp {
		column.NumPrecision = maxFsp
	}
}

// ClampTimeFsp 仅收敛超出目标库上限的小数秒精度：目标库不声明fsp时即为自身最大精度（如pg的timestamp）时使用，
// 此时无需也不应主动补齐精度
func ClampTimeFsp(column *Column, maxFsp int) {
	if column == nil {
		return
	}
	column.NumScale = 0
	if column.NumPrecision > maxFsp {
		column.NumPrecision = maxFsp
	}
}
