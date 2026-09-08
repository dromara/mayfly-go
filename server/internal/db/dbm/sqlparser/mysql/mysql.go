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
