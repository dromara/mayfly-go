package sqlparser

import (
	"io"
	"mayfly-go/internal/common/utils"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

type DbDialect string

type SqlParser interface {

	// sql解析
	// @param stmt sql语句
	Parse(stmt string) ([]sqlstmt.Stmt, error)
}

// SQLSplit sql切割
func SQLSplit(r io.Reader, callback utils.StmtCallback) error {
	return utils.SplitStmts(r, callback)
}
