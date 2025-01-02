package dbi

import (
	"fmt"
)

type CommonDbDataType int

// common column type enum
const (
	CTVarchar CommonDbDataType = iota
	CTChar
	CTText
	CTMediumtext
	CTLongtext

	CTBit  // 1 bit
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
	commonTypeConverters = make(map[DbType]map[CommonDbDataType]func(*Column) *DbDataType) // 公共列转换器
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

	commonTypeConverters[dbType] = cts
}

// ConvToTargetDbColumn 转换至异构数据库对应的列信息
func ConvToTargetDbColumn(srcDbType DbType, targetDbType DbType, targetDialect Dialect, column *Column) error {
	// 同类型数据库，不转换
	if srcDbType == targetDbType {
		return nil
	}

	srcMap := commonTypeConverters[srcDbType]
	if srcMap == nil {
		return fmt.Errorf("src database type [%s] not suport transfer", srcDbType)
	}

	targetMap := commonTypeConverters[targetDbType]
	if targetMap == nil {
		return fmt.Errorf("target database type [%s] not suport transfer", targetDbType)
	}

	srcDataType := GetDbDataType(srcDbType, column.DataType)

	// 获取目标数据库的数据类型，并进行可能存在的列信息修复，如长度、精度等
	targetDbDataType := targetMap[srcDataType.CommonType](column)
	if targetDbDataType == nil {
		return fmt.Errorf("target database type [%s] not suport transfer, src data type [%d]", targetDbType, srcDataType.CommonType)
	}

	// 替换为目标数据库的数据类型
	column.DataType = targetDbDataType.Name
	return nil
}
