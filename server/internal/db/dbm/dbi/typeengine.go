package dbi

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// TypeCategory 类型类别 — 跨方言类型映射的枢轴标识。
// 替代原 CommonDbDataType，语义更清晰，且通过 TypeEngine 注册表机制支持无界扩展（开闭原则）：
// 新增类别只需设置全局降级策略，各方言无需修改。
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

// TypeRule 类型转换规则：给定列信息，返回目标方言的 DbDataType。
// 替代原 CommonTypeConverter 接口的单个方法。
type TypeRule func(col *Column) *DbDataType

// TypeEngine 类型引擎 — 方言类型系统的单一注册中心。
// 整合了原分散在 registerColumnDbDataTypes + registerCommonTypeConverter 的两套注册机制，
// 并通过「规则 → 降级」二级查找实现开闭原则：新增类别时各方言无需修改。
type TypeEngine struct {
	// 方言类型标识
	dbType DbType

	// 源类型注册表：type_name(小写) → *DbDataType
	// 替代原 dbDataTypes[dbType] map
	types map[string]*DbDataType

	// 转换规则：TypeCategory → TypeRule
	// 替代原 CommonTypeConverter 接口方法
	rules map[TypeCategory]TypeRule

	// 降级类型：TypeCategory → *DbDataType
	// 当规则表未覆盖某类别时，按类别从降级表取默认目标类型（OCP 的关键）
	fallbacks map[TypeCategory]*DbDataType
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
	// 不区分大小写回退
	lower := strings.ToLower(typeName)
	for name, dt := range e.types {
		if strings.ToLower(name) == lower {
			return dt, true
		}
	}
	return nil, false
}

// ResolveTarget 解析目标类型：先查规则表，未命中则查降级表（本引擎 → 全局降级）。
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

	// 2. 查本引擎的降级表
	if fb, ok := e.fallbacks[category]; ok && fb != nil {
		if dt, found := e.findByCategory(fb.Category()); found {
			return dt, nil
		}
	}

	// 3. 查全局降级表（所有引擎共享的安全网）
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

// HasRule 是否注册了指定类别的转换规则（供测试验证用）
func (e *TypeEngine) HasRule(category TypeCategory) bool {
	if e == nil {
		return false
	}
	_, ok := e.rules[category]
	return ok
}

// Categories 返回引擎已注册规则的所有 TypeCategory（排序）
func (e *TypeEngine) Categories() []TypeCategory {
	if e == nil {
		return nil
	}
	cats := make([]TypeCategory, 0, len(e.rules)+len(e.fallbacks))
	seen := make(map[TypeCategory]bool)
	for c := range e.rules {
		if !seen[c] {
			cats = append(cats, c)
			seen[c] = true
		}
	}
	for c := range e.fallbacks {
		if !seen[c] {
			cats = append(cats, c)
			seen[c] = true
		}
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i] < cats[j] })
	return cats
}

// HasType 注册表是否包含指定名称的类型（不区分大小写）
func (e *TypeEngine) HasType(name string) bool {
	if e == nil {
		return false
	}
	_, ok := e.types[name]
	return ok
}

// RuleCount 已注册的规则数量
func (e *TypeEngine) RuleCount() int {
	if e == nil {
		return 0
	}
	return len(e.rules)
}

