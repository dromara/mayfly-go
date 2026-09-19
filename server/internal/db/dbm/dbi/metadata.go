package dbi

import (
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
	"sync"
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
// 各方言实现此接口（嵌入 DefaultMetadataProvider 覆写差异方法）
//
// ⚠️ 扩展规约（开闭原则，后续新增内省对象类型前必读）：
// 视图/存储过程/序列/触发器等新元数据对象类型，**禁止往本接口加方法**——
// 必填接口膨胀会一次性打断全部方言实现的编译。正确做法与本仓
// sqlparser 的 PaginationRewriter/StatementClassifier 既有范式一致（可选能力接口 + 探测）：
//  1. 在 dbi 定义独立的可选能力接口，如：
//     type ViewProvider interface { GetViews(names ...string) ([]View, error) }
//  2. 仅支持该能力的方言实现它（保留编译期断言 var _ ViewProvider = (*metadata)(nil)）
//  3. Metadata 代理方法内以类型断言探测，未实现的方言返回明确的 ErrNotSupported
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

	// GetPrimaryKey 获取表主键字段名，没有主键标识则默认第一个字段
	GetPrimaryKey(tableName string) (string, error)

	// GetTableIndex 获取表索引信息
	GetTableIndex(tableName string) ([]Index, error)

	// GetTableDDL 获取建表ddl
	GetTableDDL(tableName string, dropBeforeCreate bool) (string, error)
}

// ========== Metadata：Schema 元数据访问入口（独立类，带请求级缓存）==========

// Metadata Schema 元数据访问入口（短生命周期，按请求创建）
//
//   - 独立类，作为方言 MetadataProvider 的代理
//   - 请求级缓存：仅在本次 Metadata 生命周期内有效，避免同一请求内重复查询
//   - 用完即弃：不跨请求共享，其他平台的 DDL 变更在下次请求时可见
//
// 设计决策：持有 DbBackend（而非 *DbConn）引用，解耦元数据访问与连接容器。
// 这样 DbInfo（无连接时）也能创建 Metadata 用于纯 SQL 生成/能力查询场景。
type Metadata struct {
	backend    DbBackend        // 方言后端（用于 GetCapabilities）
	provider   MetadataProvider // 方言提供的实际查询能力
	serverInfo ServerInfo       // 方言提供的服务器信息

	// 请求级缓存：仅在本次 Metadata 生命周期内有效
	cache map[string]any
}

// NewMetadata 创建新的 Metadata（每次请求创建新实例）
// backend 可为 nil（如纯测试场景），此时 GetCapabilities/HasFeature 返回零值。
func NewMetadata(backend DbBackend, provider MetadataProvider, serverInfo ServerInfo) *Metadata {
	return &Metadata{
		backend:    backend,
		provider:   provider,
		serverInfo: serverInfo,
		cache:      make(map[string]any),
	}
}

// GetDbServer 获取数据库服务实例信息（委托 ServerInfo）
func (m *Metadata) GetDbServer() (*DbServer, error) {
	return m.serverInfo.GetDbServer()
}

// GetCompatibleDbVersion 获取兼容版本信息（委托 ServerInfo）
func (m *Metadata) GetCompatibleDbVersion() DbVersion {
	return m.serverInfo.GetCompatibleDbVersion()
}

// GetDefaultDb 获取默认库（委托 ServerInfo）
func (m *Metadata) GetDefaultDb() string {
	return m.serverInfo.GetDefaultDb()
}

