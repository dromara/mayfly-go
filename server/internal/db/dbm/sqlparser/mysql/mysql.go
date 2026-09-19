package mysql

import (
	"fmt"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/pkg/logx"
)

type MysqlParser struct {
}

func (*MysqlParser) Parse(stmt string) (result sqlstmt.Stmt, err error) {
	defer func() {
		if e := recover(); e != nil {
			// panic不得静默吞掉：返回错误让调用方能感知解析失败
			logx.ErrorTrace("mysql sql parser err: ", e)
			err = fmt.Errorf("mysql sql parse failed: %v", e)
		}
	}()
	return NewParser(stmt).Parse()
}

// RewritePagination MySQL 分页改写：LIMIT offset, count
func (p *MysqlParser) RewritePagination(sql string, offset, limit int64) (string, error) {
	// MySQL 使用标准 LIMIT/OFFSET 语法
	return sqlstmt.DefaultRewritePagination(sql, offset, limit), nil
}

// ClassifyStmt 基于 AST 判定语句类型，解析失败时回退到文本判定
func (p *MysqlParser) ClassifyStmt(sql string) (sqlstmt.StmtType, error) {
	stmt, err := p.Parse(sql)
	if err != nil {
		return "", err
	}
	return sqlstmt.DefaultClassifyStmt(stmt.StmtKind()), nil
}
