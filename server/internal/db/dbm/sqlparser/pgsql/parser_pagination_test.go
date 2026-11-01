package pgsql

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== 简单分页测试 ==========

func TestPgPaginationSimple(t *testing.T) {
	sql := "SELECT * FROM users LIMIT 10 OFFSET 0"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	t.Logf("LIMIT text: %s", selectStmt.Limit.Text)
	t.Logf("Offset: %d, Count: %d", selectStmt.Limit.Offset, selectStmt.Limit.Count)

	if selectStmt.Limit.Offset != 0 {
		t.Errorf("expected offset=0, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 10 {
		t.Errorf("expected count=10, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationWithOffset(t *testing.T) {
	sql := "SELECT id, name FROM users LIMIT 10 OFFSET 20"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 20 {
		t.Errorf("expected offset=20, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 10 {
		t.Errorf("expected count=10, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationOffsetFirst(t *testing.T) {
	// PostgreSQL 支持 OFFSET 在 LIMIT 前面
	sql := "SELECT * FROM products OFFSET 30 LIMIT 15"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	t.Logf("LIMIT text: %s", selectStmt.Limit.Text)
	t.Logf("Offset: %d, Count: %d", selectStmt.Limit.Offset, selectStmt.Limit.Count)

	if selectStmt.Limit.Offset != 30 {
		t.Errorf("expected offset=30, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 15 {
		t.Errorf("expected count=15, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationOnlyLimit(t *testing.T) {
	sql := "SELECT * FROM orders LIMIT 50"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 0 {
		t.Errorf("expected offset=0, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 50 {
		t.Errorf("expected count=50, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationOnlyOffset(t *testing.T) {
	sql := "SELECT * FROM users OFFSET 100"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 100 {
		t.Errorf("expected offset=100, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 0 {
		t.Errorf("expected count=0, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationLimitAll(t *testing.T) {
	// PostgreSQL 特有的 LIMIT ALL
	sql := "SELECT * FROM users OFFSET 50 LIMIT ALL"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	t.Logf("LIMIT text: %s", selectStmt.Limit.Text)
	if selectStmt.Limit.Offset != 50 {
		t.Errorf("expected offset=50, got %d", selectStmt.Limit.Offset)
	}
}

// ========== 复杂分页测试 ==========

func TestPgPaginationWithWhere(t *testing.T) {
	sql := "SELECT id, name, email FROM users WHERE status = 1 AND age > 18 LIMIT 20 OFFSET 10"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 10 {
		t.Errorf("expected offset=10, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 20 {
		t.Errorf("expected count=20, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationWithOrderBy(t *testing.T) {
	sql := "SELECT * FROM users ORDER BY created_at DESC, id ASC LIMIT 100 OFFSET 0"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.OrderBy) != 2 {
		t.Fatalf("expected 2 ORDER BY clauses")
	}
	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 0 {
		t.Errorf("expected offset=0, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 100 {
		t.Errorf("expected count=100, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationWithWhereOrderBy(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE status = 'active' ORDER BY score DESC LIMIT 10 OFFSET 50"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Where == nil || selectStmt.Where.Text != "status = 'active'" {
		t.Errorf("expected WHERE='status = 'active''")
	}
	if len(selectStmt.OrderBy) != 1 || selectStmt.OrderBy[0].Text != "score" {
		t.Errorf("expected ORDER BY score")
	}
	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 50 {
		t.Errorf("expected offset=50, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 10 {
		t.Errorf("expected count=10, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationWithJoin(t *testing.T) {
	sql := "SELECT u.id, u.name, o.amount FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.status = 1 ORDER BY o.created_at DESC LIMIT 20 OFFSET 100"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Joins) != 1 {
		t.Fatal("expected 1 JOIN")
	}
	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 100 {
		t.Errorf("expected offset=100, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 20 {
		t.Errorf("expected count=20, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationWithGroupBy(t *testing.T) {
	sql := "SELECT user_id, COUNT(*) as order_count FROM orders GROUP BY user_id ORDER BY order_count DESC LIMIT 10 OFFSET 0"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 0 {
		t.Errorf("expected offset=0, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 10 {
		t.Errorf("expected count=10, got %d", selectStmt.Limit.Count)
	}
}

// ========== UNION 分页测试 ==========

func TestPgPaginationWithUnion(t *testing.T) {
	sql := "SELECT id FROM users UNION SELECT id FROM admins ORDER BY 1 DESC LIMIT 50 OFFSET 0"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Unions) != 1 {
		t.Fatal("expected 1 UNION")
	}
	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause for UNION")
	}
	if selectStmt.Limit.Offset != 0 {
		t.Errorf("expected offset=0, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 50 {
		t.Errorf("expected count=50, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationUnionAll(t *testing.T) {
	sql := "SELECT name FROM products UNION ALL SELECT name FROM services LIMIT 30 OFFSET 10"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Unions) != 1 {
		t.Fatal("expected 1 UNION ALL")
	}
	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 10 {
		t.Errorf("expected offset=10, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 30 {
		t.Errorf("expected count=30, got %d", selectStmt.Limit.Count)
	}
}

func TestPgMultipleUnionsPagination(t *testing.T) {
	sql := "SELECT id FROM t1 UNION SELECT id FROM t2 UNION SELECT id FROM t3 ORDER BY 1 LIMIT 100 OFFSET 50"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Unions) != 2 {
		t.Fatalf("expected 2 UNIONs, got %d", len(selectStmt.Unions))
	}
	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 50 {
		t.Errorf("expected offset=50, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 100 {
		t.Errorf("expected count=100, got %d", selectStmt.Limit.Count)
	}
}

// ========== 子查询分页测试 ==========

func TestPgPaginationWithSubquery(t *testing.T) {
	sql := "SELECT * FROM (SELECT id, name FROM users WHERE status = 1 ORDER BY id LIMIT 100 OFFSET 0) AS tmp LIMIT 10 OFFSET 20"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.From) != 1 {
		t.Fatal("expected 1 FROM table")
	}
	if selectStmt.Limit == nil {
		t.Fatal("expected outer LIMIT clause")
	}
	if selectStmt.Limit.Offset != 20 {
		t.Errorf("expected outer offset=20, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 10 {
		t.Errorf("expected outer count=10, got %d", selectStmt.Limit.Count)
	}
}

func TestPgNestedPagination(t *testing.T) {
	// 嵌套分页查询
	sql := "SELECT * FROM (SELECT * FROM users LIMIT 100 OFFSET 0) AS tmp LIMIT 5 OFFSET 10"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected outer LIMIT clause")
	}
	if selectStmt.Limit.Offset != 10 {
		t.Errorf("expected outer offset=10, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 5 {
		t.Errorf("expected outer count=5, got %d", selectStmt.Limit.Count)
	}
}

func TestPgPaginationWithExists(t *testing.T) {
	sql := "SELECT * FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id LIMIT 1) LIMIT 20 OFFSET 0"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 0 {
		t.Errorf("expected offset=0, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 20 {
		t.Errorf("expected count=20, got %d", selectStmt.Limit.Count)
	}
}

// ========== 大数据量分页测试 ==========

func TestPgLargeOffsetPagination(t *testing.T) {
	sql := "SELECT * FROM logs ORDER BY id LIMIT 100 OFFSET 1000000"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 1000000 {
		t.Errorf("expected offset=1000000, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 100 {
		t.Errorf("expected count=100, got %d", selectStmt.Limit.Count)
	}
}

// ========== FOR UPDATE 分页测试 ==========

func TestPgPaginationForUpdate(t *testing.T) {
	sql := "SELECT * FROM users WHERE status = 0 ORDER BY id LIMIT 10 OFFSET 0 FOR UPDATE"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
	if selectStmt.Limit.Offset != 0 {
		t.Errorf("expected offset=0, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 10 {
		t.Errorf("expected count=10, got %d", selectStmt.Limit.Count)
	}
}
