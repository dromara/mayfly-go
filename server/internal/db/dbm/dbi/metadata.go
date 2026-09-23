package dbi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"mayfly-go/pkg/utils/collx"

	"github.com/spf13/cast"
)

// ========== ServerInfo：数据库服务器信息（静态/轻量）==========

// ServerInfo 数据库服务器信息
type ServerInfo interface {
	// GetDbServer 获取数据库服务实例信息
	GetDbServer() (*DbServer, error)

	// GetCompatibleDbVersion 获取兼容版本信息，如果有兼容版本，则需要实现对应版本的特殊方言处理器，以及前端的方言兼容版本
	GetCompatibleDbVersion() DbVersion

	// GetDefaultDb 获取默认库
	GetDefaultDb() string
}

// ========== MetadataProvider：方言实现的 Schema 内省能力（接口）==========

// MetadataProvider 方言提供的 Schema 内省能力
// 各方言以自身的 Metadata 结构体实现本接口全部方法（SQL 均方言特定，无通用默认实现）
//
// ⚠️ 扩展规约（开闭原则，后续新增内省对象类型前必读）：
// 视图/存储过程/序列/触发器等新元数据对象类型，**禁止往本接口加方法**——
// 必填接口膨胀会一次性打断全部方言实现的编译。正确做法与本仓
// sqlparser 的 PaginationRewriter/StatementClassifier 既有范式一致（可选能力接口 + 探测）：
//  1. 在 dbi 定义独立的可选能力接口，如：
//     type ViewProvider interface { GetViews(names ...string) ([]View, error) }
//  2. 仅支持该能力的方言实现它（保留编译期断言 var _ ViewProvider = (*metadata)(nil)）
//  3. MetadataReader 代理方法内以类型断言探测，未实现的方言返回明确的 ErrNotSupported
//  4. MetadataCapabilities 增加 SupportsViews 声明，上层/前端先查能力再调用
//
// 遵循此规约时，新增对象类型的改动面 = dbi 一个新接口 + 单方言实现，其余方言零修改。
type MetadataProvider interface {
	// GetSchemas 获取数据库的 schema 列表
	GetSchemas() ([]string, error)

	// GetDbNames 获取数据库名称列表
	GetDbNames() ([]string, error)

	// GetTables 获取表信息
	GetTables(tableNames ...string) ([]Table, error)

	// GetColumns 获取指定表名的所有列元信息
	GetColumns(tableNames ...string) ([]Column, error)

	// GetPrimaryKeys 获取表的有序主键列名（联合主键返回多列，按键内顺序）；
	// 无主键返回空切片（不兜底首列——交由上层决定无键策略，如禁用行级编辑/删除）
	GetPrimaryKeys(tableName string) ([]string, error)

	// GetTableIndex 获取表索引信息
	GetTableIndex(tableName string) ([]Index, error)

	// GetTableDDL 获取建表ddl
	GetTableDDL(tableName string, dropBeforeCreate bool) (string, error)
}

// ========== MetadataReader：Schema 元数据访问入口（独立类，带请求级缓存）==========

// MetadataReader Schema 元数据访问入口（短生命周期，按请求创建）
//
//   - 独立类，作为方言 MetadataProvider 的代理
//   - 请求级缓存：仅在本次 MetadataReader 生命周期内有效，避免同一请求内重复查询
//   - 用完即弃：不跨请求共享，其他平台的 DDL 变更在下次请求时可见
//
// 设计决策：持有 DbBackend（而非 *DbConn）引用，解耦元数据访问与连接容器。
// 这样 DbInfo（无连接时）也能创建 MetadataReader 用于纯 SQL 生成/能力查询场景。
type MetadataReader struct {
	backend    DbBackend        // 方言后端（用于 GetCapabilities）
	provider   MetadataProvider // 方言提供的实际查询能力
	serverInfo ServerInfo       // 方言提供的服务器信息

	// 请求级缓存：仅在本次 MetadataReader 生命周期内有效
	cache map[string]any

	// schemaCache 跨请求的进程内元数据缓存（归属 DbInfo，可为 nil，如无连接的纯 SQL 生成场景）。
	// 缓存策略集中于门面消费，方言 provider 不感知；能力探测（m.provider.(X)）不受影响。
	schemaCache *schemaCache
}

