package dbi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTypeEngineArrayPrefix 数组类型名的前缀解析守卫
//
// 存在理由：pg 元数据里数组列的类型名是 `_` + 元素类型名（bigint[] → _int8），逐元素类型登记数组名
// 既是第二套真相也必然漏项。前缀解析必须只认「元素类型已在注册表内」的数组——自定义类型（枚举/域）
// 的数组仍算认不出，宁可让调用方如实标「认不出」也不猜类别。
func TestTypeEngineArrayPrefix(t *testing.T) {
	dt := DbType("test-array-prefix")
	RegisterTypeEngine(dt, func(b *TypeEngineBuilder) {
		b.RegisterTypes(NewDbDataType("int8", DTInt64).WithCategory(TCInt8))
		b.RegisterArrayType("_", NewDbDataType("array", DTString).WithCategory(TCArray))
	})
	engine := GetTypeEngine(dt)

	arr, ok := engine.ResolveType("_int8")
	require.True(t, ok, "元素类型已注册时数组名必须可解析")
	assert.Equal(t, TCArray, arr.Category())
	// 类型名回填为库侧真实形态：比对结果与日志里不得出现注册用的占位名 array
	assert.Equal(t, "_int8", arr.Name)

	// 数组解析不得吞掉元素类型自身的归类
	elem, ok := engine.ResolveType("int8")
	require.True(t, ok)
	assert.Equal(t, TCInt8, elem.Category())

	// 元素类型未注册（枚举/域等自定义类型）时仍算认不出
	_, ok = engine.ResolveType("_myenum")
	assert.False(t, ok, "元素类型未注册时不得猜数组类别")

	// 只有前缀没有元素名，不是数组
	_, ok = engine.ResolveType("_")
	assert.False(t, ok)

	// 未声明前缀的方言不会把下划线开头的名字当数组
	plain := DbType("test-array-prefix-none")
	RegisterTypeEngine(plain, func(b *TypeEngineBuilder) {
		b.RegisterTypes(NewDbDataType("int8", DTInt64).WithCategory(TCInt8))
	})
	_, ok = GetTypeEngine(plain).ResolveType("_int8")
	assert.False(t, ok, "无数组前缀约定的方言不得按前缀合成类型")
}

// TestTypeEngineFallbackIsDeterministic 降级目标类型的确定性守卫
//
// 存在理由：同一类别下往往注册了多个类型（mssql 属 TCVarchar 的就有 varchar/nvarchar/char/
// uniqueidentifier/geometry…十来个）。降级解析若遍历 map 取第一个命中，Go 的 map 迭代顺序随机，
// 同一列两次迁移会落到不同目标类型（一次 varchar、一次 geometry），迁移产物不可复现。
func TestTypeEngineFallbackIsDeterministic(t *testing.T) {
	named := DbType("test-fallback-named")
	RegisterTypeEngine(named, func(b *TypeEngineBuilder) {
		b.RegisterTypes(
			// 先注册的是该类别的"最后一个"候选，用于证明按名命中优先于注册顺序
			NewDbDataType("geometry", DTString).WithCategory(TCVarchar),
			NewDbDataType("varchar", DTString).WithCategory(TCVarchar),
			NewDbDataType("nvarchar", DTString).WithCategory(TCVarchar),
		)
	})
	engine := GetTypeEngine(named)

	// TCUUID 的全局降级类型名为 varchar：目标有同名同类别类型时必须就是 varchar
	for i := 0; i < 50; i++ {
		got, err := engine.ResolveTarget(TCUUID, &Column{})
		require.NoError(t, err)
		assert.Equal(t, "varchar", got.Name, "第%d次降级结果漂移", i)
	}

	// 目标没有同名类型时按注册顺序取该类别第一个（此处 geometry 先注册）
	byOrder := DbType("test-fallback-order")
	RegisterTypeEngine(byOrder, func(b *TypeEngineBuilder) {
		b.RegisterTypes(
			NewDbDataType("zeta", DTBytes).WithCategory(TCBlob),
			NewDbDataType("alpha", DTBytes).WithCategory(TCBlob),
		)
	})
	engine = GetTypeEngine(byOrder)
	for i := 0; i < 50; i++ {
		got, err := engine.ResolveTarget(TCBlob, &Column{})
		require.NoError(t, err)
		assert.Equal(t, "zeta", got.Name, "按名不命中时应取先注册者，且结果必须稳定")
	}
}

// TestTypeEngineRegisterTypesKeepsOrderSnapshot 注册顺序快照与注册表的一致性守卫
//
// 同名类型重复注册时注册表以先注册者为准，顺序快照也必须同步跳过，否则按类别取"先注册者"
// 会漂到后注册的同名对象上，拿到的值编解码与按名查找的结果不一致。
func TestTypeEngineRegisterTypesKeepsOrderSnapshot(t *testing.T) {
	dt := DbType("test-register-dup")
	second := NewDbDataType("dup", DTString).WithCategory(TCBlob)
	RegisterTypeEngine(dt, func(b *TypeEngineBuilder) {
		b.RegisterTypes(NewDbDataType("dup", DTBytes).WithCategory(TCBlob))
		b.RegisterTypes(second)
	})
	engine := GetTypeEngine(dt)

	byName, ok := engine.ResolveType("dup")
	require.True(t, ok)
	byCategory, ok := engine.findByCategory(TCBlob)
	require.True(t, ok)
	assert.Same(t, byName, byCategory, "按名与按类别解析必须命中同一个注册对象")
	assert.NotSame(t, second, byName, "同名以先注册者为准")
}
