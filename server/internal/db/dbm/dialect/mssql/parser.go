package mssql

import (
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// MssqlParser 包装 pgsql 解析器，覆写分页改写为 SQL Server 的 OFFSET-FETCH 语法。
type MssqlParser struct {
	inner *pgsql.PgsqlParser
}

func (p *MssqlParser) Parse(stmt string) (sqlstmt.Stmt, error) {
	return p.inner.Parse(stmt)
}

// RewritePagination SQL Server 分页改写：OFFSET-FETCH 语法（需 ORDER BY）
func (p *MssqlParser) RewritePagination(sql string, offset, limit int64) (string, error) {
	return sqlstmt.MssqlRewritePagination(sql, offset, limit), nil
}

// ClassifyStmt 基于 AST 判定语句类型
func (p *MssqlParser) ClassifyStmt(sql string) (sqlstmt.StmtType, error) {
	stmt, err := p.Parse(sql)
	if err != nil {
		return "", err
	}
	return sqlstmt.DefaultClassifyStmt(stmt.StmtKind()), nil
}
