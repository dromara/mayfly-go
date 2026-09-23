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
	return nil, false
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
		if dt, found := e.findByCategory(gfb.Category()); found {
			return dt, nil
		}
	}

	return nil, fmt.Errorf("type category [%d] not supported: no rule or fallback registered", category)
}

// findByCategory 在注册表中按 TypeCategory 查找第一个匹配的类型。
// 用于降级解析：确保返回的 DbDataType 拥有本方言专有的 SQLValue/Valuer。
func (e *TypeEngine) findByCategory(category TypeCategory) (*DbDataType, bool) {
	for _, dt := range e.types {
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
// 同步维护小写索引；同名（仅大小写不同）以先注册者为准，保证查找结果确定。
func (b *TypeEngineBuilder) RegisterTypes(types ...*DbDataType) {
	for _, dt := range types {
		if dt == nil {
			continue
		}
		b.engine.types[dt.Name] = dt
		lower := strings.ToLower(dt.Name)
		if _, ok := b.engine.lowerIndex[lower]; !ok {
			b.engine.lowerIndex[lower] = dt
		}
	}
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