// AllRegisteredTypeNames 返回所有已注册源类型的名称列表（排序）
func (e *TypeEngine) AllRegisteredTypeNames() []string {
	if e == nil {
		return nil
	}
	names := make([]string, 0, len(e.types))
	for name := range e.types {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// CanResolve 目标引擎是否能解析指定的 TypeCategory（规则表或降级表命中即返回 true）
func (e *TypeEngine) CanResolve(category TypeCategory) bool {
	if e == nil {
		return false
	}
	// 1. 规则表
	if rule, ok := e.rules[category]; ok && rule != nil {
		return true
	}
	// 2. 本引擎降级表
	if fb, ok := e.fallbacks[category]; ok && fb != nil {
		if _, found := e.findByCategory(fb.Category()); found {
			return true
		}
	}
	// 3. 全局降级表
	if gfb, ok := globalFallbackTypes[category]; ok && gfb != nil {
		if _, found := e.findByCategory(gfb.Category()); found {
			return true
		}
	}
	return false
}

// ResolveTargetDryRun 预验证：返回目标类型但不修改 Column（与 ResolveTarget 相同查找逻辑）
func (e *TypeEngine) ResolveTargetDryRun(category TypeCategory, col *Column) (*DbDataType, error) {
	return e.ResolveTarget(category, col)
}

// TypeEngineDiagnostics 类型引擎运行时诊断信息（只读，用于运维可见性）
type TypeEngineDiagnostics struct {
	DbType          DbType                  `json:"dbType"`          // 方言类型
	TypeCount       int                     `json:"typeCount"`       // 已注册源类型数量
	RuleCount       int                     `json:"ruleCount"`       // 已注册转换规则数量
	FallbackCount   int                     `json:"fallbackCount"`   // 已注册降级类型数量
	Rules           []TypeCategory          `json:"rules"`           // 已注册规则的类别列表
	Fallbacks       []TypeCategory          `json:"fallbacks"`       // 已注册降级的类别列表
	GlobalFallbacks []TypeCategory          `json:"globalFallbacks"` // 依赖全局降级的类别列表
	Types           map[string]TypeCategory `json:"types"`           // 源类型名 → 类别映射
}

// Diagnostics 返回类型引擎的运行时诊断信息（只读快照）。
// 用于管理 API / 运维诊断：展示方言类型系统覆盖范围、规则完备性、全局降级依赖。
func (e *TypeEngine) Diagnostics() *TypeEngineDiagnostics {
	if e == nil {
		return nil
	}
	types := make(map[string]TypeCategory, len(e.types))
	for name, dt := range e.types {
		types[name] = dt.Category()
	}
	rules := make([]TypeCategory, 0, len(e.rules))
	for c := range e.rules {
		rules = append(rules, c)
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i] < rules[j] })

	fallbacks := make([]TypeCategory, 0, len(e.fallbacks))
	for c := range e.fallbacks {
		fallbacks = append(fallbacks, c)
	}
	sort.Slice(fallbacks, func(i, j int) bool { return fallbacks[i] < fallbacks[j] })

	// 计算依赖全局降级的类别：所有类别中本引擎既无规则又无自身降级的
	var globalFallbacks []TypeCategory
	for _, cat := range allTypeCategories {
		if _, hasRule := e.rules[cat]; hasRule {
			continue
		}
		if _, hasFallback := e.fallbacks[cat]; hasFallback {
			continue
		}
		if _, hasGlobal := globalFallbackTypes[cat]; hasGlobal {
			globalFallbacks = append(globalFallbacks, cat)
		}
	}

	return &TypeEngineDiagnostics{
		DbType:          e.dbType,
		TypeCount:       len(e.types),
		RuleCount:       len(e.rules),
		FallbackCount:   len(e.fallbacks),
		Rules:           rules,
		Fallbacks:       fallbacks,
		GlobalFallbacks: globalFallbacks,
		Types:           types,
	}
}

// allTypeCategories 所有已定义的 TypeCategory（TCUnknown 除外），供测试与诊断使用
var allTypeCategories []TypeCategory

// categoryRegistryMu 保护动态类别注册
var categoryRegistryMu sync.Mutex

// nextCustomCategory 下一个可用的自定义类别编号（从 TCMoney+1 开始）
var nextCustomCategory TypeCategory

func init() {
	// 构建 allTypeCategories：从 TCVarchar(1) 到最后一个内置类别
	// TCUnknown(0) 排除，因为它表示「未分类」
	maxCat := TCMoney // 最后一个内置类别
	nextCustomCategory = maxCat + 1
	allTypeCategories = make([]TypeCategory, 0, int(maxCat))
	for c := TCUnknown + 1; c <= maxCat; c++ {
		allTypeCategories = append(allTypeCategories, c)
	}
}

// RegisterTypeCategory 动态注册新的 TypeCategory（开闭原则）。
// 允许第三方扩展在不修改 dbi 包常量块的前提下添加自定义类型类别。
// 返回分配的 TypeCategory 值。
func RegisterTypeCategory() TypeCategory {
	categoryRegistryMu.Lock()
	defer categoryRegistryMu.Unlock()
	cat := nextCustomCategory
	nextCustomCategory++
	allTypeCategories = append(allTypeCategories, cat)
	return cat
}

// AllTypeCategories 返回所有已定义的 TypeCategory（TCUnknown 除外），含动态注册的
func AllTypeCategories() []TypeCategory {
	cp := make([]TypeCategory, len(allTypeCategories))
	copy(cp, allTypeCategories)
	return cp
}

// ---------------- Builder ----------------

// TypeEngineBuilder 类型引擎构建器，提供方便的注册 API。
type TypeEngineBuilder struct {
	engine *TypeEngine
}