// NewMetadataReader 创建新的 MetadataReader（每次请求创建新实例）
// backend 可为 nil（如纯测试场景），此时 GetCapabilities 返回零值。
// schemaCache 可为 nil；非 nil 时为全量表清单与单表列提供跨请求缓存。
func NewMetadataReader(backend DbBackend, provider MetadataProvider, serverInfo ServerInfo, schemaCache *schemaCache) *MetadataReader {
	return &MetadataReader{
		backend:     backend,
		provider:    provider,
		serverInfo:  serverInfo,
		cache:       make(map[string]any),
		schemaCache: schemaCache,
	}
}

// GetDbServer 获取数据库服务实例信息（委托 ServerInfo）
func (m *MetadataReader) GetDbServer() (*DbServer, error) {
	return m.serverInfo.GetDbServer()
}

// GetCompatibleDbVersion 获取兼容版本信息（委托 ServerInfo）
func (m *MetadataReader) GetCompatibleDbVersion() DbVersion {
	return m.serverInfo.GetCompatibleDbVersion()
}

// GetDefaultDb 获取默认库（委托 ServerInfo）
func (m *MetadataReader) GetDefaultDb() string {
	return m.serverInfo.GetDefaultDb()
}

// GetSchemas 获取数据库的 schema 列表（带请求级缓存）
func (m *MetadataReader) GetSchemas() ([]string, error) {
	const key = "schemas"
	if v, ok := m.cache[key]; ok {
		return v.([]string), nil
	}
	result, err := m.provider.GetSchemas()
	if err != nil {
		return nil, err
	}
	m.cache[key] = result
	return result, nil
}

// GetDbNames 获取数据库名称列表（带请求级缓存）
func (m *MetadataReader) GetDbNames() ([]string, error) {
	const key = "dbNames"
	if v, ok := m.cache[key]; ok {
		return v.([]string), nil
	}
	result, err := m.provider.GetDbNames()
	if err != nil {
		return nil, err
	}
	m.cache[key] = result
	return result, nil
}

// GetTableNames 获取表名列表（带请求级缓存）
func (m *MetadataReader) GetTableNames() ([]string, error) {
	const key = "tableNames"
	if v, ok := m.cache[key]; ok {
		return v.([]string), nil
	}
	tables, err := m.provider.GetTables()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = t.TableName
	}
	m.cache[key] = names
	return names, nil
}

// cachedRead 跨请求缓存读穿透：m.schemaCache 命中未过期即返回；否则执行 fn，且仅成功结果入缓存。
// schemaCache 为 nil（无连接的纯 SQL 生成场景）时直接执行 fn。
// TTL / 仅成功入缓存 / nil 兜底等策略单一收敛于此：
// 新增可跨请求缓存的元数据方法只需包一层 cachedRead，无需重复 get/set 样板（开闭原则）。
func (m *MetadataReader) cachedRead[T any](scKey string, fn func() (T, error)) (T, error) {
	if m.schemaCache == nil {
		return fn()
	}
	if v, ok := m.schemaCache.get(scKey); ok {
		return v.(T), nil
	}
	result, err := fn()
	if err != nil {
		var zero T
		return zero, err
	}
	m.schemaCache.set(scKey, result)
	return result, nil
}

// tablesRequest 带表名参数读取表信息，仅请求级 cache 去重（key 含表名列表，不入跨请求缓存）
func (m *MetadataReader) tablesRequest(tableNames ...string) ([]Table, error) {
	key := "tables:" + strings.Join(tableNames, ",")
	if v, ok := m.cache[key]; ok {
		return v.([]Table), nil
	}
	result, err := m.provider.GetTables(tableNames...)
	if err != nil {
		return nil, err
	}
	m.cache[key] = result
	return result, nil
}

// GetTables 获取表信息：无参全量走跨请求 schema 缓存（覆盖 /t-infos 大库表名清单）；带表名过滤仅请求级 cache
func (m *MetadataReader) GetTables(tableNames ...string) ([]Table, error) {
	if len(tableNames) == 0 {
		return m.cachedRead("tables", func() ([]Table, error) { return m.tablesRequest() })
	}
	return m.tablesRequest(tableNames...)
}

