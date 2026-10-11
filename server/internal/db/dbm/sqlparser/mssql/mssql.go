package mssql

import (
	"fmt"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/pkg/logx"
)

// MssqlParser SQL Server 解析器入口。
//
// 语法骨架与各方言共用 base（见 parser.go），本包装只提供 SQL Server 特有的能力面：
// 分页改写用 OFFSET-FETCH 语法。曾直接包装 pgsql 解析器，使 T-SQL 的 [方括号] 标识符
// 在解析阶段不被识别为引用名（词法能力表用的是 pg 配置），表名与列名提取因此失真
type MssqlParser struct {
}

// NewParser 构造 MssqlParser
func NewParser() *MssqlParser {
	return new(MssqlParser)
}

func (*MssqlParser) Parse(stmt string) (result sqlstmt.Stmt, err error) {
	defer func() {
		if e := recover(); e != nil {
			// panic不得静默吞掉：返回错误让调用方能感知解析失败
			logx.ErrorTrace("mssql sql parser err: ", e)
			err = fmt.Errorf("mssql sql parse failed: %v", e)
		}
	}()
	return newParser(stmt).Parse()
}

// RewritePagination SQL Server 分页改写：OFFSET-FETCH 语法（要求 ORDER BY，缺失时由实现补占位排序）
func (p *MssqlParser) RewritePagination(sql string, offset, limit int64) (string, error) {
	return sqlstmt.MssqlRewritePagination(sql, offset, limit), nil
}

// ClassifyStmt 基于 AST 判定语句类型，通用逻辑（含 nil AST 兜底）收敛于 sqlstmt.ClassifyByParse
func (p *MssqlParser) ClassifyStmt(sql string) (sqlstmt.StmtType, error) {
	return sqlstmt.ClassifyByParse(p.Parse, sql)
}
