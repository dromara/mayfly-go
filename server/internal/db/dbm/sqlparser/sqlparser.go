// Package sqlparser SQL 解析器框架。
//
// 设计原则：
//   - SQLParser 是**唯一**入口：解析、语句分类、分页改写三者都是必需能力，
//     不存在「某方言没实现就回退默认语法」的路径——回退会在 Oracle/MSSQL 上拼出
//     非法的 LIMIT 语法，且失败是静默的。
//   - 7 个方言都实现同一接口，编译期由各 dialect 包的 `var _ sqlparser.SQLParser` 断言兜住，
//     新增方言不实现就直接编译失败，而不是运行期退化成错误 SQL。
package sqlparser

import (
	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// SQLParser SQL 解析器接口（所有方言必须完整实现）。
type SQLParser interface {
	// Parse 解析单条 SQL 语句为 AST。
	//
	// 历史约定：分页改写与语句分类曾是「可选能力 + 类型断言 + 默认回退」，
	// 而 7 个方言实际全部实现，回退分支只在漏实现时被触发——那时生成的
	// LIMIT/OFFSET 在 Oracle/MSSQL 上是非法 SQL，属于静默失败。现统一为必需方法。
	Parse(stmt string) (sqlstmt.Stmt, error)

	// RewritePagination 将原始 SELECT 改写为带分页限制的 SQL（方言各自语法）。
	// offset < 0 表示不跳过行，limit <= 0 表示不限制行数。
	RewritePagination(sql string, offset, limit int64) (string, error)

	// ClassifyStmt 基于解析结果判定语句类型。
	ClassifyStmt(sql string) (sqlstmt.StmtType, error)
}

// PolicyStatementTypes 触发策略「语句类型」的可选值，来自 base 的语句注册表。
//
// 表驱动：新增语句类型只要在 base/sharedStatementSpecs 填好 PolicyType 就自动出现，
// 上层（策略选项、前端下拉）不再各自维护一份数组。
func PolicyStatementTypes() []string {
	return base.PolicyStatementTypes()
}

// SharedStatementNames 共享语句注册表里的语句名（顺序即分发优先级），供自测与文档。
func SharedStatementNames() []string {
	return base.SharedStatementNames()
}
