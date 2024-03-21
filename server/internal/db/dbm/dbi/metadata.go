package dbi

import (
	"embed"
	"fmt"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
)

// 元数据接口（表、列、等元信息）
type MetaData interface {
	BaseMetaData

	// 获取数据库服务实例信息
	GetDbServer() (*DbServer, error)

	// 获取数据库名称列表
	GetDbNames() ([]string, error)

	// 获取表信息
	GetTables(tableNames ...string) ([]Table, error)

	// 获取指定表名的所有列元信息
	GetColumns(tableNames ...string) ([]Column, error)

	// 获取表主键字段名，没有主键标识则默认第一个字段
	GetPrimaryKey(tableName string) (string, error)

	// 获取表索引信息
	GetTableIndex(tableName string) ([]Index, error)

	// 获取建表ddl
	GetTableDDL(tableName string) (string, error)

	GenerateTableDDL(columns []Column, tableInfo Table, dropBeforeCreate bool) []string

	GenerateIndexDDL(indexs []Index, tableInfo Table) []string

	GetSchemas() ([]string, error)

	// 获取数据转换器用于解析格式化列数据等
	GetDataConverter() DataConverter
}

// 数据库服务实例信息
type DbServer struct {
	Version string  `json:"version"` // 版本信息
	Extra   collx.M `json:"extra"`   // 其他额外信息
}

// 表信息
type Table struct {
	TableName    string `json:"tableName"`    // 表名
	TableNewName string `json:"tableNewName"` // 新表名，复制表生成ddl时，需要传入新表名
	TableComment string `json:"tableComment"` // 表备注
	CreateTime   string `json:"createTime"`   // 创建时间
	TableRows    int    `json:"tableRows"`
	DataLength   int64  `json:"dataLength"`
	IndexLength  int64  `json:"indexLength"`
}

// 表的列信息
type Column struct {
	TableName     string         `json:"tableName"`     // 表名
	ColumnName    string         `json:"columnName"`    // 列名
	DataType      ColumnDataType `json:"dataType"`      // 数据类型
	ColumnComment string         `json:"columnComment"` // 列备注
	IsPrimaryKey  bool           `json:"isPrimaryKey"`  // 是否为主键
	IsIdentity    bool           `json:"isIdentity"`    // 是否自增
	ColumnDefault string         `json:"columnDefault"` // 默认值
	Nullable      string         `json:"nullable"`      // 是否可为null
	CharMaxLength int            `json:"charMaxLength"` // 字符最大长度
	NumPrecision  int            `json:"numPrecision"`  // 精度(总数字位数)
	NumScale      int            `json:"numScale"`      // 小数点位数
	Extra         collx.M        `json:"extra"`         // 其他额外信息

	ShowLength   int    `json:"showLength"`
	ShowScale    int    `json:"showScale"`
	ShowDataType string `json:"showDataType"` // 显示数据类型
}

// 初始化列显示类型，拼接数据类型与长度等。如varchar(2000)，decimal(20,2)
func (c *Column) InitShowNum() string {
	if c.CharMaxLength > 0 {
		c.ShowDataType = fmt.Sprintf("%s(%d)", c.DataType, c.CharMaxLength)
		c.ShowLength = c.CharMaxLength
		c.ShowScale = 0
		return c.ShowDataType
	}
	if c.NumPrecision > 0 {
		if c.NumScale > 0 {
			c.ShowDataType = fmt.Sprintf("%s(%d,%d)", c.DataType, c.NumPrecision, c.NumScale)
			c.ShowScale = c.NumScale
		} else {
			c.ShowDataType = fmt.Sprintf("%s(%d)", c.DataType, c.NumPrecision)
			c.ShowScale = 0
		}
		c.ShowLength = c.NumPrecision

		return c.ShowDataType
	}

	c.ShowDataType = string(c.DataType)
	c.ShowLength = 0
	c.ShowScale = 0

	return c.ShowDataType
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

type ColumnDataType string

const (
	CommonTypeVarchar    ColumnDataType = "varchar"
	CommonTypeChar       ColumnDataType = "char"
	CommonTypeText       ColumnDataType = "text"
	CommonTypeBlob       ColumnDataType = "blob"
	CommonTypeLongblob   ColumnDataType = "longblob"
	CommonTypeLongtext   ColumnDataType = "longtext"
	CommonTypeBinary     ColumnDataType = "binary"
	CommonTypeMediumblob ColumnDataType = "mediumblob"
	CommonTypeMediumtext ColumnDataType = "mediumtext"
	CommonTypeVarbinary  ColumnDataType = "varbinary"

	CommonTypeInt      ColumnDataType = "int"
	CommonTypeSmallint ColumnDataType = "smallint"
	CommonTypeTinyint  ColumnDataType = "tinyint"
	CommonTypeNumber   ColumnDataType = "number"
	CommonTypeBigint   ColumnDataType = "bigint"

	CommonTypeDatetime  ColumnDataType = "datetime"
	CommonTypeDate      ColumnDataType = "date"
	CommonTypeTime      ColumnDataType = "time"
	CommonTypeTimestamp ColumnDataType = "timestamp"

	CommonTypeEnum ColumnDataType = "enum"
	CommonTypeJSON ColumnDataType = "json"
)

type DataType string

const (
	DataTypeString   DataType = "string"
	DataTypeNumber   DataType = "number"
	DataTypeDate     DataType = "date"
	DataTypeTime     DataType = "time"
	DataTypeDateTime DataType = "datetime"
)

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
