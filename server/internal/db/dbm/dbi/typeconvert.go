package dbi

import "fmt"

// ---------------- 跨方言列类型转换（通过 TypeEngine） ----------------

// isStringCategory 类型类别是否属于字符串/文本类（不接受数值精度与小数位）
func (ct *DbDataType) isStringCategory() bool {
	switch ct.Category() {
	case TCVarchar, TCChar, TCText, TCMediumtext, TCLongtext, TCEnum, TCJSON, TCUUID,
		TCInterval, TCXML, TCArray, TCNetwork, TCRange, TCHStore, TCSpatial:
		return true
	}
	return false
}

// ConvToTargetDbColumn 转换至异构数据库对应的列信息。
// 函数签名不变，内部实现改为通过 TypeEngine 查找（规则 → 降级二级解析）。
func ConvToTargetDbColumn(srcDbType DbType, targetDbType DbType, targetDialect Dialect, column *Column) error {
	// 同类型数据库，不转换
	if srcDbType == targetDbType {
		return nil
	}

	if targetDialect == nil {
		return fmt.Errorf("target database dialect [%s] is nil", targetDbType)
	}

	// 需要转换至异构数据库时，清空缓存的列类型
	column.ColumnType = ""

	srcEngine := GetTypeEngine(srcDbType)
	if srcEngine == nil {
		return fmt.Errorf("src database type [%s] has no type engine registered", srcDbType)
	}

	tgtEngine := GetTypeEngine(targetDbType)
	if tgtEngine == nil {
		return fmt.Errorf("target database type [%s] has no type engine registered", targetDbType)
	}

	// 查找源类型的 TypeCategory
	srcDataType, ok := srcEngine.ResolveType(column.DataType)
	if !ok {
		return fmt.Errorf("src database type [%s] data type [%s] not found in type engine", srcDbType, column.DataType)
	}

	category := srcDataType.Category()
	if category == TCUnknown {
		return fmt.Errorf("src database type [%s] data type [%s] not support transfer to [%s]: unknown type category",
			srcDbType, srcDataType.Name, targetDbType)
	}

	// 按类别清理列参数（整型/布尔/位/日期清精度，时间类清小数位，数值类清字符长度）
	cleanupColumnParams(category, column)

	// 通过目标引擎解析目标类型（规则 → 降级二级查找）
	targetDbDataType, err := tgtEngine.ResolveTarget(category, column)
	if err != nil {
		return fmt.Errorf("target database type [%s] not support transfer, src data type [%s] category [%d]: %w",
			targetDbType, srcDataType.Name, category, err)
	}

	// 替换为目标数据库的数据类型
	column.DataType = targetDbDataType.Name

	// 目标为字符串/文本类类型时，清除源列残留的数值精度与小数位
	if targetDbDataType.isStringCategory() {
		column.NumPrecision = 0
		column.NumScale = 0
		// 非 varchar/char 的文本类目标（text/mediumtext/longtext 等）不接受字符长度参数
		switch targetDbDataType.Category() {
		case TCText, TCMediumtext, TCLongtext, TCJSON, TCXML, TCHStore, TCArray, TCSpatial:
			column.CharMaxLength = 0
		}
	}

	return nil
}

// cleanupColumnParams 按 TypeCategory 清理列的冗余参数（整型/布尔/位/日期清精度，时间类清小数位，数值类清字符长度）。
func cleanupColumnParams(category TypeCategory, column *Column) {
	switch category {
	case TCInt1, TCInt2, TCInt4, TCInt8,
		TCUnsignedInt1, TCUnsignedInt2, TCUnsignedInt4, TCUnsignedInt8,
		TCBool, TCBit, TCDate:
		// 整型/布尔/位/日期类在任何方言都不接受精度与小数位参数
		column.NumPrecision = 0
		column.NumScale = 0
	case TCTime, TCDateTime, TCTimestamp:
		// 时间类的小数秒精度存于 NumPrecision，NumScale 在任何方言都无语义
		column.NumScale = 0
	case TCNumeric, TCDecimal:
		// 数值类型的 DDL 参数只能是(精度,小数位)，需清除字符长度
		column.CharMaxLength = 0
	}
}
