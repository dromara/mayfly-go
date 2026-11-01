package dbi

import (
	"sort"

	"mayfly-go/internal/pkg/tlsx"
)

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
	// FeatSessions 会话内省与取消能力（实现了 SessionInspector 才声明）。
	// 与对象类能力不同：它不对应某个 ObjectKind，因此不参与 ObjectKindFeatures 映射，
	// 其「声明⟺实现」校验由 dbm/boundary_test.go 的能力探测表负责
	FeatSessions
	// FeatExplain 执行计划分析能力（实现了 ExplainAnalyzer 才声明）。
	// 同为非对象类能力，不参与 ObjectKindFeatures，由 boundary_test.go 的探测表将「声明」钉到「实现」
	FeatExplain
	// FeatTLS 连接加密能力：仅当方言真实支持至少一个 TLS 档位时声明，
	// 支持的档位由 MetadataCapabilities.TLSModes 列出（声明⟺档位非空，由 dbm 护栏测试把守）。
	//
	// 与其它扩展能力一样默认不支持：TLS 的证书传递方式在各方言差异极大
	// （内存 tls.Config / 只接受文件路径 / 仅客户端证书），默认声明为「支持」必然谎报
	FeatTLS
	// FeatSlowQuery 慢查询聚合能力（实现了 SlowQueryInspector 才声明）。
	//
	// 与 FeatSessions 分开：会话内省普遍可得，而慢查询依赖 performance_schema 的
	// statements_summary / pg_stat_statements 扩展——「能看会话」不代表「能看到历史慢查询」
	FeatSlowQuery
	// FeatLockWait 锁等待链能力（实现了 LockWaitInspector 才声明）。
	// 单独成能力：有的方言能给等锁计数却拼不出「等待方 → 阻塞方」的关系
	FeatLockWait
	// FeatRecovery 能探测「这台库能否做时间点恢复、能回溯到哪」（实现了 RecoveryInspector 才声明）。
	//
	// 与 FeatSessions 等运行态能力分开：PITR 的前提是数据库侧归档开关（binlog / WAL archive），
	// 那是部署形态的事实，读不到时只能给「不可用 + 原因」，不能退化成「没有备份」
	FeatRecovery
	// FeatAccounts 能盘点实例上的账号与高危标志（实现了 AccountInspector 才声明）。
	//
	// 与 FeatSessions 分开：会话是运行态、多数账号可读；账号表属系统库，
	// 普通账号常被拒绝（MySQL 的 mysql.user 尤其如此），读不到要能明确说出来
	FeatAccounts
	// FeatTLSStatus 能读取「本连接实际协商出的加密状态」（实现了 TLSStatusInspector 才声明）。
	//
	// 与 FeatTLS 相互独立：能建 TLS 连接不等于能证明当下这条连接确实加密了
	// （配置写 verify-full、库侧回落明文仍可能接通），验收要的是后者。
	FeatTLSStatus
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
	FeatSessions:           "sessions",
	FeatSlowQuery:          "slow_query",
	FeatLockWait:           "lock_wait",
	FeatRecovery:           "recovery",
	FeatAccounts:           "accounts",
	FeatExplain:            "explain",
	FeatTLS:                "tls",
	FeatTLSStatus:          "tls_status",
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

	// TLSModes 该方言真实支持的连接加密档位（空表示不支持 TLS）。
	//
	// 与 FeatTLS 是同一事实的两种表达：Has(FeatTLS) 供前端/护栏做布尔判定，
	// TLSModes 供保存期校验与表单渲染档位下拉。二者必须同时成立，
	// 由 dbm 的护栏测试把守「声明了 FeatTLS 就必须至少有一个档位」。
	TLSModes []tlsx.Mode
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
	return MetadataCapabilities{features: m, NamespaceHierarchy: c.NamespaceHierarchy, TLSModes: c.TLSModes}
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
	return MetadataCapabilities{features: m, NamespaceHierarchy: c.NamespaceHierarchy, TLSModes: c.TLSModes}
}

// WithTLSModes 声明该方言支持的连接加密档位，并同步置上 FeatTLS。
//
// 两个事实必须一起写：只给档位不给能力位，前端不会渲染 TLS 入口；
// 只给能力位不给档位，保存期校验就没有合法集合可依据（护栏测试会直接判红）。
// 与其它建造者一样返回新值；Without/With 都透传 TLSModes，故调用顺序不影响结果。
func (c MetadataCapabilities) WithTLSModes(modes ...tlsx.Mode) MetadataCapabilities {
	c.TLSModes = append([]tlsx.Mode(nil), modes...)
	if len(c.TLSModes) > 0 {
		c = c.With(FeatTLS)
	}
	return c
}

// SupportsTLSMode 该方言是否声明支持指定档位。
// disable（含空值）恒定可用——明文是每个方言都具备的能力，不需要声明。
func (c MetadataCapabilities) SupportsTLSMode(mode tlsx.Mode) bool {
	if !mode.Encrypted() {
		return true
	}
	for _, m := range c.TLSModes {
		if m == mode {
			return true
		}
	}
	return false
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

// FeatureName 返回能力枚举的稳定字符串标识；未登记命名的能力返回空串（属声明遗漏，
// 因为每个 MetadataFeature 都必须在 featureNames 里有条目，否则能力协商与护栏日志都会失去可读名字）
func FeatureName(f MetadataFeature) string {
	return featureNames[f]
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
