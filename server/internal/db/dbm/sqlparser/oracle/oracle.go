package oracle

import (
	"fmt"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/pkg/logx"
)

type OracleParser struct {
}

func (*OracleParser) Parse(stmt string) (result sqlstmt.Stmt, err error) {
	defer func() {
		if e := recover(); e != nil {
			// panic不得静默吞掉：返回错误让调用方能感知解析失败
			logx.ErrorTrace("oracle sql parser err: ", e)
			err = fmt.Errorf("oracle sql parse failed: %v", e)
		}
	}()
	return NewParser(stmt).Parse()
}

// RewritePagination Oracle 12c+ 分页改写：OFFSET-FETCH 语法
func (p *OracleParser) RewritePagination(sql string, offset, limit int64) (string, error) {
	return sqlstmt.OracleRewritePagination(sql, offset, limit), nil
}

// ClassifyStmt 基于 AST 判定语句类型
func (p *OracleParser) ClassifyStmt(sql string) (sqlstmt.StmtType, error) {
	stmt, err := p.Parse(sql)
	if err != nil {
		return "", err
	}
	return sqlstmt.DefaultClassifyStmt(stmt.StmtKind()), nil
}
