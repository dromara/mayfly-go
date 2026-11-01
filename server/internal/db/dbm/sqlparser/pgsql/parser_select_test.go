package pgsql

import (
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== SELECT 测试 ==========

func TestPgSelectBasic(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE status = 1 ORDER BY id DESC LIMIT 10"
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

	// 验证 FROM 表名
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

	// 验证 LIMIT
	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT")
	}
	if selectStmt.Limit.Text != "LIMIT 10" {
		t.Errorf("expected LIMIT.Text='LIMIT 10', got '%s'", selectStmt.Limit.Text)
	}
	if selectStmt.Limit.Count != 10 {
		t.Errorf("expected LIMIT.Count=10, got %d", selectStmt.Limit.Count)
	}
}

func TestPgSelectDistinct(t *testing.T) {
	sql := "SELECT DISTINCT id, name FROM users"
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

func TestPgSelectStar(t *testing.T) {
	sql := "SELECT * FROM users"
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

	// 验证 SELECT *
	if len(selectStmt.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(selectStmt.Items))
	}
	if selectStmt.Items[0].Text != "*" {
		t.Errorf("expected Items[0]='*', got '%s'", selectStmt.Items[0].Text)
	}

	// 验证 FROM
	if len(selectStmt.From) != 1 {
		t.Fatalf("expected 1 from table, got %d", len(selectStmt.From))
	}
	if selectStmt.From[0].Name != "users" {
		t.Errorf("expected From[0].Name='users', got '%s'", selectStmt.From[0].Name)
	}
}

func TestPgSelectWithJoin(t *testing.T) {
	sql := "SELECT u.id, o.amount FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.status = 1"
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

	// 验证主表
	if len(selectStmt.From) != 1 {
		t.Fatalf("expected 1 from table, got %d", len(selectStmt.From))
	}
	if selectStmt.From[0].Name != "users" {
		t.Errorf("expected From[0].Name='users', got '%s'", selectStmt.From[0].Name)
	}
	if selectStmt.From[0].Alias != "u" {
		t.Errorf("expected From[0].Alias='u', got '%s'", selectStmt.From[0].Alias)
	}

	// 验证 JOIN
	if len(selectStmt.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(selectStmt.Joins))
	}
	if selectStmt.Joins[0].Table.Name != "orders" {
		t.Errorf("expected Join[0].Table.Name='orders', got '%s'", selectStmt.Joins[0].Table.Name)
	}
	if selectStmt.Joins[0].Table.Alias != "o" {
		t.Errorf("expected Join[0].Table.Alias='o', got '%s'", selectStmt.Joins[0].Table.Alias)
	}
	if selectStmt.Joins[0].Kind != sqlstmt.JoinKindLeft {
		t.Errorf("expected Join[0].Kind=JoinKindLeft, got '%d'", selectStmt.Joins[0].Kind)
	}
	if selectStmt.Joins[0].On == nil {
		t.Fatal("expected Join[0].On")
	}
	if selectStmt.Joins[0].On.Text != "u.id = o.user_id" {
		t.Errorf("expected Join[0].On.Text='u.id = o.user_id', got '%s'", selectStmt.Joins[0].On.Text)
	}

	// 验证 WHERE
	if selectStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	if selectStmt.Where.Text != "u.status = 1" {
		t.Errorf("expected WHERE='u.status = 1', got '%s'", selectStmt.Where.Text)
	}
}

func TestPgSelectComplexWhere(t *testing.T) {
	sql := "SELECT * FROM users WHERE status = 1 AND (age > 18 OR role = 'admin') ORDER BY name"
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
	expectedWhere := "status = 1 AND (age > 18 OR role = 'admin')"
	if selectStmt.Where.Text != expectedWhere {
		t.Errorf("expected WHERE='%s', got '%s'", expectedWhere, selectStmt.Where.Text)
	}

	// 验证 ORDER BY
	if len(selectStmt.OrderBy) != 1 {
		t.Fatalf("expected 1 OrderBy, got %d", len(selectStmt.OrderBy))
	}
	if selectStmt.OrderBy[0].Text != "name" {
		t.Errorf("expected OrderBy[0].Text='name', got '%s'", selectStmt.OrderBy[0].Text)
	}
}

func TestPgOffsetLimit(t *testing.T) {
	sql := "SELECT * FROM users OFFSET 10 LIMIT 20"
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

	// 验证 LIMIT/OFFSET
	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT")
	}
	if selectStmt.Limit.Text != "OFFSET 10 LIMIT 20" {
		t.Errorf("expected LIMIT.Text='OFFSET 10 LIMIT 20', got '%s'", selectStmt.Limit.Text)
	}
	if selectStmt.Limit.Count != 20 {
		t.Errorf("expected LIMIT.Count=20, got %d", selectStmt.Limit.Count)
	}
	if selectStmt.Limit.Offset != 10 {
		t.Errorf("expected LIMIT.Offset=10, got %d", selectStmt.Limit.Offset)
	}
}

func TestPgLimitAll(t *testing.T) {
	sql := "SELECT * FROM users LIMIT ALL"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Limit == nil || selectStmt.Limit.Text != "LIMIT ALL" {
		t.Errorf("expected LIMIT text='LIMIT ALL'")
	}
}

func TestPgUnion(t *testing.T) {
	sql := "SELECT 1 UNION SELECT 2 UNION ALL SELECT 3"
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

	// 验证 UNION 数量
	if len(selectStmt.Unions) != 2 {
		t.Fatalf("expected 2 unions, got %d", len(selectStmt.Unions))
	}

	// 验证第一个 UNION（DISTINCT）
	if selectStmt.Unions[0].All {
		t.Error("expected Unions[0].All=false (DISTINCT)")
	}
	if selectStmt.Unions[0].Select == nil {
		t.Fatal("expected Unions[0].Select")
	}

	// 验证第二个 UNION（ALL）
	if !selectStmt.Unions[1].All {
		t.Error("expected Unions[1].All=true")
	}
	if selectStmt.Unions[1].Select == nil {
		t.Fatal("expected Unions[1].Select")
	}
}

// ========== FOR UPDATE 测试 ==========

func TestPgForUpdate(t *testing.T) {
	sql := "SELECT id FROM users WHERE id = 1 FOR UPDATE"
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
	if selectStmt.Where.Text != "id = 1" {
		t.Errorf("expected WHERE='id = 1', got '%s'", selectStmt.Where.Text)
	}

	// 注意：FOR UPDATE 标记可能在 Base.Text 中体现
	// 只要完整文本包含 FOR UPDATE 即可
	if !strings.Contains(selectStmt.GetText(), "FOR UPDATE") {
		t.Error("expected text to contain 'FOR UPDATE'")
	}
}

func TestPgForUpdateSkipLocked(t *testing.T) {
	sql := "SELECT id FROM users WHERE status = 1 FOR UPDATE SKIP LOCKED"
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

// ========== RETURNING 测试 ==========

func TestPgUpdateReturning(t *testing.T) {
	sql := "UPDATE users SET name = 'John' WHERE id = 1 RETURNING id, name"
	parser := newParser(sql)
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

func TestPgDeleteReturning(t *testing.T) {
	sql := "DELETE FROM users WHERE id = 1 RETURNING *"
	parser := newParser(sql)
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

// ========== DDL 测试 ==========

func TestPgDDL(t *testing.T) {
	sqls := []string{
		"CREATE TABLE users (id INT PRIMARY KEY)",
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
			if stmt.GetText() != sql {
				t.Errorf("expected text='%s'", sql)
			}
		})
	}
}
