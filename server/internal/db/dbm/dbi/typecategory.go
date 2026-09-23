package dbi

// TypeCategory 类型类别 — 跨方言类型映射的枢轴标识。
// 扩展方式是「在下方常量块追加类别 + 为其实例化全局降级（typeregistry.globalFallbackTypes）」，
// 属编译期开闭：新增类别时各方言的规则表无需改动，未显式注册的类别自动落到全局降级安全网。
// 注意：不支持运行时动态注册类别——类型系统所有分支（数值/字符串谓词、降级、各方言规则）
// 都对常量值做编译期 switch/match，运行时新造的类别不会命中任何分支，形同不可用。
type TypeCategory int

// 注意：TCUnknown 必须为零值，未调用 WithCategory 指定类别的列类型默认即为 TCUnknown，
// 迁移同步时会显式报错，而不是被静默当作 varchar 处理导致数据截断/损坏
const (
	TCUnknown TypeCategory = iota
	TCVarchar
	TCChar
	TCText
	TCMediumtext
	TCLongtext

	TCBit  // 1 bit
	TCBool // 1 bit
	TCInt1 // 1字节 -128~127
	TCInt2 // 2字节 -32768~32767
	TCInt4 // 4字节 -2147483648~2147483647
	TCInt8 // 8字节 -9223372036854775808~9223372036854775807
	TCNumeric
	TCDecimal

	TCUnsignedInt8
	TCUnsignedInt4
	TCUnsignedInt2
	TCUnsignedInt1

	TCDate
	TCTime
	TCDateTime
	TCTimestamp

	TCBinary
	TCVarbinary
	TCMediumblob
	TCBlob
	TCLongblob

	TCEnum
	TCJSON

	// ---- 以下为可扩展类别，新增类别无需修改任何方言代码（全局降级策略自动覆盖） ----

	TCUUID // UUID：降级为 varchar(36) 或 text

	// 扩展类别：各方言可按需注册精确映射，未注册时全局降级自动覆盖
	TCInterval // INTERVAL：日期时间差（pg/oracle）
	TCSpatial  // Geometry/Geography：空间类型（mysql/pg）
	TCXML      // XML：结构化文档
	TCArray    // Array：数组类型（pg）
	TCNetwork  // inet/cidr/macaddr：网络地址（pg）
	TCRange    // Range：范围类型（pg）
	TCHStore   // hstore：键值存储（pg）
	TCMoney    // Money：货币类型（mssql/pg）
)

// IsNumericCategory 类型类别是否为数值类（位/布尔/各宽度整数/无符号整数/定点数）。
// 归属类型系统谓词：跨方言比对（dbi/value）与迁移校验按此选择数值宽松相等策略
func IsNumericCategory(ct TypeCategory) bool {
	switch ct {
	case TCBit, TCBool, TCInt1, TCInt2, TCInt4, TCInt8, TCNumeric, TCDecimal,
		TCUnsignedInt8, TCUnsignedInt4, TCUnsignedInt2, TCUnsignedInt1:
		return true
	default:
		return false
	}
}

// allTypeCategories 所有已定义的 TypeCategory（TCUnknown 除外），供完备性测试与诊断枚举使用。
// TypeCategory 是稠密 iota 枚举，静态遍历 TCUnknown+1..TCMoney 即为全集。
var allTypeCategories = func() []TypeCategory {
	const maxCat = TCMoney // 最后一个内置类别；TCUnknown(0) 表示「未分类」，排除
	cats := make([]TypeCategory, 0, int(maxCat))
	for c := TCUnknown + 1; c <= maxCat; c++ {
		cats = append(cats, c)
	}
	return cats
}()

// AllTypeCategories 返回所有已定义的 TypeCategory（TCUnknown 除外）
func AllTypeCategories() []TypeCategory {
	cp := make([]TypeCategory, len(allTypeCategories))
	copy(cp, allTypeCategories)
	return cp
}
