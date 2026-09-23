package dbi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TCUnknown 必须为零值：未 WithCategory 的类型默认即为 TCUnknown，迁移时须显式报错
func TestTCUnknownIsZeroValue(t *testing.T) {
	var zero TypeCategory
	assert.Equal(t, zero, TCUnknown)
	assert.Equal(t, TypeCategory(0), TCUnknown)

	// 未调用WithCategory的类型Category()即为TCUnknown
	assert.Equal(t, TCUnknown, NewDbDataType("unknown-type", DTString).Category())
}

// 测试专用dbType，避免污染真实方言注册
var (
	testSrcDbType       = DbType("test-src-db")
	testTgtDbType       = DbType("test-tgt-db")
	testSrcNoCT         = DbType("test-src-noct")
	testTgtPartial      = DbType("test-tgt-partial")
	testSrcDatetimeType = DbType("test-src-datetime")
)

// resolveAll 辅助：将源引擎的所有类型按 Category 注册到目标引擎的规则中
func resolveAll(types ...*DbDataType) func(*TypeEngineBuilder) {
	return func(b *TypeEngineBuilder) {
		b.RegisterTypes(types...)
		for _, dt := range types {
			if dt.Category() != TCUnknown {
				cat := dt.Category()
				b.RegisterRule(cat, func(col *Column) *DbDataType { return dt })
			}
		}
	}
}

func init() {
	// 源库：varchar/int8 均已声明类型类别
	RegisterTypeEngine(testSrcDbType, resolveAll(
		NewDbDataType("varchar", DTString).WithCategory(TCVarchar),
		NewDbDataType("int8", DTInt64).WithCategory(TCInt8),
	))

	// 时间类源列
	RegisterTypeEngine(testSrcDatetimeType, resolveAll(
		NewDbDataType("datetime", DTDateTime).WithCategory(TCDateTime),
	))

	// 目标库
	RegisterTypeEngine(testTgtDbType, resolveAll(
		NewDbDataType("tgt_varchar", DTString).WithCategory(TCVarchar),
		NewDbDataType("tgt_int4", DTInt32).WithCategory(TCInt4),
		NewDbDataType("tgt_int8", DTInt64).WithCategory(TCInt8),
		NewDbDataType("tgt_bool", DTBit).WithCategory(TCBool),
		NewDbDataType("tgt_bit", DTBit).WithCategory(TCBit),
		NewDbDataType("tgt_datetime", DTDateTime).WithCategory(TCDateTime),
		NewDbDataType("tgt_date", DTDate).WithCategory(TCDate),
	))

	// 未声明类型类别的源库
	RegisterTypeEngine(testSrcNoCT, func(b *TypeEngineBuilder) {
		b.RegisterTypes(NewDbDataType("custom_type", DTString)) // 无WithCategory
	})

	// 仅有TCVarchar规则的目标库
	RegisterTypeEngine(testTgtPartial, func(b *TypeEngineBuilder) {
		dt := NewDbDataType("tgt_varchar", DTString).WithCategory(TCVarchar)
		b.RegisterTypes(dt)
		b.RegisterRule(TCVarchar, func(col *Column) *DbDataType { return dt })
	})
}

