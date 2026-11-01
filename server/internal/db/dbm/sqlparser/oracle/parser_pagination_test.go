package oracle

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== FETCH FIRST 分页测试 (Oracle 12c+) ==========

func TestOraclePaginationFetchFirst(t *testing.T) {
	sql := "SELECT id, name FROM users ORDER BY id FETCH FIRST 10 ROWS ONLY"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected FETCH FIRST clause")
	}
	t.Logf("LIMIT text: %s", selectStmt.Limit.Text)
	// FETCH FIRST 不解析 offset/count，只保存文本
}

func TestOraclePaginationFetchFirstWithOffset(t *testing.T) {
	// Oracle 12c+ 支持 OFFSET ... FETCH FIRST
	sql := "SELECT id, name FROM users ORDER BY id OFFSET 20 ROWS FETCH FIRST 10 ROWS ONLY"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected FETCH FIRST clause")
	}
	t.Logf("LIMIT text: %s", selectStmt.Limit.Text)
}

func TestOraclePaginationFetchNext(t *testing.T) {
	// NEXT 和 FIRST 等价
	sql := "SELECT * FROM orders ORDER BY created_at DESC OFFSET 50 ROWS FETCH NEXT 20 ROWS ONLY"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected FETCH NEXT clause")
	}
	t.Logf("LIMIT text: %s", selectStmt.Limit.Text)
}

func TestOraclePaginationWithTies(t *testing.T) {
	// WITH TIES 返回并列结果
	sql := "SELECT * FROM products ORDER BY price DESC FETCH FIRST 10 ROWS WITH TIES"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected FETCH FIRST clause")
	}
	t.Logf("LIMIT text: %s", selectStmt.Limit.Text)
}

// ========== ROWNUM 分页测试 ==========

func TestOraclePaginationRowNum(t *testing.T) {
	// 传统 Oracle 分页方式
	sql := "SELECT * FROM (SELECT u.*, ROWNUM AS rn FROM users u WHERE ROWNUM <= 30) WHERE rn > 20"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.From) != 1 {
		t.Fatal("expected 1 FROM table")
	}
	t.Logf("ROWNUM pagination parsed successfully")
}

func TestOraclePaginationRowNumSimple(t *testing.T) {
	sql := "SELECT * FROM users WHERE ROWNUM <= 10"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	t.Logf("WHERE text: %s", selectStmt.Where.Text)
}

func TestOraclePaginationRowNumWithWhere(t *testing.T) {
	sql := "SELECT * FROM users WHERE status = 1 AND ROWNUM <= 50"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	t.Logf("WHERE text: %s", selectStmt.Where.Text)
}

// ========== 复杂分页测试 ==========

func TestOraclePaginationWithWhere(t *testing.T) {
	sql := "SELECT id, name, email FROM users WHERE status = 1 AND age > 18 ORDER BY id FETCH FIRST 20 ROWS ONLY"
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
}

func TestOraclePaginationWithOrderBy(t *testing.T) {
	sql := "SELECT * FROM users ORDER BY created_at DESC, id ASC FETCH FIRST 100 ROWS ONLY"
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
}

func TestOraclePaginationWithJoin(t *testing.T) {
	sql := "SELECT u.id, u.name, o.amount FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.status = 1 ORDER BY o.created_at DESC FETCH FIRST 20 ROWS ONLY"
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
}

func TestOraclePaginationWithGroupBy(t *testing.T) {
	sql := "SELECT user_id, COUNT(*) as order_count FROM orders GROUP BY user_id ORDER BY order_count DESC FETCH FIRST 10 ROWS ONLY"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
}

// ========== UNION 分页测试 ==========

func TestOraclePaginationWithUnion(t *testing.T) {
	sql := "SELECT id FROM users UNION SELECT id FROM admins ORDER BY 1 DESC FETCH FIRST 50 ROWS ONLY"
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
}

func TestOraclePaginationUnionAll(t *testing.T) {
	sql := "SELECT name FROM products UNION ALL SELECT name FROM services FETCH FIRST 30 ROWS ONLY"
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
}

// ========== 子查询分页测试 ==========

func TestOraclePaginationWithSubquery(t *testing.T) {
	sql := "SELECT * FROM (SELECT id, name FROM users WHERE status = 1 ORDER BY id FETCH FIRST 100 ROWS ONLY) tmp WHERE ROWNUM <= 10"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.From) != 1 {
		t.Fatal("expected 1 FROM table")
	}
	t.Logf("Nested pagination parsed successfully")
}

func TestOracleNestedPagination(t *testing.T) {
	// 嵌套分页查询
	sql := `SELECT * FROM (
		SELECT u.*, ROWNUM AS rn FROM (
			SELECT * FROM users ORDER BY id
		) u WHERE ROWNUM <= 100
	) WHERE rn > 90`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.From) != 1 {
		t.Fatal("expected outer FROM clause")
	}
	t.Logf("Nested ROWNUM pagination parsed successfully")
}

// ========== FOR UPDATE 分页测试 ==========

func TestOraclePaginationForUpdate(t *testing.T) {
	sql := "SELECT * FROM users WHERE status = 0 ORDER BY id FETCH FIRST 10 ROWS ONLY FOR UPDATE"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
}

func TestOraclePaginationForUpdateNowait(t *testing.T) {
	sql := "SELECT * FROM orders WHERE status = 'pending' ORDER BY created_at FETCH FIRST 20 ROWS ONLY FOR UPDATE NOWAIT"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT clause")
	}
}

// ========== 大数据量分页测试 ==========

func TestOracleLargeOffsetPagination(t *testing.T) {
	sql := "SELECT * FROM (SELECT t.*, ROWNUM AS rn FROM (SELECT * FROM logs ORDER BY id) t WHERE ROWNUM <= 1000100) WHERE rn > 1000000"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.From) != 1 {
		t.Fatal("expected FROM clause")
	}
	t.Logf("Large offset ROWNUM pagination parsed successfully")
}
