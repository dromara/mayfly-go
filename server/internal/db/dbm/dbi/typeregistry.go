package dbi

import "sync"

// ---------------- 全局类型引擎注册表 ----------------

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
		dbType:     dbType,
		types:      make(map[string]*DbDataType),
		lowerIndex: make(map[string]*DbDataType),
		rules:      make(map[TypeCategory]TypeRule),
	}
	b := &TypeEngineBuilder{engine: e}
	builder(b)
	return e
}
