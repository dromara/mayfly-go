package dbi

import (
	"database/sql"
	"embed"
	"strings"

	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
)

type DataType string

const (
	DataTypeString   DataType = "string"
	DataTypeNumber   DataType = "number"
	DataTypeDate     DataType = "date"
	DataTypeTime     DataType = "time"
	DataTypeDateTime DataType = "datetime"
)

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

// 表的列信息
type Column struct {
	TableName     string  `json:"tableName"`     // 表名
	ColumnName    string  `json:"columnName"`    // 列名
	ColumnType    string  `json:"columnType"`    // 列类型
	ColumnComment string  `json:"columnComment"` // 列备注
	IsPrimaryKey  bool    `json:"isPrimaryKey"`  // 是否为主键
	IsIdentity    bool    `json:"isIdentity"`    // 是否自增
	ColumnDefault string  `json:"columnDefault"` // 默认值
	Nullable      string  `json:"nullable"`      // 是否可为null
	NumScale      string  `json:"numScale"`      // 小数点
	Extra         collx.M `json:"extra"`         // 其他额外信息
}

// 表索引信息
type Index struct {
	IndexName    string `json:"indexName"`    // 索引名
	ColumnName   string `json:"columnName"`   // 列名
	IndexType    string `json:"indexType"`    // 索引类型
	IndexComment string `json:"indexComment"` // 备注
	SeqInIndex   int    `json:"seqInIndex"`
	IsUnique     bool   `json:"isUnique"`
}

type DbCopyTable struct {
	Id        uint64 `json:"id"`
	Db        string `json:"db" `
	TableName string `json:"tableName"`
	CopyData  bool   `json:"copyData"` // 是否复制数据
}

// 数据转换器
type DataConverter interface {
	// 获取数据对应的类型
	// @param dbColumnType 数据库原始列类型，如varchar等
	GetDataType(dbColumnType string) DataType

	// 根据数据类型格式化指定数据
	FormatData(dbColumnValue any, dataType DataType) string

	// 根据数据类型解析数据为符合要求的指定类型等
	ParseData(dbColumnValue any, dataType DataType) any
}

// -----------------------------------元数据接口定义------------------------------------------
// 数据库方言、元信息接口（表、列、获取表数据等元信息）
type Dialect interface {

	// 获取数据库服务实例信息
	GetDbServer() (*DbServer, error)

	// 获取数据库名称列表
	GetDbNames() ([]string, error)

	// 获取表信息
	GetTables() ([]Table, error)

	// 获取指定表名的所有列元信息
	GetColumns(tableNames ...string) ([]Column, error)

	// 获取表主键字段名，没有主键标识则默认第一个字段
	GetPrimaryKey(tableName string) (string, error)

	// 获取表索引信息
	GetTableIndex(tableName string) ([]Index, error)

	// 获取建表ddl
	GetTableDDL(tableName string) (string, error)

	GetSchemas() ([]string, error)

	// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
	GetDbProgram() (DbProgram, error)

	// 批量保存数据
	BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error)

	// 获取数据转换器用于解析格式化列数据等
	GetDataConverter() DataConverter

	CopyTable(copy *DbCopyTable) error
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
