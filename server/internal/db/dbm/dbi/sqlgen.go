package dbi

import (
	"fmt"
	"strings"
)

// SQLGenerator 各方言不一致的 SQL 生成契约：建表/索引 DDL、插入、清空、批量删除。
type SQLGenerator interface {
	// GenTableDDL 生成建表语句
	GenTableDDL(table Table, columns []Column, dropBeforeCreate bool) []string

	// GenIndexDDL 生成索引语句
	GenIndexDDL(table Table, indexes []Index) []string

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
	// 用于删除对账：删除目标中不属于完整保留集合的行，不能拆成多个独立 NOT IN(batch)。
	// 超出内联集合资源预算的场景需要暂存表反连接等方案，由调用方明确拒绝或选择能力。
	// 各方言差异：
	//   - MySQL：DELETE FROM t WHERE (pk) NOT IN ((v1),(v2),...)
	//   - PostgreSQL：DELETE FROM t WHERE pk NOT IN (v1,v2,...)
	//   - MSSQL：单键 NOT IN；复合键 VALUES + NOT EXISTS（使用内联字面量，不受绑定参数数限制）
	//   - Oracle/达梦：DELETE FROM t WHERE pk NOT IN (SELECT v FROM dual UNION ALL ...)
	//   - SQLite：DELETE FROM t WHERE pk NOT IN (v1,v2,...)
	//   - ClickHouse：ALTER TABLE t DELETE WHERE (pk) NOT IN (v1,v2,...)
	GenBatchDelete(tableName string, keyColumns []string, keyValues [][]any, targetTableMeta *TargetTableMeta) []string
}

// ParamInserter 目标方言的「占位符批量插入」可选能力（由方言 SQLGenerator 实现，复用其标识符引用）。
//
// 具备本能力的方言，调用方可走「行值 → 占位符绑定参数」的插入路径，取代「把值序列化为 SQL 文本字面量、
// 再由目标端解析回放」。后者存在转义边界、精度损失与事务控制语句被误判执行的隐患；绑定参数交由驱动处理
// 可规避这些风险。该能力本身与具体用例无关（任何批量写入皆可复用），目前由 DB→DB 迁移/同步消费。
//
// 未实现本能力的方言自动回退文本路径——渐进接入，不要求全方言一次到位（开闭原则）。
type ParamInserter interface {
	// GenInsertParams 为一批行生成【参数化】插入语句：返回一条可直接执行的 INSERT 语句
	// （列名按方言语义引用、值以占位符表达）与顺序对应的展平参数。
	//   - columns：已剔除生成列等不可插入列
	//   - rows：已转换为目标可绑定形态的行值批次
	// 实现可自由选择「多值单语句 (..),(..)」或「每行一语句」，只要 stmt 占位符数与 args 长度一致。
	//
	// 契约：仅用于直接插入（无 ignore/upsert 语义，那类仍走 GenInsert 文本路径以保留方言冲突处理）；
	// columns 或 rows 为空时返回 ("", nil, nil)，调用方须前置校验非空或将空 stmt 视为「本批无语句可执行」。
	GenInsertParams(tableName string, columns []Column, rows [][]any) (stmt string, args []any, err error)
}

// GetParamInserter 探测目标方言 SQLGenerator 是否具备参数化批量插入能力，不具备返回 nil。
// 与 sqlparser.GetStatementClassifier 同构：调用方封装类型断言细节，迁移核心据此在
// 「参数化直插」与「文本回放」两条路径间择一，未实现能力的方言自动回退。
func GetParamInserter(gen SQLGenerator) ParamInserter {
	if pi, ok := gen.(ParamInserter); ok {
		return pi
	}
	return nil
}

// DefaultSQLGenerator SQLGenerator 公共基类，提供 GenTruncate 和 GenBatchDelete 的默认实现。
// 各方言 SQLGenerator 嵌入此结构体后，仅需覆写差异方法（如 GenTableDDL/GenInsert）。
type DefaultSQLGenerator struct {
	// QuoterFn 返回当前方言的标识符引用函数。
	// 各方言嵌入后必须覆写此字段（在初始化时赋值），否则使用默认双引号引用。
	QuoterFn func() Quoter
}

// QuoteIdentFn 获取方言引用函数
func (b *DefaultSQLGenerator) QuoteIdentFn() func(string) string {
	if b.QuoterFn != nil {
		return b.QuoterFn().QuoteIdent
	}
	return DefaultQuoter.QuoteIdent
}

// GenTruncate 默认实现：TRUNCATE TABLE（适用于 MySQL/PG/Oracle/DM 等大多数方言）
func (b *DefaultSQLGenerator) GenTruncate(tableName string) []string {
	quote := b.QuoteIdentFn()
	return []string{fmt.Sprintf("TRUNCATE TABLE %s", quote(tableName))}
}

// GenBatchDelete 默认实现：DELETE FROM t WHERE (pk) NOT IN ((v1),(v2),...)
// 适用于 MySQL/PG/SQLite 等标准方言。Oracle/DM/MSSQL/ClickHouse 需覆写。
func (b *DefaultSQLGenerator) GenBatchDelete(tableName string, keyColumns []string, keyValues [][]any, targetTableMeta *TargetTableMeta) []string {
	return BuildBatchDelete("DELETE FROM", tableName, keyColumns, keyValues, b.QuoteIdentFn())
}

// BuildBatchDelete 构建通用的批量删除 SQL。
// deletePrefix 为方言的删除语句前缀（如 "DELETE FROM"）。
// 提取为公共函数，供 DefaultSQLGenerator 默认实现复用。
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
