// Package sqlparser SQL 解析器框架。
//
// 设计原则：
//   - 核心接口 SqlParser 仅定义 Parse（所有方言必须实现）
//   - 可选能力通过独立接口声明（PaginationRewriter / StatementClassifier）
//   - 调用方通过类型断言检测能力，实现开闭原则：
//     新增能力不修改现有接口/实现，仅新增接口 + 按需实现
//   - 便捷函数（如 GetPaginationRewriter）封装类型断言逻辑，返回安全零值
package sqlparser

import (
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== 核心接口 ==========

// SqlParser SQL 解析器核心接口。
// 所有方言必须实现此接口，提供基础的 SQL 解析能力。
type SqlParser interface {
	// Parse 解析单条 SQL 语句为 AST
	Parse(stmt string) (sqlstmt.Stmt, error)
}

// ========== 可选能力接口（开闭原则：新增能力不影响现有代码）==========

// PaginationRewriter 分页改写能力（可选）。
// 支持此能力的方言可生成方言特定的分页 SQL。
type PaginationRewriter interface {
	// RewritePagination 将原始 SELECT 改写为带分页限制的 SQL。
	// offset < 0 表示不跳过行，limit <= 0 表示不限制行数。
	RewritePagination(sql string, offset, limit int64) (string, error)
}

// StatementClassifier 语句分类能力（可选）。
// 支持此能力的方言可基于 AST 准确判定语句类型。
type StatementClassifier interface {
	// ClassifyStmt 基于解析结果判定语句类型
	ClassifyStmt(sql string) (sqlstmt.StmtType, error)
}

// ========== 便捷函数（封装类型断言，调用方无需关心断言逻辑）==========

// GetPaginationRewriter 尝试从 SqlParser 获取分页改写能力。
// 若 parser 不支持此能力，返回 nil（调用方应做 nil 检查并回退到默认实现）。
func GetPaginationRewriter(parser SqlParser) PaginationRewriter {
	if pw, ok := parser.(PaginationRewriter); ok {
		return pw
	}
	return nil
}

// GetStatementClassifier 尝试从 SqlParser 获取语句分类能力。
// 若 parser 不支持此能力，返回 nil。
func GetStatementClassifier(parser SqlParser) StatementClassifier {
	if sc, ok := parser.(StatementClassifier); ok {
		return sc
	}
	return nil
}

// ========== 向后兼容别名 ==========

// StmtType 别名：保留向后兼容，实际类型定义在 sqlstmt 包
type StmtType = sqlstmt.StmtType

// StmtType 常量别名
const (
	StmtTypeSelect = sqlstmt.StmtTypeSelect
	StmtTypeInsert = sqlstmt.StmtTypeInsert
	StmtTypeUpdate = sqlstmt.StmtTypeUpdate
	StmtTypeDelete = sqlstmt.StmtTypeDelete
	StmtTypeDDL    = sqlstmt.StmtTypeDDL
	StmtTypeOther  = sqlstmt.StmtTypeOther
)

// DefaultRewritePagination 默认分页改写实现（LIMIT/OFFSET 语法）。
// 供不支持 PaginationRewriter 的调用方作为回退使用。
func DefaultRewritePagination(sql string, offset, limit int64) string {
	return sqlstmt.DefaultRewritePagination(sql, offset, limit)
}
