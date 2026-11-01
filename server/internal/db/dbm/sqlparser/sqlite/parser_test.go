package sqlite_test

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlite"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// TestSqliteThinDialect SQLite 薄方言：LIMIT 分页、PRAGMA/SET 类只读导航、INSERT OR REPLACE。
func TestSqliteThinDialect(t *testing.T) {
	parser := new(sqlite.SqliteParser)

	stmt, err := parser.Parse("SELECT * FROM t LIMIT 10 OFFSET 5")
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok || sel.Limit == nil {
		t.Fatalf("LIMIT/OFFSET 未解析: %T %+v", stmt, stmt)
	}

	sql, err := parser.RewritePagination("SELECT * FROM t", 20, 10)
	if err != nil {
		t.Fatalf("rewrite err: %v", err)
	}
	if sql == "SELECT * FROM t" {
		t.Fatalf("分页改写未生效: %s", sql)
	}

	// INSERT OR REPLACE 是写：必须落 insert 类型
	stmt, err = parser.Parse("INSERT OR REPLACE INTO t (id) VALUES (1)")
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if _, ok := stmt.(*sqlstmt.InsertStmt); !ok {
		t.Fatalf("INSERT OR REPLACE 未建模为 InsertStmt: %T", stmt)
	}
}
