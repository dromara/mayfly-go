package dbi

import (
	"fmt"
	"strings"

	"mayfly-go/pkg/utils/collx"
)

// 表的列信息
type Column struct {
	TableName     string `json:"tableName"`     // 表名
	ColumnName    string `json:"columnName"`    // 列名
	ColumnType    string `json:"columnType"`    // 完整列类型，带有数据类型以及长度、精度等。如varchar(2000)，decimal(20,2)
	DataType      string `json:"dataType"`      // 数据类型
	ColumnComment string `json:"columnComment"` // 列备注
	IsPrimaryKey  bool   `json:"isPrimaryKey"`  // 是否为主键
	AutoIncrement bool   `json:"autoIncrement"` // 是否自增
	ColumnDefault string `json:"columnDefault"` // 默认值
	Nullable      bool   `json:"nullable"`      // 是否可为null
	// IsExprDefault 源库元数据表明 ColumnDefault 是「表达式默认值」（函数调用/运算式）而非字面量内容。
	// 各库表达式语法与函数名互不相通（pg的gen_random_uuid()、mysql的concat(_latin1\'x\')），跨库无法还原，
	// 生成DDL时必须省略该默认值；若按字面量写出（DEFAULT 'gen_random_uuid()'），uuid/jsonb列建表即报错，
	// 字符串列则静默把函数名当成默认值内容（数据污染）。由各方言metadata读取时按自身呈现约定标记
	IsExprDefault bool `json:"isExprDefault"`
	// IsGenerated 是否为生成列（MySQL的VIRTUAL/STORED GENERATED、SQL Server的计算列、Oracle的虚拟列等）：
	// 其值由表达式派生，不可显式INSERT（MySQL报Error 3105、pg报cannot insert a non-DEFAULT value），
	// 数据迁移/导出导入必须从插入列集中剔除；COLUMN_DEFAULT存的是派生表达式而非默认值，也不得当默认值写出
	IsGenerated   bool    `json:"isGenerated"`
	CharMaxLength int     `json:"charMaxLength"` // 字符最大长度
	NumPrecision  int     `json:"numPrecision"`  // 精度(总数字位数)
	NumScale      int     `json:"numScale"`      // 小数点位数
	Extra         collx.M `json:"extra"`         // 其他额外信息
}

// SplitColumnTypeBase 拆分列类型书写形态为其基础类型名与第一个括号参数的数值：
// "datetime(3)"→("datetime",3,true)，"decimal(20,6)"→("decimal",20,true)，"varchar"→("varchar",0,false)。
// 用于方言生成器按目标列最终书写的小数秒精度归一默认值参数（如MySQL要求CURRENT_TIMESTAMP(fsp)与列fsp一致）
func SplitColumnTypeBase(columnType string) (string, int, bool) {
	lower := strings.ToLower(strings.TrimSpace(columnType))
	open := strings.IndexByte(lower, '(')
	if open < 0 || !strings.HasSuffix(lower, ")") {
		return lower, 0, false
	}
	base := strings.TrimSpace(lower[:open])
	inner := strings.TrimSpace(lower[open+1 : len(lower)-1])
	// 取第一个逗号前的数值（decimal(20,6)的精度部分），非纯数字则视为无有效参数
	if idx := strings.IndexByte(inner, ','); idx >= 0 {
		inner = inner[:idx]
	}
	num := 0
	if inner == "" {
		return base, 0, false
	}
	for i := 0; i < len(inner); i++ {
		if inner[i] < '0' || inner[i] > '9' {
			return base, 0, false
		}
		num = num*10 + int(inner[i]-'0')
	}
	return base, num, true
}

// GetColumnType 获取完整的列类型，拼接数据类型与长度等。如varchar(2000)，decimal(20,2)
func (c *Column) GetColumnType() string {
	if c.ColumnType != "" {
		return c.ColumnType
	}

	if c.CharMaxLength > 0 {
		return fmt.Sprintf("%s(%d)", c.DataType, c.CharMaxLength)
	}
	if c.NumPrecision > 0 {
		if c.NumScale > 0 {
			return fmt.Sprintf("%s(%d,%d)", c.DataType, c.NumPrecision, c.NumScale)
		} else {
			return fmt.Sprintf("%s(%d)", c.DataType, c.NumPrecision)
		}
	}

	return c.DataType
}