// columnsRequest 带表名参数读取列信息，仅请求级 cache 去重
func (m *MetadataReader) columnsRequest(tableNames ...string) ([]Column, error) {
	key := "columns:" + strings.Join(tableNames, ",")
	if v, ok := m.cache[key]; ok {
		return v.([]Column), nil
	}
	result, err := m.provider.GetColumns(tableNames...)
	if err != nil {
		return nil, err
	}
	m.cache[key] = result
	return result, nil
}

// GetColumns 获取指定表名的列元信息：单表调用（fragment 按需取列）走跨请求 schema 缓存；
// 多表批量仅请求级 cache（避免跨请求 key 组合爆炸）
func (m *MetadataReader) GetColumns(tableNames ...string) ([]Column, error) {
	if len(tableNames) == 1 {
		return m.cachedRead("columns:"+tableNames[0], func() ([]Column, error) { return m.columnsRequest(tableNames...) })
	}
	return m.columnsRequest(tableNames...)
}

// GetPrimaryKeys 获取表的有序主键列名（联合主键多列；无主键返回空切片），带请求级缓存
func (m *MetadataReader) GetPrimaryKeys(tableName string) ([]string, error) {
	key := "pks:" + tableName
	if v, ok := m.cache[key]; ok {
		return v.([]string), nil
	}
	result, err := m.provider.GetPrimaryKeys(tableName)
	if err != nil {
		return nil, err
	}
	m.cache[key] = result
	return result, nil
}

// GetTableIndex 获取表索引信息（带请求级缓存）
func (m *MetadataReader) GetTableIndex(tableName string) ([]Index, error) {
	key := "indexes:" + tableName
	if v, ok := m.cache[key]; ok {
		return v.([]Index), nil
	}
	result, err := m.provider.GetTableIndex(tableName)
	if err != nil {
		return nil, err
	}
	m.cache[key] = result
	return result, nil
}

// GetTableDDL 获取建表 DDL（不缓存，因参数含 dropBeforeCreate 选项）
func (m *MetadataReader) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	return m.provider.GetTableDDL(tableName, dropBeforeCreate)
}

// GetCapabilities 返回当前方言的元数据能力声明（委托 DbBackend）。
// backend 为 nil 时返回零值（所有能力为 false），不会 panic。
func (m *MetadataReader) GetCapabilities() MetadataCapabilities {
	if m.backend == nil {
		return MetadataCapabilities{}
	}
	return m.backend.GetCapabilities()
}

// ========== 表名过滤（超大 schema 服务端下推 / 回退）==========

// TableSearcher 可选能力：把表名模糊过滤下推到系统目录查询，用于超大 schema 的资源树按需加载，
// 避免先全量取回再过滤。未实现本能力的方言由 MetadataReader.SearchTables 回退为「GetTables + 内存子串过滤」，
// 功能一致、仅缺下推优化（渐进接入，开闭原则）。
type TableSearcher interface {
	// SearchTables 返回表名匹配 like（子串、不区分大小写）的表；limit <= 0 表示不限制条数。
	SearchTables(like string, limit int) ([]Table, error)
}

// SearchTables 按表名过滤/限量获取表清单：like 与 limit 皆空时等价 GetTables 全量；
// 具备 TableSearcher 则把 LIKE/LIMIT 下推到系统目录，否则回退全量取回 + 不区分大小写子串过滤。
// 传 limit>0 且 like 为空即为「限量探测」（如判断某库表是否过多以决定是否启用搜索），不必然全量拉取。
func (m *MetadataReader) SearchTables(like string, limit int) ([]Table, error) {
	if like == "" && limit <= 0 {
		return m.GetTables()
	}
	if ts, ok := m.provider.(TableSearcher); ok {
		tables, err := ts.SearchTables(like, limit)
		if err == nil {
			return tables, nil
		}
		// 下推失败（如某方言专属系统目录 SQL 在特定实例/版本不兼容）：回退「全量+过滤」通用路径，
		// 保证表浏览不被专属优化拖垮；真实的连接错误会在回退的 GetTables 再次暴露并返回。
		base, ferr := m.GetTables()
		if ferr != nil {
			return nil, err
		}
		return filterTablesByLike(base, like, limit), nil
	}
	tables, err := m.GetTables()
	if err != nil {
		return nil, err
	}
	return filterTablesByLike(tables, like, limit), nil
}

