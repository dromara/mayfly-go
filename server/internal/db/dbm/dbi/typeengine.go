package dbi

import (
	"fmt"
	"strings"
)

// TypeRule 类型转换规则：给定列信息，返回目标方言的 DbDataType。
type TypeRule func(col *Column) *DbDataType

// TypeEngine 类型引擎 — 方言类型系统的单一注册中心。
// 通过「规则 → 全局降级」两级查找实现开闭原则：新增类别时各方言规则表无需修改，
// 未显式注册规则的类别自动落到全局降级安全网（globalFallbackTypes）。
type TypeEngine struct {
	// 方言类型标识
	dbType DbType

	// 源类型注册表：type_name（原始大小写） → 该方言的列类型描述符
	types map[string]*DbDataType

	// 小写索引：lower(type_name) → 列类型描述符。与 types 同步维护，
	// 供 ResolveType 做 O(1) 大小写不敏感查找（database/sql 的 DatabaseTypeName
	// 常为大写，而注册名多为小写，热路径上避免逐列全表扫描 + ToLower）
	lowerIndex map[string]*DbDataType

	// 注册顺序快照：与 types 同步追加，供降级解析按「先注册者」确定性取类型
	order []*DbDataType

	// 数组类型名前缀与其类型原型（见 RegisterArrayType），空前缀表示该方言无数组前缀约定
	arrayTypePrefix string
	arrayTypeProto  *DbDataType

	// 转换规则：TypeCategory → TypeRule（该方言如何把某类别落为其一个具体列类型）
	rules map[TypeCategory]TypeRule
}

// ResolveType 按类型名查找源类型注册表（不区分大小写），未命中返回 (nil, false)。
func (e *TypeEngine) ResolveType(typeName string) (*DbDataType, bool) {
	if e == nil {
		return nil, false
	}
	// 优先精确匹配
	if dt, ok := e.types[typeName]; ok {
		return dt, true
	}
	// 大小写不敏感回退：走预建小写索引，避免热路径上的 O(n) 线性扫描
	if dt, ok := e.lowerIndex[strings.ToLower(typeName)]; ok {
		return dt, true
	}
	return e.resolveArrayType(typeName)
}

// resolveArrayType 按方言声明的数组前缀解析类型名（如 pg 的 bigint[] 元数据名为 _int8）。
//
// 只有元素类型本身已在注册表内才算认得：自定义类型（枚举/域）的数组仍返回未命中，
// 让调用方如实标「认不出」，而不是猜一个类别。命中时返回原型副本并把类型名回填为
// 实际查询名，使日志/比对里看到的仍是库侧真实形态（_int8 而非 array）
func (e *TypeEngine) resolveArrayType(typeName string) (*DbDataType, bool) {
	if e == nil || e.arrayTypePrefix == "" || e.arrayTypeProto == nil {
		return nil, false
	}
	prefixLen := len(e.arrayTypePrefix)
	if !strings.HasPrefix(typeName, e.arrayTypePrefix) || len(typeName) <= prefixLen {
		return nil, false
	}
	element := typeName[prefixLen:]
	if _, ok := e.ResolveType(element); !ok {
		return nil, false
	}
	arrayType := *e.arrayTypeProto
	arrayType.Name = typeName
	return &arrayType, true
}

// ResolveTarget 解析目标类型：先查规则表，未命中则查全局降级表（所有引擎共享的安全网）。
// 降级表中的类型按 TypeCategory 匹配目标引擎注册表中的等价类型，
// 确保返回的 DbDataType 在目标引擎中有正确的 SQLValue/Valuer。
func (e *TypeEngine) ResolveTarget(category TypeCategory, col *Column) (*DbDataType, error) {
	// 1. 查规则表
	if rule, ok := e.rules[category]; ok && rule != nil {
		result := rule(col)
		if result != nil {
			return result, nil
		}
	}

	// 2. 查全局降级表（所有引擎共享的安全网）
	if gfb, ok := globalFallbackTypes[category]; ok && gfb != nil {
		// 先按类型名精确命中目标方言的同名类型（降级为 varchar 时目标就该是 varchar），
		// 名字不命中才按类别取先注册者
		if dt, found := e.ResolveType(gfb.Name); found && dt.Category() == gfb.Category() {
			return dt, nil
		}
		if dt, found := e.findByCategory(gfb.Category()); found {
			return dt, nil
		}
	}

	return nil, fmt.Errorf("type category [%d] not supported: no rule or fallback registered", category)
}

// findByCategory 按 TypeCategory 查找第一个匹配的类型（注册顺序优先）。
// 用于降级解析：确保返回的 DbDataType 拥有本方言专有的 SQLValue/Valuer。
//
// 必须按 order 而非遍历 types：同一类别下往往注册了多个类型（mssql 的 varchar/nvarchar/
// char/uniqueidentifier/geometry 都属 TCVarchar），Go 的 map 迭代顺序随机，
// 同一列两次迁移会落到不同目标类型。
func (e *TypeEngine) findByCategory(category TypeCategory) (*DbDataType, bool) {
	for _, dt := range e.order {
		if dt.Category() == category {
			return dt, true
		}
	}
	return nil, false
}

// Types 返回源类型注册表的快照（供测试验证用）
func (e *TypeEngine) Types() map[string]*DbDataType {
	if e == nil {
		return nil
	}
	cp := make(map[string]*DbDataType, len(e.types))
	for k, v := range e.types {
		cp[k] = v
	}
	return cp
}

// ---------------- Builder ----------------

// TypeEngineBuilder 类型引擎构建器，提供方便的注册 API。
type TypeEngineBuilder struct {
	engine *TypeEngine
}

// RegisterTypes 批量注册源类型到 TypeEngine（单一数据源）。
// 同步维护小写索引与注册顺序；同名（仅大小写不同）以先注册者为准，保证查找结果确定。
func (b *TypeEngineBuilder) RegisterTypes(types ...*DbDataType) {
	for _, dt := range types {
		if dt == nil {
			continue
		}
		if _, exists := b.engine.types[dt.Name]; exists {
			continue
		}
		b.engine.types[dt.Name] = dt
		b.engine.order = append(b.engine.order, dt)
		lower := strings.ToLower(dt.Name)
		if _, ok := b.engine.lowerIndex[lower]; !ok {
			b.engine.lowerIndex[lower] = dt
		}
	}
}

// RegisterArrayType 声明该方言元数据里数组类型名的固定前缀，以及前缀命中时使用的类型原型。
//
// 原型只需给出类别与值编解码（其 Name 被忽略，解析时回填为库侧真实类型名）：
// 数组的元素类型已在注册表内，逐个登记 `_int8`/`_text`/`_uuid`… 既是第二套真相也必然漏项。
func (b *TypeEngineBuilder) RegisterArrayType(prefix string, proto *DbDataType) {
	if prefix == "" || proto == nil {
		return
	}
	b.engine.arrayTypePrefix = prefix
	b.engine.arrayTypeProto = proto
}

// RegisterRule 注册指定类别的转换规则。
func (b *TypeEngineBuilder) RegisterRule(category TypeCategory, rule TypeRule) {
	b.engine.rules[category] = rule
}

// RegisterRules 批量注册：多个 TypeCategory 共享同一个目标类型（简单 1:1 映射）。
func (b *TypeEngineBuilder) RegisterRules(targetType *DbDataType, categories ...TypeCategory) {
	for _, cat := range categories {
		b.engine.rules[cat] = func(col *Column) *DbDataType { return targetType }
	}
}