// RegisterTypes 批量注册源类型到 TypeEngine（单一数据源）。
func (b *TypeEngineBuilder) RegisterTypes(types ...*DbDataType) {
	for _, dt := range types {
		if dt != nil {
			b.engine.types[dt.Name] = dt
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

// RegisterSimpleRules 批量注册：多个 category→target 的 1:1 映射。
func (b *TypeEngineBuilder) RegisterSimpleRules(mappings map[TypeCategory]*DbDataType) {
	for cat, target := range mappings {
		b.engine.rules[cat] = func(col *Column) *DbDataType { return target }
	}
}

// RegisterFallback 注册指定类别的降级类型。
// 当规则表未覆盖某类别时，按 Category() 在本引擎注册表中匹配等价类型。
func (b *TypeEngineBuilder) RegisterFallback(category TypeCategory, fallbackType *DbDataType) {
	b.engine.fallbacks[category] = fallbackType
}

// ---------------- 全局注册表 ----------------

var (
	typeEnginesMu      sync.RWMutex
	typeEngines        = make(map[DbType]*TypeEngine)
	typeEngineBuilders = make(map[DbType]func(*TypeEngineBuilder))

	// 别名映射：别名 DbType → 主方言 DbType
	// 别名方言（如 mariadb/gauss/kingbaseEs/vastbase）复用主方言的类型引擎，
	// 避免重复注册相同构建器。GetTypeEngine 在找不到直接注册的 builder 时自动回退。
	typeEngineAliases = make(map[DbType]DbType)

	// 全局降级类型：TypeCategory → *DbDataType
	// 在 init() 中初始化，提供各类别的通用默认值（名称不必精确匹配方言注册表，
	// ResolveTarget 会按 Category 在目标引擎中查找等价类型）
	globalFallbackTypes map[TypeCategory]*DbDataType
)

func init() {
	// 全局降级类型 — 各类别的通用默认值
	// 这些类型的 Name 不必精确匹配目标方言注册表：ResolveTarget 会按 TypeCategory
	// 在目标引擎中查找等价类型，确保返回的 DbDataType 拥有方言专有的 SQLValue/Valuer
	globalFallbackTypes = map[TypeCategory]*DbDataType{
		// 字符串类
		TCVarchar:    NewDbDataType("varchar", DTString).WithCategory(TCVarchar),
		TCChar:       NewDbDataType("char", DTString).WithCategory(TCChar),
		TCText:       NewDbDataType("text", DTString).WithCategory(TCText),
		TCMediumtext: NewDbDataType("text", DTString).WithCategory(TCText),
		TCLongtext:   NewDbDataType("text", DTString).WithCategory(TCText),
		TCEnum:       NewDbDataType("varchar", DTString).WithCategory(TCVarchar),
		TCJSON:       NewDbDataType("text", DTString).WithCategory(TCText),
		TCUUID:       NewDbDataType("varchar", DTString).WithCategory(TCVarchar),

		// 整数类
		TCBit:  NewDbDataType("int", DTInt32).WithCategory(TCInt4),
		TCBool: NewDbDataType("int", DTInt32).WithCategory(TCInt4),
		TCInt1: NewDbDataType("int", DTInt32).WithCategory(TCInt4),
		TCInt2: NewDbDataType("int", DTInt32).WithCategory(TCInt4),
		TCInt4: NewDbDataType("int", DTInt32).WithCategory(TCInt4),
		TCInt8: NewDbDataType("bigint", DTInt64).WithCategory(TCInt8),

		// 无符号整数类
		TCUnsignedInt1: NewDbDataType("int", DTInt32).WithCategory(TCInt4),
		TCUnsignedInt2: NewDbDataType("int", DTInt32).WithCategory(TCInt4),
		TCUnsignedInt4: NewDbDataType("bigint", DTInt64).WithCategory(TCInt8),
		TCUnsignedInt8: NewDbDataType("bigint", DTInt64).WithCategory(TCInt8),

		// 数值类
		TCNumeric: NewDbDataType("double", DTNumeric).WithCategory(TCNumeric),
		TCDecimal: NewDbDataType("decimal", DTDecimal).WithCategory(TCDecimal),

		// 时间类
		TCDate:      NewDbDataType("date", DTDate).WithCategory(TCDate),
		TCTime:      NewDbDataType("time", DTTime).WithCategory(TCTime),
		TCDateTime:  NewDbDataType("datetime", DTDateTime).WithCategory(TCDateTime),
		TCTimestamp: NewDbDataType("datetime", DTDateTime).WithCategory(TCDateTime),

		// 二进制类
		TCBinary:     NewDbDataType("blob", DTBytes).WithCategory(TCBlob),
		TCVarbinary:  NewDbDataType("blob", DTBytes).WithCategory(TCBlob),
		TCBlob:       NewDbDataType("blob", DTBytes).WithCategory(TCBlob),
		TCMediumblob: NewDbDataType("blob", DTBytes).WithCategory(TCBlob),
		TCLongblob:   NewDbDataType("blob", DTBytes).WithCategory(TCBlob),

		// 扩展类别（全局降级安全网）
		TCInterval: NewDbDataType("varchar", DTString).WithCategory(TCVarchar),
		TCSpatial:  NewDbDataType("text", DTString).WithCategory(TCText),
		TCXML:      NewDbDataType("text", DTString).WithCategory(TCText),
		TCArray:    NewDbDataType("text", DTString).WithCategory(TCText),
		TCNetwork:  NewDbDataType("varchar", DTString).WithCategory(TCVarchar),
		TCRange:    NewDbDataType("varchar", DTString).WithCategory(TCVarchar),
		TCHStore:   NewDbDataType("text", DTString).WithCategory(TCText),
		TCMoney:    NewDbDataType("decimal", DTDecimal).WithCategory(TCDecimal),
	}
}

// RegisterTypeEngine 注册方言的类型引擎构建器（与 Backend 的 Register 平行但独立）。
// 仅支持在程序启动阶段（方言包 init 中）调用。
func RegisterTypeEngine(dbType DbType, builder func(*TypeEngineBuilder)) {
	typeEnginesMu.Lock()
	defer typeEnginesMu.Unlock()
	typeEngineBuilders[dbType] = builder
}

// RegisterTypeEngineAlias 注册别名方言到主方言的类型引擎映射。
// 别名方言（如 mariadb→mysql、gauss→postgres）复用主方言的类型系统，
// 避免重复注册相同构建器。应在 RegisterTypeEngine 之后调用。
func RegisterTypeEngineAlias(alias DbType, primary DbType) {
	typeEnginesMu.Lock()
	defer typeEnginesMu.Unlock()
	typeEngineAliases[alias] = primary
}

// GetTypeEngine 获取方言的类型引擎（首次调用时执行构建器，懒初始化）。
// 若该方言未直接注册构建器，则按别名映射回退到主方言的类型引擎。
func GetTypeEngine(dbType DbType) *TypeEngine {
	// 迭代解析别名链（防止循环别名），最多跳 4 层
	resolved := dbType
	for i := 0; i < 4; i++ {
		typeEnginesMu.RLock()
		if e, ok := typeEngines[resolved]; ok {
			typeEnginesMu.RUnlock()
			// 若经过了别名解析，缓存结果加速后续访问
			if resolved != dbType {
				typeEnginesMu.Lock()
				typeEngines[dbType] = e
				typeEnginesMu.Unlock()
			}
			return e
		}
		typeEnginesMu.RUnlock()

		typeEnginesMu.Lock()
		// 双重检查
		if e, ok := typeEngines[resolved]; ok {
			typeEnginesMu.Unlock()
			if resolved != dbType {
				typeEnginesMu.Lock()
				typeEngines[dbType] = e
				typeEnginesMu.Unlock()
			}
			return e
		}

		if builder, ok := typeEngineBuilders[resolved]; ok {
			e := buildTypeEngine(resolved, builder)
			typeEngines[resolved] = e
			typeEnginesMu.Unlock()
			if resolved != dbType {
				typeEnginesMu.Lock()
				typeEngines[dbType] = e
				typeEnginesMu.Unlock()
			}
			return e
		}

		// 无直接 builder，尝试别名回退
		if primary, aliased := typeEngineAliases[resolved]; aliased {
			typeEnginesMu.Unlock()
			resolved = primary
			continue
		}
		typeEnginesMu.Unlock()
		return nil
	}
	return nil
}

// buildTypeEngine 执行构建器创建 TypeEngine 实例（调用方持有锁）。
func buildTypeEngine(dbType DbType, builder func(*TypeEngineBuilder)) *TypeEngine {
	e := &TypeEngine{
		dbType:    dbType,
		types:     make(map[string]*DbDataType),
		rules:     make(map[TypeCategory]TypeRule),
		fallbacks: make(map[TypeCategory]*DbDataType),
	}
	b := &TypeEngineBuilder{engine: e}
	builder(b)
	return e
}

// ---------------- ConvToTargetDbColumn (通过 TypeEngine) ----------------

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

// cleanupColumnParams 按 TypeCategory 清理列的冗余参数。
// 替代原 ConvToTargetDbColumn 中的 switch 语句。
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
