package dbi

import (
	"database/sql"
)

const (
	CommonTypeVarchar    string = "varchar"
	CommonTypeChar       string = "char"
	CommonTypeText       string = "text"
	CommonTypeBlob       string = "blob"
	CommonTypeLongblob   string = "longblob"
	CommonTypeLongtext   string = "longtext"
	CommonTypeBinary     string = "binary"
	CommonTypeMediumblob string = "mediumblob"
	CommonTypeMediumtext string = "mediumtext"
	CommonTypeVarbinary  string = "varbinary"

	CommonTypeInt      string = "int"
	CommonTypeSmallint string = "smallint"
	CommonTypeTinyint  string = "tinyint"
	CommonTypeNumber   string = "number"
	CommonTypeBigint   string = "bigint"

	CommonTypeDatetime  string = "datetime"
	CommonTypeDate      string = "date"
	CommonTypeTime      string = "time"
	CommonTypeTimestamp string = "timestamp"

	CommonTypeEnum string = "enum"
	CommonTypeJSON string = "json"
)

const (
	// -1. 无操作
	DuplicateStrategyNone = -1
	// 1. 忽略
	DuplicateStrategyIgnore = 1
	// 2. 更新
	DuplicateStrategyUpdate = 2
)

type DbCopyTable struct {
	Id        uint64 `json:"id"`
	Db        string `json:"db" `
	TableName string `json:"tableName"`
	CopyData  bool   `json:"copyData"` // 是否复制数据
}

// -----------------------------------元数据接口定义------------------------------------------
// 数据库方言 用于获取元信息接口、批量插入等各个数据库方言不一致的实现方式
type Dialect interface {

	// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
	GetDbProgram() (DbProgram, error)

	// 批量保存数据
	BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error)

	// 拷贝表
	CopyTable(copy *DbCopyTable) error

	CreateTable(commonColumns []Column, tableInfo Table, dropOldTable bool) (int, error)

	CreateIndex(tableInfo Table, indexs []Index) error

	// 把方言类型转换为通用类型
	TransColumns(columns []Column) []Column

	// 有些数据库迁移完数据之后，需要更新表自增序列为当前表最大值
	UpdateSequence(tableName string, columns []Column)
}
