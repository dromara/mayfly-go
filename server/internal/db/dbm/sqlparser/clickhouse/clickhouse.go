package clickhouse

import (
	"fmt"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/pkg/logx"
)

// ClickhouseParser ClickHouse 解析器入口（实现 SQLParser 与可选能力接口）
type ClickhouseParser struct {
}

func (*ClickhouseParser) Parse(stmt string) (result sqlstmt.Stmt, err error) {
	defer func() {
		if e := recover(); e != nil {
			// panic不得静默吞掉：返回错误让调用方能感知解析失败
			logx.ErrorTrace("clickhouse sql parser err: ", e)
			err = fmt.Errorf("clickhouse sql parse failed: %v", e)
		}
	}()
	return NewParser(stmt).Parse()
}

// RewritePagination ClickHouse 分页改写：LIMIT count OFFSET offset
func (p *ClickhouseParser) RewritePagination(sql string, offset, limit int64) (string, error) {
	return sqlstmt.DefaultRewritePagination(sql, offset, limit), nil
}

// ClassifyStmt 基于 AST 判定语句类型，通用逻辑（含 nil AST 兜底）收敛于 sqlstmt.ClassifyByParse
func (p *ClickhouseParser) ClassifyStmt(sql string) (sqlstmt.StmtType, error) {
	return sqlstmt.ClassifyByParse(p.Parse, sql)
}
