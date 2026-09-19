package dbi

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
	"strings"
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
//
// 返回的 SqlParser 可能同时实现可选能力接口（PaginationRewriter / StatementClassifier），
// 调用方可通过 sqlparser.GetPaginationRewriter / GetStatementClassifier 检测。
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

	// GenTruncate 生成清空表语句。
	// 各方言差异：
	//   - MySQL/PG/MSSQL/Oracle/达梦：TRUNCATE TABLE
	//   - SQLite：无 TRUNCATE，退化为 DELETE FROM
	//   - ClickHouse：不支持 TRUNCATE，返回空切片并记录日志告警
	GenTruncate(tableName string) []string

	// GenBatchDelete 生成按主键/唯一键批量删除语句（删除 NOT IN 给定值的记录）。
	// 用于硬删除/软删除同步：源库不存在但目标库存在的记录需删除。
	// 各方言差异：
	//   - MySQL：DELETE FROM t WHERE (pk) NOT IN ((v1),(v2),...)
	//   - PostgreSQL：DELETE FROM t WHERE pk NOT IN (v1,v2,...)
	//   - MSSQL：DELETE FROM t WHERE ... NOT IN ... (参数上限 2100 需分批)
	//   - Oracle/达梦：DELETE FROM t WHERE pk NOT IN (SELECT v FROM dual UNION ALL ...)
	//   - SQLite：DELETE FROM t WHERE pk NOT IN (v1,v2,...)
	//   - ClickHouse：ALTER TABLE t DELETE WHERE (pk) NOT IN (v1,v2,...)
	GenBatchDelete(tableName string, keyColumns []string, keyValues [][]any, targetTableMeta *TargetTableMeta) []string
}

// BaseSQLGenerator SQLGenerator 公共基类，提供 GenTruncate 和 GenBatchDelete 的默认实现。
// 各方言 SQLGenerator 嵌入此结构体后，仅需覆写差异方法（如 GenTableDDL/GenInsert）。
type BaseSQLGenerator struct {
	// QuoterFn 返回当前方言的标识符引用函数。
	// 各方言嵌入后必须覆写此字段（在初始化时赋值），否则使用默认双引号引用。
	QuoterFn func() Quoter
}

// GetQuoter 获取方言引用函数
func (b *BaseSQLGenerator) GetQuoter() func(string) string {
	if b.QuoterFn != nil {
		return b.QuoterFn().QuoteIdent
	}
	return DefaultQuoter.QuoteIdent
}

// GenTruncate 默认实现：TRUNCATE TABLE（适用于 MySQL/PG/Oracle/DM 等大多数方言）
func (b *BaseSQLGenerator) GenTruncate(tableName string) []string {
	quote := b.GetQuoter()
	return []string{fmt.Sprintf("TRUNCATE TABLE %s", quote(tableName))}
}

// GenBatchDelete 默认实现：DELETE FROM t WHERE (pk) NOT IN ((v1),(v2),...)
// 适用于 MySQL/PG/SQLite 等标准方言。Oracle/DM/MSSQL/ClickHouse 需覆写。
func (b *BaseSQLGenerator) GenBatchDelete(tableName string, keyColumns []string, keyValues [][]any, targetTableMeta *TargetTableMeta) []string {
	return BuildBatchDelete("DELETE FROM", tableName, keyColumns, keyValues, b.GetQuoter())
}

// BuildBatchDelete 构建通用的批量删除 SQL。
// deletePrefix 为方言的删除语句前缀（如 "DELETE FROM"）。
// 提取为公共函数，供 BaseSQLGenerator 默认实现复用。
func BuildBatchDelete(deletePrefix string, tableName string, keyColumns []string, keyValues [][]any, quote func(string) string) []string {
	if len(keyColumns) == 0 || len(keyValues) == 0 {
		return nil
	}
	tuples := buildBatchDeleteTuples(keyColumns, keyValues)
	where := BuildBatchDeleteWhere(keyColumns, tuples, quote)
	return []string{fmt.Sprintf("%s %s WHERE %s", deletePrefix, quote(tableName), where)}
}

// BuildBatchDeleteWhere 构建批量删除的 WHERE 子句（col(s) NOT IN (tuples)）。
// 供 BuildBatchDelete 与特殊方言（如 ClickHouse ALTER TABLE DELETE）复用。
func BuildBatchDeleteWhere(keyColumns []string, tuples []string, quote func(string) string) string {
	if len(keyColumns) == 1 {
		return fmt.Sprintf("%s NOT IN (%s)", quote(keyColumns[0]), strings.Join(tuples, ", "))
	}
	quotedCols := make([]string, len(keyColumns))
	for i, col := range keyColumns {
		quotedCols[i] = quote(col)
	}
	return fmt.Sprintf("(%s) NOT IN (%s)", strings.Join(quotedCols, ", "), strings.Join(tuples, ", "))
}

// buildBatchDeleteTuples 构建值元组列表（内部辅助函数）
func buildBatchDeleteTuples(keyColumns []string, keyValues [][]any) []string {
	tuples := make([]string, 0, len(keyValues))
	for _, row := range keyValues {
		vals := make([]string, 0, len(keyColumns))
		for _, v := range row {
			vals = append(vals, SQLValueString(v))
		}
		if len(keyColumns) == 1 {
			tuples = append(tuples, vals[0])
		} else {
			tuples = append(tuples, fmt.Sprintf("(%s)", strings.Join(vals, ", ")))
		}
	}
	return tuples
}

// BuildBatchDeleteSubQuery 构建「SELECT v1 col1, v2 col2 FROM dual UNION ALL ...」子查询。
// 供 Oracle/DM 等使用 dual 语法的方言复用，避免重复代码。
func BuildBatchDeleteSubQuery(keyColumns []string, keyValues [][]any, quote func(string) string) string {
	selects := make([]string, 0, len(keyValues))
	for _, row := range keyValues {
		vals := make([]string, 0, len(keyColumns))
		for i, v := range row {
			vals = append(vals, fmt.Sprintf("%s %s", SQLValueString(v), quote(keyColumns[i])))
		}
		selects = append(selects, fmt.Sprintf("SELECT %s FROM dual", strings.Join(vals, ", ")))
	}
	return strings.Join(selects, " UNION ALL ")
}
