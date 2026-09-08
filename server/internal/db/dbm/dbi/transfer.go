package dbi

import (
	"fmt"
	"sync"
)

type CommonDbDataType int

// common column type enum
// 注意：CTUnknown 必须为零值，未调用 WithCT 指定公共类型的列类型默认即为 CTUnknown，
// 迁移同步时会显式报错，而不是被静默当作 varchar 处理导致数据截断/损坏
const (
	CTUnknown CommonDbDataType = iota
	CTVarchar
	CTChar
	CTText
	CTMediumtext
	CTLongtext

	CTBit  // 1 bit
	CTBool // 1 bit
	CTInt1 // 1字节 -128~127
	CTInt2 // 2字节 -32768~32767
	CTInt4 // 4字节 -2147483648~2147483647
	CTInt8 // 8字节 -9223372036854775808~9223372036854775807
	CTNumeric
	CTDecimal

	CTUnsignedInt8
	CTUnsignedInt4
	CTUnsignedInt2
	CTUnsignedInt1

	CTDate
	CTTime
	CTDateTime
	CTTimestamp

	CTBinary
	CTVarbinary
	CTMediumblob
	CTBlob
	CTLongblob

	CTEnum
	CTJSON
)

type CommonTypeConverter interface {
	Varchar(*Column) *DbDataType
	Char(*Column) *DbDataType
	Text(*Column) *DbDataType
	Mediumtext(*Column) *DbDataType
	Longtext(*Column) *DbDataType

	Bit(*Column) *DbDataType
	Bool(*Column) *DbDataType
	Int1(*Column) *DbDataType
	Int2(*Column) *DbDataType
	Int4(*Column) *DbDataType
	Int8(*Column) *DbDataType
	Numeric(*Column) *DbDataType
	Decimal(*Column) *DbDataType

	UnsignedInt8(*Column) *DbDataType
	UnsignedInt4(*Column) *DbDataType
	UnsignedInt2(*Column) *DbDataType
	UnsignedInt1(*Column) *DbDataType

	Date(*Column) *DbDataType
	Time(*Column) *DbDataType
	Datetime(*Column) *DbDataType
	Timestamp(*Column) *DbDataType

	Binary(*Column) *DbDataType
	Varbinary(*Column) *DbDataType
	Mediumblob(*Column) *DbDataType
	Blob(*Column) *DbDataType
	Longblob(*Column) *DbDataType

	Enum(*Column) *DbDataType
	JSON(*Column) *DbDataType
}

var (
	commonTypeConvertersMu sync.RWMutex                                                      // 保护 commonTypeConverters 的并发读写（GetMeta 首次初始化时写入）
	commonTypeConverters   = make(map[DbType]map[CommonDbDataType]func(*Column) *DbDataType) // 公共列转换器
)

// registerCommonTypeConverter 注册公共列转换器
func registerCommonTypeConverter(dbType DbType, ctc CommonTypeConverter) {
	if ctc == nil {
		return
	}

	cts := make(map[CommonDbDataType]func(*Column) *DbDataType)
	cts[CTVarchar] = ctc.Varchar
	cts[CTChar] = ctc.Char
	cts[CTText] = ctc.Text
	cts[CTMediumtext] = ctc.Mediumtext
	cts[CTLongtext] = ctc.Longtext

	cts[CTBit] = ctc.Bit
	cts[CTBool] = ctc.Bool
	cts[CTInt1] = ctc.Int1
	cts[CTInt2] = ctc.Int2
	cts[CTInt4] = ctc.Int4
	cts[CTInt8] = ctc.Int8
	cts[CTNumeric] = ctc.Numeric
	cts[CTDecimal] = ctc.Decimal
	cts[CTUnsignedInt8] = ctc.UnsignedInt8
	cts[CTUnsignedInt4] = ctc.UnsignedInt4
	cts[CTUnsignedInt2] = ctc.UnsignedInt2
	cts[CTUnsignedInt1] = ctc.UnsignedInt1

	cts[CTDate] = ctc.Date
	cts[CTTime] = ctc.Time
	cts[CTDateTime] = ctc.Datetime
	cts[CTTimestamp] = ctc.Timestamp

	cts[CTBinary] = ctc.Binary
	cts[CTVarbinary] = ctc.Varbinary
	cts[CTMediumblob] = ctc.Mediumblob
	cts[CTBlob] = ctc.Blob
	cts[CTLongblob] = ctc.Longblob

	cts[CTEnum] = ctc.Enum
	cts[CTJSON] = ctc.JSON

	commonTypeConvertersMu.Lock()
	defer commonTypeConvertersMu.Unlock()
	commonTypeConverters[dbType] = cts
}

