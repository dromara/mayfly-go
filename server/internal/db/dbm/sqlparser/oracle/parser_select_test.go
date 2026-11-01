package oracle

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== SELECT 测试 ==========

func TestOracleSelectBasic(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE status = 1 ORDER BY id DESC"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	// 验证完整文本
	if selectStmt.GetText() != sql {
		t.Errorf("expected text='%s', got '%s'", sql, selectStmt.GetText())
	}

	// 验证 SELECT 字段
	if len(selectStmt.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(selectStmt.Items))
	}
	if selectStmt.Items[0].Text != "id" {
		t.Errorf("expected Items[0]='id', got '%s'", selectStmt.Items[0].Text)
	}
	if selectStmt.Items[1].Text != "name" {
		t.Errorf("expected Items[1]='name', got '%s'", selectStmt.Items[1].Text)
	}

	// 验证 FROM
	if len(selectStmt.From) != 1 {
		t.Fatalf("expected 1 from table, got %d", len(selectStmt.From))
	}
	if selectStmt.From[0].Name != "users" {
		t.Errorf("expected From[0].Name='users', got '%s'", selectStmt.From[0].Name)
	}

	// 验证 WHERE
	if selectStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	if selectStmt.Where.Text != "status = 1" {
		t.Errorf("expected WHERE='status = 1', got '%s'", selectStmt.Where.Text)
	}

	// 验证 ORDER BY
	if len(selectStmt.OrderBy) != 1 {
		t.Fatalf("expected 1 OrderBy, got %d", len(selectStmt.OrderBy))
	}
	if selectStmt.OrderBy[0].Text != "id" {
		t.Errorf("expected OrderBy[0].Text='id', got '%s'", selectStmt.OrderBy[0].Text)
	}
	if !selectStmt.OrderBy[0].Desc {
		t.Error("expected OrderBy[0].Desc=true")
	}
}

func TestOracleSelectDistinct(t *testing.T) {
	sql := "SELECT DISTINCT department_id FROM users"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if !selectStmt.Distinct {
		t.Error("expected Distinct=true")
	}
}

func TestOracleSelectWithJoin(t *testing.T) {
	sql := "SELECT u.id, o.amount FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.status = 1"
	parser := newParser(sql)
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

// ========== FETCH FIRST 测试 ==========

func TestOracleFetchFirst(t *testing.T) {
	sql := "SELECT id, name FROM users ORDER BY id FETCH FIRST 10 ROWS ONLY"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Limit == nil || selectStmt.Limit.Text != "FETCH FIRST 10 ROWS ONLY" {
		t.Errorf("expected LIMIT text='FETCH FIRST 10 ROWS ONLY'")
	}
}

// ========== ROWNUM 测试 ==========

func TestOracleRowNum(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE status = 1 AND ROWNUM <= 10 ORDER BY id"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	expectedWhere := "status = 1 AND ROWNUM <= 10"
	if selectStmt.Where.Text != expectedWhere {
		t.Errorf("expected WHERE='%s'", expectedWhere)
	}
}

// ========== FOR UPDATE 测试 ==========

func TestOracleForUpdate(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE id = 1 FOR UPDATE"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Where == nil || selectStmt.Where.Text != "id = 1" {
		t.Errorf("expected WHERE='id = 1'")
	}
}

func TestOracleForUpdateNowait(t *testing.T) {
	sql := "SELECT id FROM users WHERE status = 1 FOR UPDATE NOWAIT"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Where == nil || selectStmt.Where.Text != "status = 1" {
		t.Errorf("expected WHERE='status = 1'")
	}
}

// ========== CONNECT BY 测试 ==========

func TestOracleConnectBy(t *testing.T) {
	sql := "SELECT id, name, level FROM employees CONNECT BY PRIOR id = manager_id START WITH manager_id IS NULL"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if len(selectStmt.Items) == 0 {
		t.Fatal("expected SELECT items")
	}
}

// ========== UNION 测试 ==========

func TestOracleUnion(t *testing.T) {
	sql := "SELECT id FROM users UNION SELECT id FROM admins ORDER BY 1"
	parser := newParser(sql)
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

// ========== WITH (CTE) 测试 ==========

func TestOracleWithClause(t *testing.T) {
	sql := "WITH temp_users AS (SELECT * FROM users WHERE status = 1) SELECT * FROM temp_users"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if stmt == nil {
		t.Fatalf("expected stmt not nil")
	}
}

// ========== DDL 测试 ==========

func TestOracleDDL(t *testing.T) {
	sqls := []string{
		"CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))",
		"DROP TABLE users",
		"ALTER TABLE users ADD COLUMN email VARCHAR(255)",
	}

	for _, sql := range sqls {
		t.Run(sql[:10], func(t *testing.T) {
			parser := newParser(sql)
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
