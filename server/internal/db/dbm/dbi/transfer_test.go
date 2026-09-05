package dbi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// CTUnknown 必须为零值：未 WithCT 的类型默认即为 CTUnknown，迁移时须显式报错
func TestCTUnknownIsZeroValue(t *testing.T) {
	var zero CommonDbDataType
	assert.Equal(t, zero, CTUnknown)
	assert.Equal(t, CommonDbDataType(0), CTUnknown)

	// 未调用WithCT的类型CommonType即为CTUnknown
	assert.Equal(t, CTUnknown, NewDbDataType("unknown-type", DTString).CommonType)
}

// testConverter 测试用公共类型转换器（仅实现测试关注的方法返回非nil）
type testConverter struct{}

func (c *testConverter) Varchar(*Column) *DbDataType { return NewDbDataType("tgt_varchar", DTString).WithCT(CTVarchar) }
func (c *testConverter) Char(*Column) *DbDataType    { return NewDbDataType("tgt_char", DTString).WithCT(CTChar) }
func (c *testConverter) Text(*Column) *DbDataType    { return NewDbDataType("tgt_text", DTString).WithCT(CTText) }
func (c *testConverter) Mediumtext(*Column) *DbDataType { return NewDbDataType("tgt_mediumtext", DTString).WithCT(CTMediumtext) }
func (c *testConverter) Longtext(*Column) *DbDataType   { return NewDbDataType("tgt_longtext", DTString).WithCT(CTLongtext) }
func (c *testConverter) Bit(*Column) *DbDataType        { return NewDbDataType("tgt_bit", DTBit).WithCT(CTBit) }
func (c *testConverter) Int1(*Column) *DbDataType       { return NewDbDataType("tgt_int1", DTInt16).WithCT(CTInt1) }
func (c *testConverter) Int2(*Column) *DbDataType       { return NewDbDataType("tgt_int2", DTInt16).WithCT(CTInt2) }
func (c *testConverter) Int4(*Column) *DbDataType       { return NewDbDataType("tgt_int4", DTInt32).WithCT(CTInt4) }
func (c *testConverter) Int8(*Column) *DbDataType       { return NewDbDataType("tgt_int8", DTInt64).WithCT(CTInt8) }
func (c *testConverter) Numeric(*Column) *DbDataType    { return NewDbDataType("tgt_numeric", DTNumeric).WithCT(CTNumeric) }
func (c *testConverter) Decimal(*Column) *DbDataType    { return NewDbDataType("tgt_decimal", DTDecimal).WithCT(CTDecimal) }
func (c *testConverter) UnsignedInt8(*Column) *DbDataType  { return NewDbDataType("tgt_uint8", DTUint64).WithCT(CTUnsignedInt8) }
func (c *testConverter) UnsignedInt4(*Column) *DbDataType  { return NewDbDataType("tgt_uint4", DTUint64).WithCT(CTUnsignedInt4) }
func (c *testConverter) UnsignedInt2(*Column) *DbDataType  { return NewDbDataType("tgt_uint2", DTUint64).WithCT(CTUnsignedInt2) }
func (c *testConverter) UnsignedInt1(*Column) *DbDataType  { return NewDbDataType("tgt_uint1", DTUint64).WithCT(CTUnsignedInt1) }
func (c *testConverter) Date(*Column) *DbDataType          { return NewDbDataType("tgt_date", DTDate).WithCT(CTDate) }
func (c *testConverter) Time(*Column) *DbDataType          { return NewDbDataType("tgt_time", DTTime).WithCT(CTTime) }
func (c *testConverter) Datetime(*Column) *DbDataType      { return NewDbDataType("tgt_datetime", DTDateTime).WithCT(CTDateTime) }
func (c *testConverter) Timestamp(*Column) *DbDataType     { return NewDbDataType("tgt_timestamp", DTDateTime).WithCT(CTTimestamp) }
func (c *testConverter) Binary(*Column) *DbDataType        { return NewDbDataType("tgt_binary", DTBytes).WithCT(CTBinary) }
func (c *testConverter) Varbinary(*Column) *DbDataType     { return NewDbDataType("tgt_varbinary", DTBytes).WithCT(CTVarbinary) }
func (c *testConverter) Mediumblob(*Column) *DbDataType    { return NewDbDataType("tgt_mediumblob", DTBytes).WithCT(CTMediumblob) }
func (c *testConverter) Blob(*Column) *DbDataType          { return NewDbDataType("tgt_blob", DTBytes).WithCT(CTBlob) }
func (c *testConverter) Longblob(*Column) *DbDataType      { return NewDbDataType("tgt_longblob", DTBytes).WithCT(CTLongblob) }
func (c *testConverter) Enum(*Column) *DbDataType          { return NewDbDataType("tgt_enum", DTString).WithCT(CTEnum) }
func (c *testConverter) JSON(*Column) *DbDataType          { return NewDbDataType("tgt_json", DTString).WithCT(CTJSON) }

