package dbi

import (
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
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

// SQLDialect SQL 文本层方言契约：标识符引用、解析/切割器、导出辅助等纯语法相关能力。
// Dialect 在此基础上再叠加表级操作（CopyTable、GetSQLGenerator）。
type SQLDialect interface {

	// Quoter sql关键字引用处理，如 table -> `table`、table -> "table"
	Quoter() Quoter

	// GetDumpTxnWrapper 导出辅助（各库 dump 包装差异）
	GetDumpTxnWrapper() DumpTxnWrapper

	// GetSQLParser 获取sql解析器
	GetSQLParser() sqlparser.SQLParser

	// GetSQLSplitter 获取sql切割器
	GetSQLSplitter() sqlparser.SQLSplitter
}

// Dialect 数据库方言：SQL 文本层能力（SQLDialect）+ 表级 SQL 生成/拷贝。
// 用于生成 sql、批量插入等各个数据库方言不一致的实现方式
type Dialect interface {
	SQLDialect

	// CopyTable 拷贝表
	CopyTable(copy *DbCopyTable) error

	// GetSQLGenerator 获取sql生成器
	GetSQLGenerator() SQLGenerator
}

// DefaultDialect SQLDialect 默认实现，若需要覆盖则由各个数据库 dialect 实现去覆盖重写
type DefaultDialect struct {
}

var _ (SQLDialect) = (*DefaultDialect)(nil)

func (dx *DefaultDialect) Quoter() Quoter {
	return DefaultQuoter
}

func (dd *DefaultDialect) GetDumpTxnWrapper() DumpTxnWrapper {
	return new(DefaultDumpTxnWrapper)
}

// GetSQLParser 获取默认sql解析器。
// dbi为通用层，不得依赖任何具体方言解析器（依赖倒置）：各方言必须显式覆写本方法
// 选择自身解析器（语法相近的方言可复用其它方言解析器，如mssql/sqlite沿用pgsql）；
// 未覆写时返回nil，调用处fail-fast暴露，避免新方言静默用错解析器
//
// 返回的 SQLParser 可能同时实现可选能力接口（PaginationRewriter / StatementClassifier），
// 调用方可通过 sqlparser.GetPaginationRewriter / GetStatementClassifier 检测。
func (pd *DefaultDialect) GetSQLParser() sqlparser.SQLParser {
	return nil
}

func (pd *DefaultDialect) GetSQLSplitter() sqlparser.SQLSplitter {
	// 未注册方言的回落语义：标准SQL（反斜杠为普通字符，双引号为标识符引用符）
	return sqlparser.NewSplitter(tokenizer.StdConfig)
}