func TestConvToTargetDbColumn_SameDbType(t *testing.T) {
	column := &Column{DataType: "varchar", ColumnType: "varchar(2000)"}
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

func TestConvToTargetDbColumn_UnknownCategory(t *testing.T) {
	column := &Column{DataType: "custom_type"}
	err := ConvToTargetDbColumn(testSrcNoCT, testTgtDbType, newStubDialect(), column)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type category")
}

func TestConvToTargetDbColumn_TargetNoRule(t *testing.T) {
	// 源为int8(TCInt8)，目标仅注册了TCVarchar
	srcInt8 := DbType("test-src-int8-te")
	RegisterTypeEngine(srcInt8, resolveAll(
		NewDbDataType("int8", DTInt64).WithCategory(TCInt8),
	))

	column := &Column{DataType: "int8"}
	err := ConvToTargetDbColumn(srcInt8, testTgtPartial, newStubDialect(), column)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not support transfer")
}

func TestConvToTargetDbColumn_Success(t *testing.T) {
	column := &Column{
		DataType:      "varchar",
		ColumnType:    "varchar(2000)",
		ColumnName:    "name",
		CharMaxLength: 2000,
	}
	err := ConvToTargetDbColumn(testSrcDbType, testTgtDbType, newStubDialect(), column)
	assert.NoError(t, err)
	assert.Equal(t, "", column.ColumnType)
	assert.Equal(t, "tgt_varchar", column.DataType)

	// 数值类型转换
	intColumn := &Column{DataType: "int8", ColumnType: "bigint"}
	err = ConvToTargetDbColumn(testSrcDbType, testTgtDbType, newStubDialect(), intColumn)
	assert.NoError(t, err)
	assert.Equal(t, "tgt_int8", intColumn.DataType)
	assert.Equal(t, "", intColumn.ColumnType)
}

func TestConvToTargetDbColumn_StringTargetClearsNumericLength(t *testing.T) {
	column := &Column{DataType: "varchar", NumPrecision: 10, NumScale: 2, CharMaxLength: 0}
	require.NoError(t, ConvToTargetDbColumn(testSrcDbType, testTgtDbType, newStubDialect(), column))
	assert.Equal(t, "tgt_varchar", column.DataType)
	assert.Equal(t, 0, column.NumPrecision)
	assert.Equal(t, 0, column.NumScale)

	// 非字符串目标类型不受字符串清理影响
	timeColumn := &Column{DataType: "datetime", NumPrecision: 3, NumScale: 2}
	require.NoError(t, ConvToTargetDbColumn(testSrcDatetimeType, testTgtDbType, newStubDialect(), timeColumn))
	assert.Equal(t, "tgt_datetime", timeColumn.DataType)
	assert.Equal(t, 3, timeColumn.NumPrecision, "时间列的小数秒精度不得被误清除")
	assert.Equal(t, 0, timeColumn.NumScale)
}

func TestConvToTargetDbColumn_ClearSpuriousIntPrecision(t *testing.T) {
	for _, src := range []struct {
		name string
		cat  TypeCategory
	}{
		{"int8", TCInt8}, {"int4", TCInt4}, {"bool", TCBool}, {"date", TCDate}, {"unsigned_int1", TCUnsignedInt1},
	} {
		srcDb := DbType("test-src-te-" + src.name)
		RegisterTypeEngine(srcDb, resolveAll(
			NewDbDataType(src.name, DTInt64).WithCategory(src.cat),
		))

		column := &Column{DataType: src.name, NumPrecision: 32, NumScale: 2}
		require.NoError(t, ConvToTargetDbColumn(srcDb, testTgtDbType, newStubDialect(), column))
		assert.Equal(t, 0, column.NumPrecision, "%s 的残留精度未清空", src.name)
		assert.Equal(t, 0, column.NumScale, "%s 的残留小数位未清空", src.name)
	}
}

func TestConvToTargetDbColumn_Bool(t *testing.T) {
	srcBool := DbType("test-src-bool-te")
	RegisterTypeEngine(srcBool, resolveAll(
		NewDbDataType("bool", DTBit).WithCategory(TCBool),
	))

	column := &Column{DataType: "bool", ColumnType: "boolean"}
	err := ConvToTargetDbColumn(srcBool, testTgtDbType, newStubDialect(), column)
	assert.NoError(t, err)
	assert.Equal(t, "tgt_bool", column.DataType)
	assert.Equal(t, "", column.ColumnType)
}

func TestConvToTargetDbColumn_Bit(t *testing.T) {
	srcBit := DbType("test-src-bit-te")
	RegisterTypeEngine(srcBit, resolveAll(
		NewDbDataType("bit", DTBit).WithCategory(TCBit),
	))

	column := &Column{DataType: "bit", ColumnType: "bit(8)"}
	err := ConvToTargetDbColumn(srcBit, testTgtDbType, newStubDialect(), column)
	assert.NoError(t, err)
	assert.Equal(t, "tgt_bit", column.DataType)
}

func TestConvToTargetDbColumn_ErrorInfoContainsDbType(t *testing.T) {
	column := &Column{DataType: "custom_type"}
	err := ConvToTargetDbColumn(testSrcNoCT, testTgtDbType, newStubDialect(), column)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), string(testSrcNoCT)))
}

// TestTypeEngine_GlobalFallback 验证全局降级策略：未注册规则的类别通过全局降级解析
func TestTypeEngine_GlobalFallback(t *testing.T) {
	// 注册一个只有源类型没有转换规则的目标引擎
	fallbackTgt := DbType("test-tgt-fallback")
	RegisterTypeEngine(fallbackTgt, func(b *TypeEngineBuilder) {
		// 注册一个 TCVarchar 类别的类型
		vc := NewDbDataType("fb_varchar", DTString).WithCategory(TCVarchar)
		b.RegisterTypes(vc)
		// 不注册任何规则，依赖全局降级
	})

	srcFB := DbType("test-src-fallback")
	RegisterTypeEngine(srcFB, resolveAll(
		NewDbDataType("uuid_col", DTString).WithCategory(TCUUID),
	))

	column := &Column{DataType: "uuid_col", CharMaxLength: 36}
	err := ConvToTargetDbColumn(srcFB, fallbackTgt, newStubDialect(), column)
	// TCUUID 全局降级为 TCVarchar，目标引擎有 fb_varchar(TCVarchar)，应成功
	assert.NoError(t, err)
	assert.Equal(t, "fb_varchar", column.DataType)
}
