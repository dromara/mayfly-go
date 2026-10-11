package mysql

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== 基础 SELECT 测试 ==========

func TestSelectBasic(t *testing.T) {
	sql := "-- 测试查询 sql \n SELECT `id`, name FROM users WHERE status = 1 ORDER BY id DESC LIMIT 10"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.GetText() != sql {
		t.Errorf("expected text='%s', got '%s'", sql, selectStmt.GetText())
	}

	if len(selectStmt.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(selectStmt.Items))
	}
	// Text 的契约是「原始文本」（保留引用符），跨方言一致；去引用符后的名字在 ColumnName。
	// 保留引用符而非整体 Unquote：整体去引号会把 `u`.`phone` 破坏成 u`.`phone，污染限定符分段，
	// 而脱敏血缘正是按段取词（base.ExtractExprQualifiedPairs 同时支持 `x` 与 "x"）
	if selectStmt.Items[0].Text != "`id`" || selectStmt.Items[0].ColumnName != "id" {
		t.Errorf("expected item[0] text='`id`' column='id', got %+v", selectStmt.Items[0])
	}
	if selectStmt.Items[1].Text != "name" || selectStmt.Items[1].ColumnName != "name" {
		t.Errorf("expected item[1] text='name' column='name', got %+v", selectStmt.Items[1])
	}

	if len(selectStmt.From) != 1 || selectStmt.From[0].Name != "users" {
		t.Errorf("expected table='users'")
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

func TestSelectDistinct(t *testing.T) {
	sql := "SELECT DISTINCT id, name FROM users"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if !selectStmt.Distinct {
		t.Error("expected Distinct=true")
	}
	if len(selectStmt.Items) != 2 {
		t.Fatalf("expected 2 items")
	}
}

func TestSelectStar(t *testing.T) {
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

func TestSelectTableStar(t *testing.T) {
	sql := "SELECT u.*, o.amount FROM users u, orders o"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Items) != 2 {
		t.Fatalf("expected 2 items")
	}
	if selectStmt.Items[0].Text != "u.*" {
		t.Errorf("expected item[0]='u.*', got '%s'", selectStmt.Items[0].Text)
	}
}

func TestSelectWithAlias(t *testing.T) {
	sql := "SELECT id AS user_id, name AS user_name FROM users"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Items) != 2 {
		t.Fatalf("expected 2 items")
	}
	if selectStmt.Items[0].Alias != "user_id" {
		t.Errorf("expected alias='user_id', got '%s'", selectStmt.Items[0].Alias)
	}
	if selectStmt.Items[1].Alias != "user_name" {
		t.Errorf("expected alias='user_name', got '%s'", selectStmt.Items[1].Alias)
	}
}

func TestSelectMultipleTables(t *testing.T) {
	sql := "SELECT * FROM users, orders, products"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.From) != 3 {
		t.Fatalf("expected 3 tables, got %d", len(selectStmt.From))
	}
	if selectStmt.From[0].Name != "users" || selectStmt.From[1].Name != "orders" || selectStmt.From[2].Name != "products" {
		t.Errorf("expected tables: users, orders, products")
	}
}

// ========== JOIN 测试 ==========

func TestSelectWithJoin(t *testing.T) {
	sql := "SELECT u.id, o.amount FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.status = 1"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Joins) != 1 {
		t.Fatalf("expected 1 join")
	}
	if selectStmt.Joins[0].Table.Name != "orders" {
		t.Errorf("expected join table='orders'")
	}
	if selectStmt.Joins[0].On == nil || selectStmt.Joins[0].On.Text != "u.id = o.user_id" {
		t.Errorf("expected ON='u.id = o.user_id'")
	}
	if selectStmt.Where == nil || selectStmt.Where.Text != "u.status = 1" {
		t.Errorf("expected WHERE='u.status = 1'")
	}
}

func TestSelectInnerJoin(t *testing.T) {
	sql := "SELECT * FROM users u INNER JOIN orders o ON u.id = o.user_id"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Joins) != 1 {
		t.Fatalf("expected 1 join")
	}
}

func TestSelectRightJoin(t *testing.T) {
	sql := "SELECT * FROM users u RIGHT JOIN orders o ON u.id = o.user_id"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Joins) != 1 {
		t.Fatalf("expected 1 join")
	}
}

func TestSelectCrossJoin(t *testing.T) {
	sql := "SELECT * FROM users CROSS JOIN roles"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Joins) != 1 {
		t.Fatalf("expected 1 join")
	}
}

// ========== WHERE 测试 ==========

func TestSelectComplexWhere(t *testing.T) {
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
		t.Errorf("expected WHERE='%s', got '%s'", expectedWhere, selectStmt.Where.Text)
	}
}

// ========== LIMIT/OFFSET 测试 ==========

func TestSelectLimitOffset(t *testing.T) {
	sql := "SELECT * FROM users LIMIT 10, 20"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT")
	}
	if selectStmt.Limit.Text != "LIMIT 10, 20" {
		t.Errorf("expected LIMIT text='LIMIT 10, 20'")
	}
	if selectStmt.Limit.Offset != 10 {
		t.Errorf("expected offset=10, got %d", selectStmt.Limit.Offset)
	}
	if selectStmt.Limit.Count != 20 {
		t.Errorf("expected count=20, got %d", selectStmt.Limit.Count)
	}
}

// ========== UNION 测试 ==========

func TestUnionSelect(t *testing.T) {
	sql := "SELECT 1 UNION SELECT 2 UNION ALL SELECT 3"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Unions) != 2 {
		t.Fatalf("expected 2 unions, got %d", len(selectStmt.Unions))
	}
	if selectStmt.Unions[0].All {
		t.Error("expected first union not ALL")
	}
	if !selectStmt.Unions[1].All {
		t.Error("expected second union ALL")
	}
}

func TestUnionWithOrderByLimit(t *testing.T) {
	sql := "SELECT id FROM users UNION SELECT id FROM admins ORDER BY 1 DESC LIMIT 20"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.OrderBy) != 1 || selectStmt.OrderBy[0].Text != "1" || !selectStmt.OrderBy[0].Desc {
		t.Errorf("expected ORDER BY 1 DESC")
	}
	if selectStmt.Limit == nil || selectStmt.Limit.Count != 20 {
		t.Errorf("expected LIMIT 20")
	}
}

func TestUnionMultiple(t *testing.T) {
	sql := "SELECT id FROM t1 UNION SELECT id FROM t2 UNION ALL SELECT id FROM t3 UNION SELECT id FROM t4"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Unions) != 3 {
		t.Fatalf("expected 3 unions, got %d", len(selectStmt.Unions))
	}
}

// ========== SHOW 测试 ==========

func TestShow(t *testing.T) {
	sql := "SHOW VARIABLES LIKE 'max_connections'"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if stmt == nil {
		t.Fatalf("expected stmt not nil")
	}
	if stmt.GetText() != sql {
		t.Errorf("expected text='%s'", sql)
	}
}

func TestShowTables(t *testing.T) {
	sql := "SHOW TABLES"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if stmt == nil {
		t.Fatalf("expected stmt not nil")
	}
}
