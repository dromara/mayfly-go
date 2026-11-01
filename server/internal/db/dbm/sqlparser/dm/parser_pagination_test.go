package dm

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== 简单分页测试 ==========

func TestDmPaginationSimple(t *testing.T) {
	sql := "SELECT * FROM users LIMIT 0, 10"
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

func TestDmPaginationWithOffset(t *testing.T) {
	sql := "SELECT id, name FROM users LIMIT 20, 10"
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

func TestDmPaginationKeywordOffset(t *testing.T) {
	// DM 也支持 LIMIT count OFFSET offset
	sql := "SELECT * FROM products LIMIT 15 OFFSET 30"
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

func TestDmPaginationOnlyLimit(t *testing.T) {
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

// ========== SELECT TOP 分页测试 ==========

func TestDmPaginationTopN(t *testing.T) {
	// DM 支持 SELECT TOP n
	sql := "SELECT TOP 10 id, name FROM users ORDER BY id"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Items) != 2 {
		t.Fatalf("expected 2 items")
	}
	t.Logf("SELECT TOP parsed successfully")
}

func TestDmPaginationTopWithWhere(t *testing.T) {
	sql := "SELECT TOP 20 id, name FROM users WHERE status = 1 ORDER BY created_at DESC"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	t.Logf("SELECT TOP with WHERE parsed successfully")
}

// ========== 复杂分页测试 ==========

func TestDmPaginationWithWhere(t *testing.T) {
	sql := "SELECT id, name, email FROM users WHERE status = 1 AND age > 18 LIMIT 10, 20"
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

func TestDmPaginationWithOrderBy(t *testing.T) {
	sql := "SELECT * FROM users ORDER BY created_at DESC, id ASC LIMIT 0, 100"
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

func TestDmPaginationWithWhereOrderBy(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE status = 'active' ORDER BY score DESC LIMIT 50, 10"
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

func TestDmPaginationWithJoin(t *testing.T) {
	sql := "SELECT u.id, u.name, o.amount FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.status = 1 ORDER BY o.created_at DESC LIMIT 100, 20"
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

func TestDmPaginationWithGroupBy(t *testing.T) {
	sql := "SELECT user_id, COUNT(*) as order_count FROM orders GROUP BY user_id ORDER BY order_count DESC LIMIT 0, 10"
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

func TestDmPaginationWithUnion(t *testing.T) {
	sql := "SELECT id FROM users UNION SELECT id FROM admins ORDER BY 1 DESC LIMIT 0, 50"
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

func TestDmPaginationUnionAll(t *testing.T) {
	sql := "SELECT name FROM products UNION ALL SELECT name FROM services LIMIT 10, 30"
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

func TestDmMultipleUnionsPagination(t *testing.T) {
	sql := "SELECT id FROM t1 UNION SELECT id FROM t2 UNION SELECT id FROM t3 ORDER BY 1 LIMIT 50, 100"
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

func TestDmPaginationWithSubquery(t *testing.T) {
	sql := "SELECT * FROM (SELECT id, name FROM users WHERE status = 1 ORDER BY id LIMIT 0, 100) AS tmp LIMIT 20, 10"
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

func TestDmNestedPagination(t *testing.T) {
	// 嵌套分页查询
	sql := "SELECT * FROM (SELECT * FROM users LIMIT 0, 100) AS tmp LIMIT 10, 5"
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

func TestDmPaginationWithExists(t *testing.T) {
	sql := "SELECT * FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id LIMIT 1) LIMIT 20, 10"
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
	if selectStmt.Limit.Offset != 20 {
		t.Errorf("expected offset=20, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 10 {
		t.Errorf("expected count=10, got %d", selectStmt.Limit.Count)
	}
}

// ========== 大数据量分页测试 ==========

func TestDmLargeOffsetPagination(t *testing.T) {
	sql := "SELECT * FROM logs ORDER BY id LIMIT 1000000, 100"
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

func TestDmPaginationForUpdate(t *testing.T) {
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

// ========== 双引号标识符分页测试 ==========

func TestDmPaginationDoubleQuote(t *testing.T) {
	sql := `SELECT "id", "name" FROM "users" WHERE "status" = 1 ORDER BY "id" DESC LIMIT 20, 10`
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