// ColumnExtraOnUpdate Column.Extra的key：源库元数据里的「自动更新」子句原文（如MySQL的
// on update CURRENT_TIMESTAMP(3)）。属于方言专有语法，故以扩展信息而非独立字段承载，
// 仅目标方言自身支持时（MySQL生成器）才写入DDL
const ColumnExtraOnUpdate = "onUpdate"

// 生成列的派生信息（存于Column.Extra，方言专有语法且长度不定，不占用独立字段）：
// 表达式文本属于源方言SQL，跨方言无法保证其语法与函数存在，故必须连同源库类型一起记录
const (
	// ColumnExtraGenExpr 生成列的派生表达式原文（如MySQL的 (`a` + `b`)、pg的 (c1 * 2)）
	ColumnExtraGenExpr = "genExpr"
	// ColumnExtraGenKind 生成列的物化方式：stored（结果物化存储，如MySQL STORED、pg STORED）
	// 或virtual（查询时计算）
	ColumnExtraGenKind = "genKind"
	// ColumnExtraGenDbType 表达式所属的源数据库类型，仅目标与源同方言时才可原样重建
	ColumnExtraGenDbType = "genDbType"
)

// 生成列物化方式的取值（与各方言元数据的存储语义对齐）
const (
	GenerationStored  = "stored"
	GenerationVirtual = "virtual"
)

// MarkGeneratedColumn 记录生成列的派生表达式与物化方式，供同方言迁移重建DDL以及插入时剔除该列使用。
// 仅当源库能交出完整的派生表达式文本时才记录：无表达式则无法重建，必须退回「目标建普通列 + 插入源值」的语义，
// 否则会出现目标建成普通列却又不插入该列的静默NULL失真（比直接报错更隐蔽）
func MarkGeneratedColumn(column *Column, srcDbType, expr, kind string) {
	if column == nil || expr == "" {
		return
	}
	column.IsGenerated = true
	if column.Extra == nil {
		column.Extra = collx.M{}
	}
	column.Extra[ColumnExtraGenExpr] = expr
	column.Extra[ColumnExtraGenKind] = kind
	column.Extra[ColumnExtraGenDbType] = srcDbType
}

// PreservableGeneratedColumn 生成列能否在目标方言下原样重建（而非退化为普通列）：
// 必须同时满足已取到派生表达式、且目标与表达式所属源库为同一方言。
//
// DDL生成与INSERT剔除必须共用本判定，否则会出现“目标建成了普通列却又不插入该列”
// 导致生成列值静默变NULL（比直接报错更隐蔽）的数据失真
func PreservableGeneratedColumn(column Column, targetDbType DbType) bool {
	if !column.IsGenerated {
		return false
	}
	expr, _ := column.Extra[ColumnExtraGenExpr].(string)
	if strings.TrimSpace(expr) == "" {
		return false
	}
	srcDbType, _ := column.Extra[ColumnExtraGenDbType].(string)
	return srcDbType == string(targetDbType)
}

// GeneratedColumnExpr 取生成列的派生表达式原文，无则返回空字符串
func GeneratedColumnExpr(column Column) string {
	expr, _ := column.Extra[ColumnExtraGenExpr].(string)
	return strings.TrimSpace(expr)
}

// GeneratedColumnStored 生成列是否为物化存储（stored）：仅物化存储的列才可安全重建，
// 因为部分方言语义上不支持虚拟生成列（如pg 12/13只有STORED，VIRTUAL自pg 18才存在）
func GeneratedColumnStored(column Column) bool {
	kind, _ := column.Extra[ColumnExtraGenKind].(string)
	return kind == GenerationStored
}
