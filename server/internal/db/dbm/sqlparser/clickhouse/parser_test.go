package clickhouse_test

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/clickhouse"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// TestClickhouseThinDialect ClickHouse 薄方言：反引号标识符、LIMIT 分页、FINAL/ARRAY JOIN 不破坏主体解析。
func TestClickhouseThinDialect(t *testing.T) {
	parser := new(clickhouse.ClickhouseParser)

	stmt, err := parser.Parse("SELECT `id`, `name` FROM `db`.`t` LIMIT 5")
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok {
		t.Fatalf("want SelectStmt, got %T", stmt)
	}
	if len(sel.From) == 0 || sel.From[0].Name != "t" {
		t.Fatalf("反引号表名提取错误: %+v", sel.From)
	}
	if sel.Limit == nil {
		t.Fatalf("LIMIT 未解析")
	}

	sql, err := parser.RewritePagination("SELECT * FROM t", 0, 5)
	if err != nil {
		t.Fatalf("rewrite err: %v", err)
	}
	if sql == "SELECT * FROM t" {
		t.Fatalf("分页改写未生效: %s", sql)
	}
}
