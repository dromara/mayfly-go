package dbi

import (
	"errors"
	"io"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
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

// BaseDialect 基础dialect，在DefaultDialect 都有默认的实现方法
type BaseDialect interface {

	// Quoter sql关键字引用处理，如 table -> `table`、table -> "table"
	Quoter() Quoter

	// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
	GetDbProgram() (DbProgram, error)

	// GetDumpHeler
	GetDumpHelper() DumpHelper

	// GetSQLParser 获取sql解析器
	GetSQLParser() sqlparser.SqlParser
}

// -----------------------------------元数据接口定义------------------------------------------
// Dialect 数据库方言 用于生成sql、批量插入等各个数据库方言不一致的实现方式
type Dialect interface {
	BaseDialect

	// CopyTable 拷贝表
	CopyTable(copy *DbCopyTable) error

	// GetSQLGenerator 获取sql生成器
	GetSQLGenerator() SQLGenerator
}

// DefaultDialect 默认实现，若需要覆盖，则由各个数据库dialect实现去覆盖重写
type DefaultDialect struct {
}

var _ (BaseDialect) = (*DefaultDialect)(nil)

func (dx *DefaultDialect) Quoter() Quoter {
	return DefaultQuoter
}

func (dd *DefaultDialect) GetDbProgram() (DbProgram, error) {
	return nil, errors.New("not support db program")
}

func (dd *DefaultDialect) GetDumpHelper() DumpHelper {
	return new(DefaultDumpHelper)
}

func (pd *DefaultDialect) GetSQLParser() sqlparser.SqlParser {
	return new(pgsql.PgsqlParser)
}

// DumpHelper 导出辅助方法
type DumpHelper interface {
	BeforeInsert(writer io.Writer, tableName string)

	BeforeInsertSql(quoteSchema string, quoteTableName string) string

	AfterInsert(writer io.Writer, tableName string, columns []Column)
}

type DefaultDumpHelper struct {
}

func (dd *DefaultDumpHelper) BeforeInsert(writer io.Writer, tableName string) {
	writer.Write([]byte("BEGIN;\n"))
}

func (dd *DefaultDumpHelper) BeforeInsertSql(quoteSchema string, quoteTableName string) string {
	return ""
}

func (dd *DefaultDumpHelper) AfterInsert(writer io.Writer, tableName string, columns []Column) {
	writer.Write([]byte("COMMIT;\n"))
}

type SQLGenerator interface {
	// GenTableDDL 生成建表语句
	GenTableDDL(table Table, columns []Column, dropBeforeCreate bool) []string

	// GenIndexDDL 生成索引语句
	GenIndexDDL(table Table, indexs []Index) []string

	// GenInsert 生成插入语句
	GenInsert(tableName string, columns []Column, values [][]any, duplicateStrategy int) []string
}
