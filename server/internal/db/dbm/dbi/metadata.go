package dbi

import (
	"embed"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
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
//go:embed metasql/*
var metasql embed.FS

// sql缓存 key: sql备注的key 如：MYSQL_TABLE_MA  value: sql内容
var sqlCache = make(map[string]string, 20)

// 获取本地文件的sql内容，并进行解析，获取对应key的sql内容
func GetLocalSql(file, key string) string {
	sql := sqlCache[key]
	if sql != "" {
		return sql
	}

	bytes, err := metasql.ReadFile(file)
	biz.ErrIsNilAppendErr(err, "failed to get the contents of the sql meta file: %s")
	allSql := string(bytes)

	sqls := strings.Split(allSql, "---------------------------------------")
	var resSql string
	for _, sql := range sqls {
		sql = stringx.TrimSpaceAndBr(sql)
		// 获取sql第一行的sql备注信息如：--MYSQL_TABLE_MA 表信息元数据
		info := strings.SplitN(sql, "\n", 2)
		// 原始sql，即去除第一行的key与备注信息
		rowSql := info[1]
		// 获取sql key；如：MYSQL_TABLE_MA
		sqlKey := strings.Split(strings.Split(info[0], " ")[0], "--")[1]
		if key == sqlKey {
			resSql = rowSql
		}
		sqlCache[sqlKey] = rowSql
	}
	return resSql
}
