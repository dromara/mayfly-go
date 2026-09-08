package dbi

import (
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
	"sync"
)

// Metadata 元数据接口（表、列、等元信息）
type Metadata interface {

	// GetDbServer 获取数据库服务实例信息
	GetDbServer() (*DbServer, error)

	// GetCompatibleDbVersion 获取兼容版本信息，如果有兼容版本，则需要实现对应版本的特殊方言处理器，以及前端的方言兼容版本
	GetCompatibleDbVersion() DbVersion

	// GetDefaultDb 获取默认库
	GetDefaultDb() string

	// GetSchemas
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

// 默认实现，若需要覆盖，则由各个数据库MetaData实现去覆盖重写
type DefaultMetadata struct {
}

func (dd *DefaultMetadata) GetCompatibleDbVersion() DbVersion {
	return ""
}

func (dd *DefaultMetadata) GetDefaultDb() string {
	return ""
}

// 数据库服务实例信息
type DbServer struct {
	Version string  `json:"version"` // 版本信息
	Extra   collx.M `json:"extra"`   // 其他额外信息
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
