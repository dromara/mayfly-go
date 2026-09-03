package dm

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== SELECT 测试 ==========

func TestDmSelectBasic(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE status = 1 ORDER BY id DESC LIMIT 10"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.GetText() != sql {
		t.Errorf("expected text='%s'", sql)
	}
	if selectStmt.Where == nil || selectStmt.Where.Text != "status = 1" {
		t.Errorf("expected WHERE='status = 1'")
	}
	if len(selectStmt.OrderBy) != 1 || selectStmt.OrderBy[0].Text != "id" || !selectStmt.OrderBy[0].Desc {
		t.Errorf("expected ORDER BY id DESC")
	}
	if selectStmt.Limit == nil || selectStmt.Limit.Text != "LIMIT 10" || selectStmt.Limit.Count != 10 {
		t.Errorf("expected LIMIT 10")
	}
}

func TestDmSelectTop(t *testing.T) {
	sql := "SELECT TOP 10 id, name FROM users ORDER BY id"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if len(selectStmt.Items) != 2 {
		t.Fatalf("expected 2 items")
	}
}

func TestDmSelectDistinct(t *testing.T) {
	sql := "SELECT DISTINCT department_id FROM users"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if !selectStmt.Distinct {
		t.Error("expected Distinct=true")
	}
}

func TestDmSelectStar(t *testing.T) {
	sql := "SELECT * FROM users"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if len(selectStmt.Items) != 1 || selectStmt.Items[0].Text != "*" {
		t.Errorf("expected SELECT *")
	}
}

func TestDmSelectWithJoin(t *testing.T) {
	sql := "SELECT u.id, o.amount FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.status = 1"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if len(selectStmt.Joins) != 1 || selectStmt.Joins[0].Table.Name != "orders" {
		t.Errorf("expected JOIN orders")
	}
	if selectStmt.Where == nil || selectStmt.Where.Text != "u.status = 1" {
		t.Errorf("expected WHERE='u.status = 1'")
	}
}

func TestDmSelectMultipleJoins(t *testing.T) {
	sql := "SELECT u.id, o.amount, p.name FROM users u LEFT JOIN orders o ON u.id = o.user_id INNER JOIN products p ON o.product_id = p.id WHERE u.status = 1"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if len(selectStmt.Joins) != 2 {
		t.Fatalf("expected 2 joins")
	}
}

func TestDmSelectComplexWhere(t *testing.T) {
	sql := "SELECT * FROM users WHERE status = 1 AND (age > 18 OR role = 'admin') ORDER BY name"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	expectedWhere := "status = 1 AND (age > 18 OR role = 'admin')"
	if selectStmt.Where.Text != expectedWhere {
		t.Errorf("expected WHERE='%s'", expectedWhere)
	}
}

// ========== LIMIT/OFFSET 测试 ==========

func TestDmLimitOffset(t *testing.T) {
	sql := "SELECT id FROM users ORDER BY id LIMIT 20, 10"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Limit == nil || selectStmt.Limit.Text != "LIMIT 20, 10" {
		t.Errorf("expected LIMIT text='LIMIT 20, 10'")
	}
}

func TestDmLimitOnly(t *testing.T) {
	sql := "SELECT * FROM users LIMIT 5"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Limit == nil || selectStmt.Limit.Text != "LIMIT 5" || selectStmt.Limit.Count != 5 {
		t.Errorf("expected LIMIT 5")
	}
}

// ========== UNION 测试 ==========

func TestDmUnion(t *testing.T) {
	sql := "SELECT id FROM users UNION SELECT id FROM admins ORDER BY 1"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if len(selectStmt.Unions) != 1 {
		t.Fatalf("expected 1 union")
	}
	if len(selectStmt.OrderBy) != 1 {
		t.Fatalf("expected ORDER BY")
	}
}

func TestDmUnionWithOrderBy(t *testing.T) {
	sql := "SELECT id FROM users UNION SELECT id FROM admins ORDER BY 1 DESC"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if len(selectStmt.Unions) != 1 {
		t.Fatalf("expected 1 union")
	}
	if len(selectStmt.OrderBy) != 1 || selectStmt.OrderBy[0].Text != "1" || !selectStmt.OrderBy[0].Desc {
		t.Errorf("expected ORDER BY 1 DESC")
	}
}

func TestDmUnionWithLimit(t *testing.T) {
	sql := "SELECT id FROM users UNION ALL SELECT id FROM admins LIMIT 20"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Limit == nil || selectStmt.Limit.Text != "LIMIT 20" || selectStmt.Limit.Count != 20 {
		t.Errorf("expected LIMIT 20")
	}
}

// ========== FOR UPDATE 测试 ==========

func TestDmForUpdate(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE id = 1 FOR UPDATE"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Where == nil || selectStmt.Where.Text != "id = 1" {
		t.Errorf("expected WHERE='id = 1'")
	}
}

// ========== WITH (CTE) 测试 ==========

func TestDmWithClause(t *testing.T) {
	sql := "WITH temp_users AS (SELECT * FROM users WHERE status = 1) SELECT * FROM temp_users"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if stmt == nil {
		t.Fatalf("expected stmt not nil")
	}
}

func TestDmMultipleWith(t *testing.T) {
	sql := `WITH 
		active_users AS (SELECT * FROM users WHERE status = 1),
		recent_orders AS (SELECT * FROM orders WHERE created_at > NOW())
	SELECT * FROM active_users u JOIN recent_orders o ON u.id = o.user_id`

	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if stmt == nil {
		t.Fatalf("expected stmt not nil")
	}
}

// ========== DDL 测试 ==========

func TestDmDDL(t *testing.T) {
	sqls := []string{
		"CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))",
		"DROP TABLE users",
		"ALTER TABLE users ADD COLUMN email VARCHAR(255)",
	}

	for _, sql := range sqls {
		t.Run(sql[:10], func(t *testing.T) {
			parser := NewParser(sql)
			stmt, err := parser.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if stmt == nil {
				t.Fatalf("expected stmt not nil")
			}
		})
	}
}