// GetSchemas 获取数据库的 schema 列表（带请求级缓存）
func (m *Metadata) GetSchemas() ([]string, error) {
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
func (m *Metadata) GetDbNames() ([]string, error) {
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
func (m *Metadata) GetTableNames() ([]string, error) {
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

// GetTables 获取表信息（带请求级缓存）
func (m *Metadata) GetTables(tableNames ...string) ([]Table, error) {
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

// GetColumns 获取指定表名的所有列元信息（带请求级缓存）
func (m *Metadata) GetColumns(tableNames ...string) ([]Column, error) {
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

// GetPrimaryKey 获取表主键字段名（带请求级缓存）
func (m *Metadata) GetPrimaryKey(tableName string) (string, error) {
	key := "pk:" + tableName
	if v, ok := m.cache[key]; ok {
		return v.(string), nil
	}
	result, err := m.provider.GetPrimaryKey(tableName)
	if err != nil {
		return "", err
	}
	m.cache[key] = result
	return result, nil
}

// GetTableIndex 获取表索引信息（带请求级缓存）
func (m *Metadata) GetTableIndex(tableName string) ([]Index, error) {
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
func (m *Metadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	return m.provider.GetTableDDL(tableName, dropBeforeCreate)
}

// ClearCache 清空请求级缓存（同一请求内需要刷新时调用）
func (m *Metadata) ClearCache() {
	m.cache = make(map[string]any)
}

// GetCapabilities 返回当前方言的元数据能力声明（委托 DbBackend）。
// backend 为 nil 时返回零值（所有能力为 false），不会 panic。
func (m *Metadata) GetCapabilities() MetadataCapabilities {
	if m.backend == nil {
		return MetadataCapabilities{}
	}
	return m.backend.GetCapabilities()
}

// HasFeature 检查当前方言在给定版本下是否支持指定特性。
// 先查版本相关能力表（VersionFeatures），再查静态能力声明。
// serverVersion 为当前数据库版本（由 GetDbServer 获取），空串表示版本未知。
func (m *Metadata) HasFeature(featureName string) bool {
	caps := m.GetCapabilities()

	// 1. 查版本相关能力表
	if minVer, ok := caps.VersionFeatures[featureName]; ok {
		server, err := m.GetDbServer()
		if err != nil || server == nil {
			return false // 无法获取版本信息，保守返回 false
		}
		return CompareDbVersion(server.Version, string(minVer)) >= 0
	}

	// 2. 查静态能力声明
	return caps.HasStaticFeature(featureName)
}

// ========== MetadataCapabilities：方言能力声明 ==========

// MetadataCapabilities 方言元数据能力声明
// 用于前端动态查询方言支持的能力、运维诊断、新增方言时声明特性
type MetadataCapabilities struct {
	SupportsSchemas     bool // 是否支持 Schema（pg/mssql/oracle/dm 支持，mysql/sqlite/clickhouse 不支持）
	SupportsIndexes     bool // 是否支持索引
	SupportsForeignKeys bool // 是否支持外键
	SupportsComments    bool // 是否支持表/列注释
	SupportsDDLExport   bool // 是否支持 DDL 导出

	// 高级能力维度
	SupportsGeneratedColumns  bool // 是否支持生成列（且可通过元数据准确识别）
	SupportsIdentityColumns   bool // 是否支持标识列/序列自增语义
	SupportsExpressionDefault bool // 是否能安全区分字面量默认值与表达式默认值

	// NamespaceHierarchy 命名空间层次声明（Catalog > Schema > Table 三层可选模型）
	NamespaceHierarchy NamespaceHierarchy

	// VersionFeatures 版本相关能力表
	// key: 特性名称（如 "window_functions"、"stored_generated_columns"）
	// value: 最低支持版本（数据库版本 >= 此版本时支持该特性）
	VersionFeatures map[string]DbVersion
}

// HasStaticFeature 检查静态能力声明（按特性名称映射到对应字段）
func (c MetadataCapabilities) HasStaticFeature(featureName string) bool {
	switch featureName {
	case "schemas":
		return c.SupportsSchemas
	case "indexes":
		return c.SupportsIndexes
	case "foreign_keys":
		return c.SupportsForeignKeys
	case "comments":
		return c.SupportsComments
	case "ddl_export":
		return c.SupportsDDLExport
	case "generated_columns":
		return c.SupportsGeneratedColumns
	case "identity_columns":
		return c.SupportsIdentityColumns
	case "expression_default":
		return c.SupportsExpressionDefault
	default:
		return false
	}
}

// ========== NamespaceHierarchy：命名空间层次声明 ==========

// NamespaceHierarchy 数据库命名空间层次结构声明。
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

// ========== CompareDbVersion：版本比较 ==========

// CompareDbVersion 比较两个版本字符串的大小。
// 返回 -1（a < b）、0（a == b）、1（a > b）。
// 支持格式："8.0.32"、"16.1"、"22.0.0.0.0" 等。
// 从左到右逐段比较数值，跳过非数字前缀。
func CompareDbVersion(a, b string) int {
	aParts := parseVersionParts(a)
	bParts := parseVersionParts(b)

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		aVal, bVal := 0, 0
		if i < len(aParts) {
			aVal = aParts[i]
		}
		if i < len(bParts) {
			bVal = bParts[i]
		}
		if aVal < bVal {
			return -1
		}
		if aVal > bVal {
			return 1
		}
	}
	return 0
}

// parseVersionParts 从版本字符串中提取数值段
func parseVersionParts(version string) []int {
	// 跳过非数字前缀
	start := 0
	for start < len(version) && (version[start] < '0' || version[start] > '9') {
		start++
	}
	if start >= len(version) {
		return nil
	}
	version = version[start:]

	var parts []int
	current := 0
	hasDigit := false
	for i := 0; i < len(version); i++ {
		c := version[i]
		if c >= '0' && c <= '9' {
			current = current*10 + int(c-'0')
			hasDigit = true
		} else if c == '.' && hasDigit {
			parts = append(parts, current)
			current = 0
			hasDigit = false
		} else if hasDigit {
			parts = append(parts, current)
			current = 0
			hasDigit = false
		}
	}
	if hasDigit {
		parts = append(parts, current)
	}
	return parts
}

// ParseDbVersion 从原始版本字符串解析主/次版本号，填充到 DbServer。
// 支持常见格式："8.0.32"、"16.1 (Debian 16.1-1.pgdg120+1)"、"22.0.0.0.0" 等。
// 解析失败时 MajorVersion/MinorVersion 保持零值，不影响 Version 原始字段。
func ParseDbVersion(server *DbServer) {
	if server == nil || server.Version == "" {
		return
	}
	parts := parseVersionParts(server.Version)
	if len(parts) > 0 {
		server.MajorVersion = parts[0]
	}
	if len(parts) > 1 {
		server.MinorVersion = parts[1]
	}
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

// ========== DefaultMetadataProvider：MetadataProvider 默认实现 ==========

// DefaultMetadataProvider MetadataProvider 默认实现
// 当前 MetadataProvider 的所有方法均需方言特定实现，无通用默认值
// 各方言嵌入此结构体仅为保持嵌入模式一致性，便于未来添加通用默认实现
type DefaultMetadataProvider struct{}

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
func GroupIndexColumns(indexes []Index) []Index {
	result := make([]Index, 0)
	prevKey := ""
	for _, idx := range indexes {
		if prevKey == idx.IndexName {
			// 同索引字段以逗号连接
			last := len(result) - 1
			result[last].ColumnName += "," + idx.ColumnName
		} else {
			prevKey = idx.IndexName
			result = append(result, idx)
		}
	}
	return result
}

// ------------------------- 元数据sql操作 -------------------------
//
// 各方言元数据SQL模板由方言包自持（//go:embed 与方言实现同居一处，新增方言时包内自包含），
// dbi仅提供通用的解析与缓存能力（SqlTemplates）

// SqlTemplates 方言元数据SQL模板：解析「--KEY 备注说明」分段格式的sql文件内容，
// 按备注key取用并缓存。格式：段落以分隔线切分，每段首行为 --KEY 备注信息
// （如 --MYSQL_TABLE_INFO 表详细信息），正文为实际sql
//
// 用法（方言包内）：
//
//	//go:embed meta.sql
//	var metaSqlFile string
//	var metaSql = dbi.NewSqlTemplates(metaSqlFile)
type SqlTemplates struct {
	content string
	mu      sync.RWMutex // 保护 cache 的并发读写
	cache   map[string]string
}

func NewSqlTemplates(content string) *SqlTemplates {
	return &SqlTemplates{content: content, cache: make(map[string]string, 20)}
}

// Get 获取key对应的sql内容，首次访问时解析全量段落并缓存
func (t *SqlTemplates) Get(key string) string {
	t.mu.RLock()
	sql := t.cache[key]
	t.mu.RUnlock()
	if sql != "" {
		return sql
	}

	allSql := t.content
	sqls := strings.Split(allSql, "---------------------------------------")
	var resSql string
	for _, sql := range sqls {
		sql = stringx.TrimSpaceAndBr(sql)
		if sql == "" {
			continue
		}
		// 获取sql第一行的sql备注信息如：--MYSQL_TABLE_MA 表信息元数据
		info := strings.SplitN(sql, "\n", 2)
		if len(info) < 2 {
			// 内容只有一行（无实际sql），跳过，避免越界
			continue
		}
		// 获取sql key；如：MYSQL_TABLE_MA，格式不合法则跳过
		keyParts := strings.Split(strings.Split(info[0], " ")[0], "--")
		if len(keyParts) < 2 {
			continue
		}
		sqlKey := keyParts[1]
		// 原始sql，即去除第一行的key与备注信息
		rowSql := info[1]
		if key == sqlKey {
			resSql = rowSql
		}
		t.mu.Lock()
		t.cache[sqlKey] = rowSql
		t.mu.Unlock()
	}
	if resSql == "" {
		logx.Error("sql metadata key not found: %s", key)
	}
	return resSql
}
