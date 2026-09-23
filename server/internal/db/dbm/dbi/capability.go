package dbi

import "sort"

// MetadataFeature 方言的静态能力维度。
//
// 以「枚举 + 集合」而非「一组具名 bool 字段」表达：新增一种能力只需在此加一个常量
// （并由方言按需声明），既不改动能力结构体、也不新增任何 switch 分支——开闭原则。
type MetadataFeature uint8

const (
	// —— 核心静态能力：DefaultBackend 默认全支持，能力不足方言经 Without 删减 ——
	FeatSchemas MetadataFeature = iota
	FeatIndexes
	FeatForeignKeys
	FeatComments
	FeatDDLExport
	FeatGeneratedColumns
	FeatIdentityColumns
	FeatExpressionDefaults

	// —— 扩展对象能力：默认不支持，方言按「真实实现了对应内省接口」显式 With。
	//    与 ObjectKind 取值对齐（view/procedure/...），使能力清单可直接驱动前端资源树 ——
	FeatViews
	FeatProcedures
	FeatFunctions
	FeatSequences
	FeatTriggers
	// FeatTableRelations 表间外键「内省清单」能力（实现了 ForeignKeyProvider 才声明，供 ER 图/迁移拓扑）。
	// 与上方核心静态的 FeatForeignKeys 不同：后者表达「DDL 能否定义外键约束」（普遍为真），
	// 本能力表达「能否枚举某表的外键关系清单」（各方言实现不一），二者不可合并。
	FeatTableRelations
)

// allStaticFeatures 核心静态能力集合。DefaultBackend 默认视为「全支持」，能力不足的方言经 Without 删减。
// 注意：扩展对象能力（FeatViews 等）刻意不在此列——它们默认不支持，由实现了相应内省接口的方言显式 With。
var allStaticFeatures = []MetadataFeature{
	FeatSchemas, FeatIndexes, FeatForeignKeys, FeatComments,
	FeatDDLExport, FeatGeneratedColumns, FeatIdentityColumns, FeatExpressionDefaults,
}

// featureNames 能力枚举的稳定字符串标识，供能力协商端点、日志与字符串入口对齐。
// 扩展对象能力的命名与 ObjectKind 取值一致（view/procedure/...），前端可直接据此渲染对应树节点。
var featureNames = map[MetadataFeature]string{
	FeatSchemas:            "schemas",
	FeatIndexes:            "indexes",
	FeatForeignKeys:        "foreign_keys",
	FeatComments:           "comments",
	FeatDDLExport:          "ddl_export",
	FeatGeneratedColumns:   "generated_columns",
	FeatIdentityColumns:    "identity_columns",
	FeatExpressionDefaults: "expression_default",
	FeatViews:              string(KindView),
	FeatProcedures:         string(KindProcedure),
	FeatFunctions:          string(KindFunction),
	FeatSequences:          string(KindSequence),
	FeatTriggers:           string(KindTrigger),
	FeatTableRelations:     "relation",
}

// featureByName 由 featureNames 反向构建，避免手写两份映射而漂移。
var featureByName = func() map[string]MetadataFeature {
	m := make(map[string]MetadataFeature, len(featureNames))
	for f, n := range featureNames {
		m[n] = f
	}
	return m
}()

// ObjectKindFeatures 扩展「对象类别能力」→ 对应 ObjectKind 的单一映射，供能力协商与
// 「声明⟺实现」护栏数据驱动校验：方言声明了某对象能力，其 MetadataNavigator.SupportedKinds()
// 必须包含对应 kind，否则即为谎报。新增对象类别只需在此加一行，护栏与端点零改动。
// 注意：FeatTableRelations 不在其中——它对应 ForeignKeyProvider 接口，而非 MetadataNavigator 的某个 kind。
var ObjectKindFeatures = map[MetadataFeature]ObjectKind{
	FeatViews:      KindView,
	FeatProcedures: KindProcedure,
	FeatFunctions:  KindFunction,
	FeatSequences:  KindSequence,
	FeatTriggers:   KindTrigger,
}

