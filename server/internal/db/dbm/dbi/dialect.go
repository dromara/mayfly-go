package dbi

import (
	"io"
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

// TargetTableMeta 目标表元信息，用于生成 upsert/merge 类插入语句。
// 由调用方（持有真实目标库连接）预先查询好传入，保证 SQLGenerator 为纯函数：
// 生成 SQL 的过程中不做任何数据库查询，避免在伪连接（仅生成SQL）场景下 panic
type TargetTableMeta struct {
	UniqueColumns   []string // 主键与唯一索引涉及到的列名（不含引号）
	IdentityColumns []string // 自增列名（生成merge语句时需排除）
}

// BaseDialect 基础dialect，在DefaultDialect 都有默认的实现方法
type BaseDialect interface {

	// Quoter sql关键字引用处理，如 table -> `table`、table -> "table"
	Quoter() Quoter

	// GetDumpHeler
	GetDumpHelper() DumpHelper

	// GetSQLParser 获取sql解析器
	GetSQLParser() sqlparser.SqlParser

	// GetSQLSplitter 获取sql切割器
	GetSQLSplitter() sqlparser.SQLSplitter
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

func (dd *DefaultDialect) GetDumpHelper() DumpHelper {
	return new(DefaultDumpHelper)
}

// GetSQLParser 获取默认sql解析器。
// dbi为通用层，不得依赖任何具体方言解析器（依赖倒置）：各方言必须显式覆写本方法
// 选择自身解析器（语法相近的方言可复用其它方言解析器，如mssql/sqlite沿用pgsql）；
// 未覆写时返回nil，调用处fail-fast暴露，避免新方言静默用错解析器
func (pd *DefaultDialect) GetSQLParser() sqlparser.SqlParser {
	return nil
}

func (pd *DefaultDialect) GetSQLSplitter() sqlparser.SQLSplitter {
	// 未注册方言的回落语义：标准SQL（反斜杠为普通字符，双引号为标识符引用符）
	return sqlparser.NewSplitter(tokenizer.StdConfig)
}

// DumpHelper 导出辅助方法
type DumpHelper interface {
	BeforeInsert(writer io.Writer, tableName string) error

	// BeforeInsertSql 生成每批insert语句前的前置语句（如mssql/dm的set identity_insert on）
	// - tableName为裸表名，由各方言helper自行quote（避免调用方使用源方言引用符）
	// - columns用于判断表是否含自增列（对无自增列的表set identity_insert会报错）
	BeforeInsertSql(tableName string, columns []Column) string

	AfterInsert(writer io.Writer, tableName string, columns []Column) error
}

type DefaultDumpHelper struct {
}

func (dd *DefaultDumpHelper) BeforeInsert(writer io.Writer, tableName string) error {
	_, err := writer.Write([]byte("BEGIN;\n"))
	return err
}

func (dd *DefaultDumpHelper) BeforeInsertSql(tableName string, columns []Column) string {
	return ""
}

func (dd *DefaultDumpHelper) AfterInsert(writer io.Writer, tableName string, columns []Column) error {
	_, err := writer.Write([]byte("COMMIT;\n"))
	return err
}

type SQLGenerator interface {
	// GenTableDDL 生成建表语句
	GenTableDDL(table Table, columns []Column, dropBeforeCreate bool) []string

	// GenIndexDDL 生成索引语句
	GenIndexDDL(table Table, indexs []Index) []string

	// GenInsert 生成插入语句，返回可直接执行的sql列表
	//  - duplicateStrategy: 重复数据处理策略
	//  - targetTableMeta: 目标表元信息，仅在生成 upsert/merge 语句（DuplicateStrategyUpdate/Ignore 且方言需要）时使用；可为nil，
	//    为nil时若无法生成冲突处理语句则退化为直接插入。生成过程中不会查询数据库
	GenInsert(tableName string, columns []Column, values [][]any, duplicateStrategy int, targetTableMeta *TargetTableMeta) []string
}
