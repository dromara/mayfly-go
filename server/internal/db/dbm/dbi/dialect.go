package dbi

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"strings"

	pq "gitee.com/liuzongyang/libpq"
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

	// GetIdentifierQuoteString 用于引用 SQL 标识符（关键字）的字符串
	GetIdentifierQuoteString() string

	// QuoteIdentifier quotes an "identifier" (e.g. a table or a column name) to be
	// used as part of an SQL statement.  For example:
	//
	//	tblname := "my_table"
	//	data := "my_data"
	//	quoted := quoteIdentifier(tblname, '"')
	//	err := db.Exec(fmt.Sprintf("INSERT INTO %s VALUES ($1)", quoted), data)
	//
	// Any double quotes in name will be escaped.  The quoted identifier will be
	// case sensitive when used in a query.  If the input string contains a zero
	// byte, the result will be truncated immediately before it.
	QuoteIdentifier(name string) string

	RemoveQuote(name string) string

	// QuoteEscape 引号转义，多用于sql注释转义，防止拼接sql报错，如： comment xx is '注''释'   最终注释文本为:  注'释
	QuoteEscape(str string) string

	// QuoteLiteral quotes a 'literal' (e.g. a parameter, often used to pass literal
	// to DDL and other statements that do not accept parameters) to be used as part
	// of an SQL statement.  For example:
	//
	//	exp_date := pq.QuoteLiteral("2023-01-05 15:00:00Z")
	//	err := db.Exec(fmt.Sprintf("CREATE ROLE my_user VALID UNTIL %s", exp_date))
	//
	// Any single quotes in name will be escaped. Any backslashes (i.e. "\") will be
	// replaced by two backslashes (i.e. "\\") and the C-style escape identifier
	QuoteLiteral(literal string) string

	// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
	GetDbProgram() (DbProgram, error)

	// GetColumnHelper
	GetColumnHelper() ColumnHelper

	// GetDumpHeler
	GetDumpHelper() DumpHelper

	// GetDataHelper 获取数据处理助手 用于解析格式化列数据等
	GetDataHelper() DataHelper

	// GetSQLParser 获取sql解析器
	GetSQLParser() sqlparser.SqlParser
}

// -----------------------------------元数据接口定义------------------------------------------
// Dialect 数据库方言 用于生成sql、批量插入等各个数据库方言不一致的实现方式
type Dialect interface {
	BaseDialect

	// BatchInsert 批量insert数据
	BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error)

	// CopyTable 拷贝表
	CopyTable(copy *DbCopyTable) error

	// GenerateTableDDL 生成建表ddl
	GenerateTableDDL(columns []Column, tableInfo Table, dropBeforeCreate bool) []string

	// GenerateIndexDDL 生成索引ddl
	GenerateIndexDDL(indexs []Index, tableInfo Table) []string

	// UpdateSequence 有些数据库迁移完数据之后，需要更新表自增序列为当前表最大值
	UpdateSequence(tableName string, columns []Column)
}

// DefaultDialect 默认实现，若需要覆盖，则由各个数据库dialect实现去覆盖重写
type DefaultDialect struct {
}

var _ (BaseDialect) = (*DefaultDialect)(nil)

func (dd *DefaultDialect) GetIdentifierQuoteString() string {
	return `"`
}

func (dx *DefaultDialect) QuoteIdentifier(name string) string {
	quoter := dx.GetIdentifierQuoteString()
	// 兼容mssql
	if quoter == "[" {
		return fmt.Sprintf("[%s]", name)
	}

	end := strings.IndexRune(name, 0)
	if end > -1 {
		name = name[:end]
	}
	return quoter + strings.Replace(name, quoter, quoter+quoter, -1) + quoter
}

func (dx *DefaultDialect) RemoveQuote(name string) string {
	quoter := dx.GetIdentifierQuoteString()

	// 兼容mssql
	if quoter == "[" {
		return strings.Trim(name, "[]")
	}

	return strings.ReplaceAll(name, quoter, "")
}

func (dd *DefaultDialect) QuoteEscape(str string) string {
	return strings.Replace(str, `'`, `''`, -1)
}

func (dd *DefaultDialect) QuoteLiteral(literal string) string {
	return pq.QuoteLiteral(literal)
}

func (dd *DefaultDialect) UpdateSequence(tableName string, columns []Column) {}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (dd *DefaultDialect) GetDbProgram() (DbProgram, error) {
	return nil, errors.New("not support db program")
}

func (dd *DefaultDialect) GetDumpHelper() DumpHelper {
	return new(DefaultDumpHelper)
}

func (dd *DefaultDialect) GetColumnHelper() ColumnHelper {
	return new(DefaultColumnHelper)
}

func (pd *DefaultDialect) GetSQLParser() sqlparser.SqlParser {
	return new(pgsql.PgsqlParser)
}

func (pd *DefaultDialect) GetDataHelper() DataHelper {
	return nil
}

// ColumnHelper 数据库迁移辅助方法
type ColumnHelper interface {
	// ToCommonColumn 数据库方言自带的列转换为公共列
	ToCommonColumn(dialectColumn *Column)

	// ToColumn 公共列转为各个数据库方言自带的列
	ToColumn(commonColumn *Column)

	// FixColumn 根据数据库类型修复字段长度、精度等
	FixColumn(column *Column)
}

type DefaultColumnHelper struct {
}

func (dd *DefaultColumnHelper) ToCommonColumn(dialectColumn *Column) {}

func (dd *DefaultColumnHelper) ToColumn(commonColumn *Column) {}

func (dd *DefaultColumnHelper) FixColumn(column *Column) {}

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
