package dbi

import (
	"database/sql"
	"errors"
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

	// BatchInsert 批量insert数据
	BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error)

	// CopyTable 拷贝表
	CopyTable(copy *DbCopyTable) error

	// CreateTable 创建表
	CreateTable(columns []Column, tableInfo Table, dropOldTable bool) (int, error)

	// CreateIndex 创建索引
	CreateIndex(tableInfo Table, indexs []Index) error

	// UpdateSequence 有些数据库迁移完数据之后，需要更新表自增序列为当前表最大值
	UpdateSequence(tableName string, columns []Column)
}

type DefaultDialect struct {
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (dd *DefaultDialect) GetDbProgram() (DbProgram, error) {
	return nil, errors.New("not support db program")
}

func (dd *DefaultDialect) UpdateSequence(tableName string, columns []Column) {}
