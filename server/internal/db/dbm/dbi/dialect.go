package dbi

import (
	"database/sql"
)

type DataType string

const (
	DataTypeString   DataType = "string"
	DataTypeNumber   DataType = "number"
	DataTypeDate     DataType = "date"
	DataTypeTime     DataType = "time"
	DataTypeDateTime DataType = "datetime"
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
// 数据库方言 用于获取元信息接口、批量插入等各个数据库方言不一致的实现方式
type Dialect interface {

	// 获取元数据信息接口
	GetMetaData() MetaData

	// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
	GetDbProgram() (DbProgram, error)

	// 批量保存数据
	BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error)

	// 获取数据转换器用于解析格式化列数据等
	GetDataConverter() DataConverter

	CopyTable(copy *DbCopyTable) error
}