// getCommonTypeConverters 获取指定数据库的公共类型转换器map（并发安全）
func getCommonTypeConverters(dbType DbType) map[CommonDbDataType]func(*Column) *DbDataType {
	commonTypeConvertersMu.RLock()
	defer commonTypeConvertersMu.RUnlock()
	return commonTypeConverters[dbType]
}

// ConvToTargetDbColumn 转换至异构数据库对应的列信息
func ConvToTargetDbColumn(srcDbType DbType, targetDbType DbType, targetDialect Dialect, column *Column) error {
	// 同类型数据库，不转换
	if srcDbType == targetDbType {
		return nil
	}

	if targetDialect == nil {
		return fmt.Errorf("target database dialect [%s] is nil", targetDbType)
	}

	// 需要转换至异构数据库时，需要将该字段清空，否则如mysql可以查出该值，其他数据库可能不行，会导致Column.GetColumnType错误。
	column.ColumnType = ""

	srcMap := getCommonTypeConverters(srcDbType)
	if srcMap == nil {
		return fmt.Errorf("src database type [%s] not support transfer", srcDbType)
	}

	targetMap := getCommonTypeConverters(targetDbType)
	if targetMap == nil {
		return fmt.Errorf("target database type [%s] not support transfer", targetDbType)
	}

	srcDataType := GetDbDataType(srcDbType, column.DataType)

	// 未声明公共类型的数据类型无法确认目标类型，显式报错，避免被静默当作varchar导致数据截断
	if srcDataType.CommonType == CTUnknown {
		return fmt.Errorf("src database type [%s] data type [%s] not support transfer to [%s]: unknown common type", srcDbType, srcDataType.Name, targetDbType)
	}

	convertFunc, ok := targetMap[srcDataType.CommonType]
	if !ok || convertFunc == nil {
		return fmt.Errorf("target database type [%s] not support transfer, src data type [%s] common type [%d]", targetDbType, srcDataType.Name, srcDataType.CommonType)
	}

	// 整型/布尔/位/日期类公共类型在任何方言都不接受精度与小数位参数，而部分源库元数据会为它们回报
	// 无意义的numeric_precision（SQL Server的int回报10、pg的int4回报32），残留精度被目标方言的
	// GetColumnType拼成int4(32)、date(10)这类非法DDL会使结构迁移直接失败，故统一清空。
	// 日期时间类（CTDateTime/CTTimestamp/CTTime）不清：其小数秒精度就存在NumPrecision，由各目标转换归一
	switch srcDataType.CommonType {
	case CTInt1, CTInt2, CTInt4, CTInt8,
		CTUnsignedInt1, CTUnsignedInt2, CTUnsignedInt4, CTUnsignedInt8,
		CTBool, CTBit, CTDate:
		column.NumPrecision = 0
		column.NumScale = 0
	case CTTime, CTDateTime, CTTimestamp:
		// 时间类的fsp只存于NumPrecision，NumScale在任何方言都无语义；不清会拼出datetime(3,2)非法DDL，
		// 而清空不能复用整型分支——那会一并抹掉fsp导致小数秒静默丢失
		column.NumScale = 0
	case CTNumeric, CTDecimal:
		// 数值类型的DDL参数只能是(精度,小数位)，而部分源库会同时回报“字符长度”（SQL Server的
		// max_length对numeric(18,2)是9字节），GetColumnType优先取字符长度会拼成decimal(9)——
		// 等价decimal(9,0)，迁入目标库后所有小数被静默舍入为整数（无报错），故必须清除
		column.CharMaxLength = 0
	}

	// 获取目标数据库的数据类型，并进行可能存在的列信息修复，如长度、精度等
	targetDbDataType := convertFunc(column)
	if targetDbDataType == nil {
		return fmt.Errorf("target database type [%s] not support transfer, src data type [%s] common type [%d]", targetDbType, srcDataType.Name, srcDataType.CommonType)
	}

	// 替换为目标数据库的数据类型
	column.DataType = targetDbDataType.Name
	// 目标为字符串/文本类类型时，必须清除源列残留的数值精度与小数位：未注册的类型（如sqlite声明的
	// decimal(10,2)）会回退为varchar，残留精度被Column.GetColumnType拼成 varchar(10,2) 这类非法DDL，
	// 使整表结构迁移直接失败
	if targetDbDataType.isStringCommonType() {
		column.NumPrecision = 0
		column.NumScale = 0
	}
	return nil
}

// isStringCommonType 目标公共类型是否属于字符串/文本类（不接受数值精度与小数位）
func (ct *DbDataType) isStringCommonType() bool {
	switch ct.CommonType {
	case CTVarchar, CTChar, CTText, CTMediumtext, CTLongtext, CTEnum, CTJSON:
		return true
	}
	return false
}
