package application

import (
	"embed"
	"mayfly-go/pkg/biz"
	"strings"
)

// 表信息
type Table struct {
	TableName    string `json:"tableName"`    // 表名
	TableComment string `json:"tableComment"` // 表备注
	CreateTime   string `json:"createTime"`   // 创建时间
	TableRows    int    `json:"tableRows"`
	DataLength   int64  `json:"dataLength"`
	IndexLength  int64  `json:"indexLength"`
}

// 表的列信息
type Column struct {
	TableName     string `json:"tableName"`     // 表名
	ColumnName    string `json:"columnName"`    // 列名
	ColumnType    string `json:"columnType"`    // 列类型
	ColumnComment string `json:"columnComment"` // 列备注
	ColumnKey     string `json:"columnKey"`
	ColumnDefault string `json:"columnDefault"`
	Nullable      string `json:"nullable"` // 是否可为null
	Extra         string `json:"extra"`    // 其他信息
}

// 表索引信息
type Index struct {
	IndexName    string `json:"indexName"`    // 索引名
	ColumnName   string `json:"columnName"`   // 列名
	IndexType    string `json:"indexType"`    // 索引类型
	IndexComment string `json:"indexComment"` // 备注
	SeqInIndex   int    `json:"seqInIndex"`
	NonUnique    int    `json:"nonUnique"`
}

// -----------------------------------元数据接口定义------------------------------------------
// 数据库元信息接口（表、列、获取表数据等元信息）
type DbMetadata interface {

	// 获取表基础元信息
	GetTables() []Table

	// 获取指定表名的所有列元信息
	GetColumns(tableNames ...string) []Column

	// 获取表主键字段名，没有主键标识则默认第一个字段
	GetPrimaryKey(tablename string) string

	// 获取表信息，比GetTables获取更详细的表信息
	GetTableInfos() []Table

	// 获取表索引信息
	GetTableIndex(tableName string) []Index

	// 获取建表ddl
	GetCreateTableDdl(tableName string) string

	// 获取指定表的数据-分页查询
	// @return columns: 列字段名；result: 结果集；error: 错误
	GetTableRecord(tableName string, pageNum, pageSize int) ([]string, []map[string]interface{}, error)
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
	biz.ErrIsNilAppendErr(err, "获取sql meta文件内容失败: %s")
	allSql := string(bytes)

	sqls := strings.Split(allSql, "\n\n")
	var resSql string
	for _, sql := range sqls {
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