// NamespaceHierarchy 数据库命名空间层次结构声明（与 MetadataCapabilities 同属方言能力声明体系）。
// 不同数据库的命名空间层次差异很大：
//   - MySQL：database = schema（一层，HasDatabase=true, HasSchema=false）
//   - PostgreSQL：database > schema（两层）
//   - SQL Server：server > database > schema（三层）
//   - Oracle：container > pdb > schema（三层）
//   - SQLite：无 schema 概念（一层，HasDatabase=true）
type NamespaceHierarchy struct {
	HasDatabase bool // 是否有 database 层
	HasSchema   bool // 是否有 schema 层（独立于 database）
	HasCatalog  bool // 是否有 catalog 层（如 SQL Server 的 server 级）
}

// MetadataCapabilities 方言元数据能力声明：静态能力集合 + 命名空间层次。
//
// features 为私有集合，只能经 NewCapabilities/NewAllCapabilities/Without 构造，
// 避免运行期被随意改写；读取统一走 Has/HasStaticFeature。
type MetadataCapabilities struct {
	features map[MetadataFeature]bool

	NamespaceHierarchy NamespaceHierarchy
}

// NewCapabilities 以「支持的能力列表」构造：列出的即支持，未列即不支持。
func NewCapabilities(fs ...MetadataFeature) MetadataCapabilities {
	m := make(map[MetadataFeature]bool, len(fs))
	for _, f := range fs {
		m[f] = true
	}
	return MetadataCapabilities{features: m}
}

// NewAllCapabilities 构造「全部静态能力均支持」的声明，供全功能方言基类，再按需 Without 删减。
func NewAllCapabilities() MetadataCapabilities {
	return NewCapabilities(allStaticFeatures...)
}

// Without 返回移除指定能力后的副本（表达「默认全支持、减去不支持项」）。
func (c MetadataCapabilities) Without(fs ...MetadataFeature) MetadataCapabilities {
	m := make(map[MetadataFeature]bool, len(c.features))
	for f, v := range c.features {
		m[f] = v
	}
	for _, f := range fs {
		m[f] = false
	}
	return MetadataCapabilities{features: m, NamespaceHierarchy: c.NamespaceHierarchy}
}

// With 返回追加指定能力后的副本（方言声明「我真实支持这些扩展对象能力」，须已实现对应内省接口）。
// 与 Without 对称：均返回新值不改动接收者，零值（features==nil）亦安全。
func (c MetadataCapabilities) With(fs ...MetadataFeature) MetadataCapabilities {
	m := make(map[MetadataFeature]bool, len(c.features)+len(fs))
	for f, v := range c.features {
		m[f] = v
	}
	for _, f := range fs {
		m[f] = true
	}
	return MetadataCapabilities{features: m, NamespaceHierarchy: c.NamespaceHierarchy}
}

// WithNamespace 返回补充命名空间层次声明后的副本。
//
// 与 With/Without 同为链式建造者：能力集合与命名空间层次都在构造链上一并声明，
// 避免「用 NewAllCapabilities 构造完再单独给公开字段赋值」这一容易被 override 遗漏的写法
// （遗漏会让方言静默退化为「无 database 无 schema」的单层，且被能力协商端点直接暴露给前端）。
func (c MetadataCapabilities) WithNamespace(h NamespaceHierarchy) MetadataCapabilities {
	c.NamespaceHierarchy = h
	return c
}

// Has 报告是否支持某静态能力。零值（features==nil）安全返回 false。
func (c MetadataCapabilities) Has(f MetadataFeature) bool { return c.features[f] }

// HasStaticFeature 字符串能力名入口（供外部按名查询），映射到枚举后判定；未知名返回 false。
//
// 经 featureByName 数据表映射判定：新增能力只需在 featureNames 加一行，不触碰本函数逻辑（开闭）。
func (c MetadataCapabilities) HasStaticFeature(name string) bool {
	f, ok := featureByName[name]
	if !ok {
		return false
	}
	return c.Has(f)
}

// SupportedFeatures 返回当前支持的能力稳定字符串列表（升序），供能力协商端点与诊断暴露。
func (c MetadataCapabilities) SupportedFeatures() []string {
	names := make([]string, 0, len(c.features))
	for f, ok := range c.features {
		if !ok {
			continue
		}
		if n, exists := featureNames[f]; exists {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}