var _ CommonTypeConverter = (*testConverter)(nil)

// 测试专用dbType，避免污染真实方言注册
var (
	testSrcDbType  = DbType("test-src-db")
	testTgtDbType  = DbType("test-tgt-db")
	testSrcNoCT    = DbType("test-src-noct")
	testTgtPartial = DbType("test-tgt-partial")
)

func init() {
	// 源库：varchar/int8 均已声明公共类型
	registerColumnDbDataTypes(testSrcDbType,
		NewDbDataType("varchar", DTString).WithCT(CTVarchar),
		NewDbDataType("int8", DTInt64).WithCT(CTInt8),
	)
	registerCommonTypeConverter(testSrcDbType, &testConverter{})

	// 目标库
	registerColumnDbDataTypes(testTgtDbType,
		NewDbDataType("tgt_varchar", DTString).WithCT(CTVarchar),
		NewDbDataType("tgt_int4", DTInt32).WithCT(CTInt4),
	)
	registerCommonTypeConverter(testTgtDbType, &testConverter{})

	// 未声明公共类型的源库（模拟迁移兼容性缺失场景）
	registerColumnDbDataTypes(testSrcNoCT,
		NewDbDataType("custom_type", DTString), // 无WithCT
	)
	registerCommonTypeConverter(testSrcNoCT, &testConverter{})

	// 仅有CTVarchar转换器的目标库（模拟不完整的转换器）
	registerColumnDbDataTypes(testTgtPartial,
		NewDbDataType("tgt_varchar", DTString).WithCT(CTVarchar),
	)
	commonTypeConvertersMu.Lock()
	commonTypeConverters[testTgtPartial] = map[CommonDbDataType]func(*Column) *DbDataType{
		CTVarchar: (&testConverter{}).Varchar,
	}
	commonTypeConvertersMu.Unlock()
}

func TestConvToTargetDbColumn_SameDbType(t *testing.T) {
	column := &Column{DataType: "varchar", ColumnType: "varchar(2000)"}
	// 同类型数据库不做转换，Column保持原样
	assert.NoError(t, ConvToTargetDbColumn(testSrcDbType, testSrcDbType, newStubDialect(), column))
	assert.Equal(t, "varchar(2000)", column.ColumnType)
	assert.Equal(t, "varchar", column.DataType)
}

func TestConvToTargetDbColumn_NilDialect(t *testing.T) {
	column := &Column{DataType: "varchar"}
	err := ConvToTargetDbColumn(testSrcDbType, testTgtDbType, nil, column)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dialect")
}

func TestConvToTargetDbColumn_SrcNotSupport(t *testing.T) {
	column := &Column{DataType: "varchar"}
	err := ConvToTargetDbColumn(DbType("no-such-src"), testTgtDbType, newStubDialect(), column)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "src database type")
}

func TestConvToTargetDbColumn_TargetNotSupport(t *testing.T) {
	column := &Column{DataType: "varchar"}
	err := ConvToTargetDbColumn(testSrcDbType, DbType("no-such-tgt"), newStubDialect(), column)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target database type")
}

