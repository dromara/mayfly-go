package dm

import (
	"fmt"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/pkg/logx"
)

type DmParser struct {
}

func (*DmParser) Parse(stmt string) (result sqlstmt.Stmt, err error) {
	defer func() {
		if e := recover(); e != nil {
			// panic不得静默吞掉：返回错误让调用方能感知解析失败
			logx.ErrorTrace("dm sql parser err: ", e)
			err = fmt.Errorf("dm sql parse failed: %v", e)
		}
	}()
	return NewParser(stmt).Parse()
}

// RewritePagination 达梦分页改写：与 Oracle 12c+ 兼容，使用 OFFSET-FETCH 语法
func (p *DmParser) RewritePagination(sql string, offset, limit int64) (string, error) {
	return sqlstmt.OracleRewritePagination(sql, offset, limit), nil
}

// ClassifyStmt 基于 AST 判定语句类型，通用逻辑（含 nil AST 兜底）收敛于 sqlstmt.ClassifyByParse
func (p *DmParser) ClassifyStmt(sql string) (sqlstmt.StmtType, error) {
	return sqlstmt.ClassifyByParse(p.Parse, sql)
}