// EscapeLikeWildcards 转义 LIKE 模式中的通配符 % \ _（各方言 TableSearcher 下推共用）。
// 用户输入的表名按字面子串匹配，不应被当作 LIKE 通配符；反斜杠须最先转义。
// mysql/postgres 的 LIKE 均以反斜杠为默认转义符；oracle 无默认转义符需自带 ESCAPE 子句。
func EscapeLikeWildcards(s string) string {
	if !strings.ContainsAny(s, `%\_`) {
		return s
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// filterTablesByLike 不区分大小写的表名子串过滤（like 为空则不过滤），limit>0 时截断。
// MetadataReader.SearchTables 的无下推回退路径复用；空 like + limit 即「限量探测」。
func filterTablesByLike(tables []Table, like string, limit int) []Table {
	pattern := strings.ToLower(like)
	out := make([]Table, 0, 16)
	for _, t := range tables {
		if pattern == "" || strings.Contains(strings.ToLower(t.TableName), pattern) {
			out = append(out, t)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out
}

// ========== 扩展元数据对象模型：表/列/索引之外的对象（视图/序列/外键等）==========
//
// 与上方「核心访问」的分工：MetadataProvider / Metadata 是各方言**必须**实现的表/列/索引内省；
// 本段 MetadataNavigator / ForeignKeyProvider 是**可选**能力接口，供资源树按类别懒加载「新对象」，
// 方言未实现则其 MetadataReader 代理返回 ErrUnsupportedKind。新增对象类别只加常量 + 单方言实现，其余方言零改动
// （与 sqlparser 的 PaginationRewriter/StatementClassifier、本包 ParamInserter 同属「可选能力 + 探测」范式）。

// ErrUnsupportedKind 表示某方言未支持请求的元数据对象类别。
// 上层用 errors.Is 判定，可据此在前端隐藏对应树节点。
var ErrUnsupportedKind = errors.New("unsupported metadata object kind")

// ObjectKind 元数据对象类别标识。
//
// 仅覆盖「尚无一等类型」的新对象：schema/table/column/index 已有 Table/Column/Index
// 与专用方法，不在此列。新增类别只需加常量并在目标方言 ListObjects 内处理，其余方言零改动。
type ObjectKind string

const (
	KindView      ObjectKind = "view"
	KindProcedure ObjectKind = "procedure"
	KindFunction  ObjectKind = "function"
	KindSequence  ObjectKind = "sequence"
	KindTrigger   ObjectKind = "trigger"
)

// MetadataObject 元数据对象的统一轻量描述，供资源树渲染成对象节点。
//
// 通用字段覆盖大多数对象的展示需求；类别特有信息（如视图定义文本、序列的数据类型）
// 放 Attrs，避免为每新增一类对象就往结构体塞字段。
type MetadataObject struct {
	Name    string         `json:"name"`
	Kind    ObjectKind     `json:"kind"`
	Schema  string         `json:"schema,omitempty"`
	Comment string         `json:"comment,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// MetadataNavigator 可选能力接口：对新对象类别做「列对象 + 取 DDL」的通用内省。
//
// 方言按需实现（编译期 `var _ MetadataNavigator = (*XxxMetadata)(nil)` 断言），
// 只支持自身具备的 kind，对不支持的 kind 返回 ErrUnsupportedKind。
type MetadataNavigator interface {
	// SupportedKinds 返回本方言 ListObjects 真正可列举的对象类别，供「能力声明⟺实现」做 kind 级一致性校验。
	// 必须是纯静态方法（返回字面量集合，不得触碰连接/执行 SQL）——护栏需在无库环境下构造 provider 引用即可调用。
	// 注意：某 kind 可列举不代表 ObjectDDL 也支持（ObjectDDL 支持面可更窄，如某方言能列出某类对象但不生成其 DDL）。
	SupportedKinds() []ObjectKind

	// ListObjects 列出 schema 下某类对象。schema 为空表示当前库/模式。
	ListObjects(ctx context.Context, schema string, kind ObjectKind) ([]MetadataObject, error)

	// ObjectDDL 返回对象的重建 DDL 原文。kind 不受支持时返回 ErrUnsupportedKind。
	ObjectDDL(ctx context.Context, schema string, kind ObjectKind, name string) (string, error)
}

// ForeignKey 表间关系（外键）的结构化详情。
//
// 列级映射无法用通用导航节点表达，且被 ER 图、跨库迁移建表拓扑排序等结构化消费，
// 故单独建模，而非塞进 MetadataObject.Attrs。
type ForeignKey struct {
	Name      string `json:"name"`
	Table     string `json:"table"`
	Column    string `json:"column"`
	RefTable  string `json:"refTable"`
	RefColumn string `json:"refColumn"`
	OnUpdate  string `json:"onUpdate,omitempty"`
	OnDelete  string `json:"onDelete,omitempty"`
}

// ForeignKeyProvider 可选能力接口：内省指定表的外键关系。
type ForeignKeyProvider interface {
	GetForeignKeys(ctx context.Context, schema, table string) ([]ForeignKey, error)
}

// KeyType 表键约束类别。仅覆盖主键与唯一键——它们才是「行标识」的来源；
// 外键另有 ForeignKeyProvider，普通索引另有 GetTableIndex，职责不混。
type KeyType string

const (
	KeyTypePrimary KeyType = "primary" // 主键
	KeyTypeUnique  KeyType = "unique"  // 唯一键
)

// KeyColumn 键约束中的一列。Ordinal 为键内序号（1 起），联合键顺序敏感（PK(a,b) ≠ PK(b,a)），
// 与 information_schema.KEY_COLUMN_USAGE.ORDINAL_POSITION 对齐。
type KeyColumn struct {
	Name    string `json:"name"`
	Ordinal int    `json:"ordinal"`
}

// KeyConstraint 表键约束（主键/唯一键）的有序列集合。
//
// 建模对齐 information_schema.TABLE_CONSTRAINTS + KEY_COLUMN_USAGE：键是一等约束对象、
// 成员列带序，而非「每列一个 isPrimaryKey 布尔」——后者无法表达联合键的顺序，也逼出 GetPrimaryKey
// 单列兜底之类的错误。行级更新/删除的 WHERE、迁移 upsert 冲突键都应以本结构为准。
type KeyConstraint struct {
	Name    string      `json:"name"`
	Type    KeyType     `json:"type"`
	Columns []KeyColumn `json:"columns"`
}

// KeyProvider 可选能力接口：内省指定表的主键与唯一键约束（有序列）。
type KeyProvider interface {
	GetKeys(ctx context.Context, schema, table string) ([]KeyConstraint, error)
}

// ParseKeyRows 将统一形状的内省结果（列别名 keyName/keyType/columnName/ordinal）聚合为有序 KeyConstraint 列表。
// 采用 information_schema 风格四列查询的方言（mysql/pgsql/mssql/oracle/dm）共用本函数，避免分组逻辑在各方言重复漂移；
// keyType 值为 'PRIMARY KEY' 视为主键，其余按唯一键。sqlite/clickhouse 结构不同，各自组装。
func ParseKeyRows(res []map[string]any) []KeyConstraint {
	keys := make([]KeyConstraint, 0, len(res))
	idx := make(map[string]int, len(res))
	for _, re := range res {
		name := cast.ToString(re["keyName"])
		kt := KeyTypeUnique
		if cast.ToString(re["keyType"]) == "PRIMARY KEY" {
			kt = KeyTypePrimary
		}
		col := KeyColumn{Name: cast.ToString(re["columnName"]), Ordinal: cast.ToInt(re["ordinal"])}
		if at, ok := idx[name]; ok {
			keys[at].Columns = append(keys[at].Columns, col)
			continue
		}
		idx[name] = len(keys)
		keys = append(keys, KeyConstraint{Name: name, Type: kt, Columns: []KeyColumn{col}})
	}
	return keys
}

// ---------- MetadataReader 代理：对 provider 做能力探测，未实现者返回明确错误而非静默空 ----------

// ListObjects 代理到 provider 的 MetadataNavigator 能力；方言未实现则返回 ErrUnsupportedKind。
func (m *MetadataReader) ListObjects(ctx context.Context, schema string, kind ObjectKind) ([]MetadataObject, error) {
	nav, ok := m.provider.(MetadataNavigator)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedKind, kind)
	}
	return nav.ListObjects(ctx, schema, kind)
}

// ObjectDDL 代理到 provider 的 MetadataNavigator 能力；方言未实现则返回 ErrUnsupportedKind。
func (m *MetadataReader) ObjectDDL(ctx context.Context, schema string, kind ObjectKind, name string) (string, error) {
	nav, ok := m.provider.(MetadataNavigator)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedKind, kind)
	}
	return nav.ObjectDDL(ctx, schema, kind, name)
}

// GetForeignKeys 代理到 provider 的 ForeignKeyProvider 能力；方言未实现则返回 ErrUnsupportedKind。
func (m *MetadataReader) GetForeignKeys(ctx context.Context, schema, table string) ([]ForeignKey, error) {
	rr, ok := m.provider.(ForeignKeyProvider)
	if !ok {
		return nil, fmt.Errorf("%w: relations", ErrUnsupportedKind)
	}
	return rr.GetForeignKeys(ctx, schema, table)
}

// GetKeys 代理到 provider 的 KeyProvider 能力（主键/唯一键有序列）；方言未实现则返回 ErrUnsupportedKind。
func (m *MetadataReader) GetKeys(ctx context.Context, schema, table string) ([]KeyConstraint, error) {
	kr, ok := m.provider.(KeyProvider)
	if !ok {
		return nil, fmt.Errorf("%w: keys", ErrUnsupportedKind)
	}
	return kr.GetKeys(ctx, schema, table)
}

// ========== DefaultServerInfo：ServerInfo 默认实现 ==========

// DefaultServerInfo ServerInfo 默认实现，若需要覆盖则由各方言实现去覆盖重写
type DefaultServerInfo struct{}

func (dd *DefaultServerInfo) GetCompatibleDbVersion() DbVersion {
	return ""
}

func (dd *DefaultServerInfo) GetDefaultDb() string {
	return ""
}

// 数据库服务实例信息
type DbServer struct {
	Version      string  `json:"version"`      // 原始版本字符串
	MajorVersion int     `json:"majorVersion"` // 主版本号（如 MySQL 8、PostgreSQL 16）
	MinorVersion int     `json:"minorVersion"` // 次版本号
	Extra        collx.M `json:"extra"`        // 其他额外信息
}

// 表信息
type Table struct {
	TableName    string `json:"tableName"`    // 表名
	TableComment string `json:"tableComment"` // 表备注
	CreateTime   string `json:"createTime"`   // 创建时间
	TableRows    int    `json:"tableRows"`
	DataLength   int64  `json:"dataLength"`
	IndexLength  int64  `json:"indexLength"`
}

// 表索引信息
type Index struct {
	IndexName    string  `json:"indexName"`    // 索引名
	ColumnName   string  `json:"columnName"`   // 列名
	IndexType    string  `json:"indexType"`    // 索引类型
	IndexComment string  `json:"indexComment"` // 备注
	SeqInIndex   int     `json:"seqInIndex"`
	IsUnique     bool    `json:"isUnique"`
	IsPrimaryKey bool    `json:"isPrimaryKey"` // 是否是主键索引，某些情况需要判断并过滤掉主键索引
	Extra        collx.M `json:"extra"`        // 其他额外信息，如索引列的前缀长度等
}

// ------------------------- 元数据查询辅助函数 -------------------------

// GroupIndexColumns 将平铺的索引记录按索引名分组，同索引的多个列名以逗号连接。
// 数据库索引查询结果通常每个列一行，本函数将其合并为每个索引一条记录。
// 适用于 MySQL/PostgreSQL/Oracle/DM 等标准方言的索引结果合并。
//
// 采用「索引名→首次出现位置」映射做顺序无关分组（与 ParseKeyRows 一致），
// 避免同名索引行不相邻时被拆成多条重复记录；列名按输入顺序拼接。
func GroupIndexColumns(indexes []Index) []Index {
	result := make([]Index, 0, len(indexes))
	pos := make(map[string]int, len(indexes))
	for _, idx := range indexes {
		if at, ok := pos[idx.IndexName]; ok {
			// 同索引字段以逗号连接
			result[at].ColumnName += "," + idx.ColumnName
			continue
		}
		pos[idx.IndexName] = len(result)
		result = append(result, idx)
	}
	return result
}