func TestConvToTargetDbColumn_UnknownCommonType(t *testing.T) {
	column := &Column{DataType: "custom_type"}
	err := ConvToTargetDbColumn(testSrcNoCT, testTgtDbType, newStubDialect(), column)
	// 未声明公共类型必须显式报错，禁止静默按varchar处理导致数据截断
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown common type")
}

func TestConvToTargetDbColumn_TargetMissConvertFunc(t *testing.T) {
	// 源为int8(CTInt8)，目标转换器仅注册了CTVarchar
	registerColumnDbDataTypes(DbType("test-src-int8"), NewDbDataType("int8", DTInt64).WithCT(CTInt8))
	registerCommonTypeConverter(DbType("test-src-int8"), &testConverter{})

	column := &Column{DataType: "int8"}
	err := ConvToTargetDbColumn(DbType("test-src-int8"), testTgtPartial, newStubDialect(), column)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not support transfer")
}

func TestConvToTargetDbColumn_ConvertFuncReturnNil(t *testing.T) {
	// 转换函数存在但返回nil，同样应报错
	returnNilTgt := DbType("test-tgt-nilfunc")
	registerColumnDbDataTypes(returnNilTgt, NewDbDataType("tgt_varchar", DTString).WithCT(CTVarchar))
	commonTypeConvertersMu.Lock()
	commonTypeConverters[returnNilTgt] = map[CommonDbDataType]func(*Column) *DbDataType{
		CTVarchar: func(*Column) *DbDataType { return nil },
	}
	commonTypeConvertersMu.Unlock()

	column := &Column{DataType: "varchar"}
	err := ConvToTargetDbColumn(testSrcDbType, returnNilTgt, newStubDialect(), column)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not support transfer")
}

func TestConvToTargetDbColumn_Success(t *testing.T) {
	column := &Column{
		DataType:      "varchar",
		ColumnType:    "varchar(2000)", // 源库查出的完整类型，转换前必须清空
		ColumnName:    "name",
		CharMaxLength: 2000,
	}
	err := ConvToTargetDbColumn(testSrcDbType, testTgtDbType, newStubDialect(), column)
	assert.NoError(t, err)

	// ColumnType被清空（防止跨库GetColumnType错误拼接）
	assert.Equal(t, "", column.ColumnType)
	// DataType被替换为目标库类型
	assert.Equal(t, "tgt_varchar", column.DataType)

	// 数值类型转换
	intColumn := &Column{DataType: "int8", ColumnType: "bigint"}
	err = ConvToTargetDbColumn(testSrcDbType, testTgtDbType, newStubDialect(), intColumn)
	assert.NoError(t, err)
	assert.Equal(t, "tgt_int8", intColumn.DataType)
	assert.Equal(t, "", intColumn.ColumnType)
}

func TestRegisterCommonTypeConverter_Nil(t *testing.T) {
	// nil转换器注册不应panic也不应生效
	before := getCommonTypeConverters(DbType("nil-ctc-db"))
	registerCommonTypeConverter(DbType("nil-ctc-db"), nil)
	assert.Nil(t, getCommonTypeConverters(DbType("nil-ctc-db")))
	assert.Equal(t, before, getCommonTypeConverters(DbType("nil-ctc-db")))
}

func TestRegisterCommonTypeConverter_AllCommonTypes(t *testing.T) {
	registerCommonTypeConverter(DbType("test-ctc-full"), &testConverter{})
	cts := getCommonTypeConverters(DbType("test-ctc-full"))
	assert.NotNil(t, cts)
	// 必须注册全部26个公共类型转换函数
	assert.Len(t, cts, 27)
	for ct, fn := range cts {
		assert.NotNil(t, fn, "common type [%d] convert func should not be nil", ct)
	}
}

// ConvToTargetDbColumn 错误信息应包含关键定位信息
func TestConvToTargetDbColumn_ErrorInfoContainsDbType(t *testing.T) {
	column := &Column{DataType: "custom_type"}
	err := ConvToTargetDbColumn(testSrcNoCT, testTgtDbType, newStubDialect(), column)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), string(testSrcNoCT)))
}
