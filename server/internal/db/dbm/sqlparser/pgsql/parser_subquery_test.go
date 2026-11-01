package pgsql

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== 子查询测试 ==========

func TestPgSubqueryInFrom(t *testing.T) {
	sql := "SELECT * FROM (SELECT id, name FROM users WHERE status = 1) AS u WHERE u.id > 10"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("FROM 表数量: %d", len(selectStmt.From))

	if len(selectStmt.From) != 1 {
		t.Fatalf("expected 1 table in FROM, got %d", len(selectStmt.From))
	}

	t.Logf("FROM[0] Name: %s", selectStmt.From[0].Name)
	t.Logf("FROM[0] Alias: %s", selectStmt.From[0].Alias)

	if selectStmt.From[0].Alias != "u" {
		t.Errorf("expected alias='u', got '%s'", selectStmt.From[0].Alias)
	}

	if selectStmt.Where == nil {
		t.Fatal("expected outer WHERE clause")
	}
	t.Logf("外层 WHERE: %s", selectStmt.Where.Text)
	if selectStmt.Where.Text != "u.id > 10" {
		t.Errorf("expected outer WHERE text='u.id > 10', got '%s'", selectStmt.Where.Text)
	}
}

func TestPgSubqueryInWhere(t *testing.T) {
	sql := "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders WHERE amount > 100)"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("WHERE: %s", selectStmt.Where.Text)

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	expectedWhere := "id IN (SELECT user_id FROM orders WHERE amount > 100)"
	if selectStmt.Where.Text != expectedWhere {
		t.Errorf("expected WHERE text='%s', got '%s'", expectedWhere, selectStmt.Where.Text)
	}
}

func TestPgSubqueryInSelect(t *testing.T) {
	sql := "SELECT id, name, (SELECT COUNT(*) FROM orders WHERE orders.user_id = users.id) AS order_count FROM users"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("SELECT 项数量: %d", len(selectStmt.Items))

	for i, item := range selectStmt.Items {
		t.Logf("  Item[%d]: Text='%s', ColumnName='%s', Alias='%s'", i, item.Text, item.ColumnName, item.Alias)
	}

	if len(selectStmt.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(selectStmt.Items))
	}

	if selectStmt.Items[2].Alias != "order_count" {
		t.Errorf("expected item[2] alias='order_count', got '%s'", selectStmt.Items[2].Alias)
	}
}

func TestPgNestedSubquery(t *testing.T) {
	sql := "SELECT * FROM (SELECT * FROM (SELECT id FROM users WHERE status = 1) AS inner_u) AS outer_u WHERE outer_u.id > 5"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("外层 WHERE: %s", selectStmt.Where.Text)

	if selectStmt.Where == nil {
		t.Fatal("expected outer WHERE")
	}
	if selectStmt.Where.Text != "outer_u.id > 5" {
		t.Errorf("expected WHERE text='outer_u.id > 5', got '%s'", selectStmt.Where.Text)
	}
}

func TestPgSubqueryWithUnion(t *testing.T) {
	sql := "SELECT * FROM (SELECT id FROM users UNION SELECT id FROM admins) AS all_users WHERE all_users.id > 10"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	if selectStmt.Where.Text != "all_users.id > 10" {
		t.Errorf("expected WHERE text='all_users.id > 10', got '%s'", selectStmt.Where.Text)
	}
}

func TestPgCorrelatedSubquery(t *testing.T) {
	sql := "SELECT u.id, u.name FROM users u WHERE u.id IN (SELECT o.user_id FROM orders o WHERE o.user_id = u.id AND o.amount > 50)"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("WHERE: %s", selectStmt.Where.Text)

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	expectedWhere := "u.id IN (SELECT o.user_id FROM orders o WHERE o.user_id = u.id AND o.amount > 50)"
	if selectStmt.Where.Text != expectedWhere {
		t.Errorf("expected WHERE text='%s', got '%s'", expectedWhere, selectStmt.Where.Text)
	}
}

func TestPgSubqueryWithExists(t *testing.T) {
	sql := "SELECT * FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id)"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("WHERE: %s", selectStmt.Where.Text)

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	expectedWhere := "EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id)"
	if selectStmt.Where.Text != expectedWhere {
		t.Errorf("expected WHERE text='%s', got '%s'", expectedWhere, selectStmt.Where.Text)
	}
}

func TestPgMultipleSubqueries(t *testing.T) {
	sql := "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders) AND department_id IN (SELECT id FROM departments WHERE name = 'IT')"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("WHERE: %s", selectStmt.Where.Text)

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	expectedWhere := "id IN (SELECT user_id FROM orders) AND department_id IN (SELECT id FROM departments WHERE name = 'IT')"
	if selectStmt.Where.Text != expectedWhere {
		t.Errorf("expected WHERE text='%s', got '%s'", expectedWhere, selectStmt.Where.Text)
	}
}

func TestPgSubqueryWithLimit(t *testing.T) {
	sql := "SELECT * FROM (SELECT id, name FROM users ORDER BY created_at DESC LIMIT 10) AS top_users"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("FROM 数量: %d", len(selectStmt.From))

	if len(selectStmt.From) != 1 {
		t.Fatalf("expected 1 table, got %d", len(selectStmt.From))
	}

	if selectStmt.From[0].Alias != "top_users" {
		t.Errorf("expected alias='top_users', got '%s'", selectStmt.From[0].Alias)
	}
}

func TestPgComplexSubquery(t *testing.T) {
	sql := `SELECT 
		u.id, 
		u.name,
		(SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) AS order_count,
		(SELECT SUM(amount) FROM orders o WHERE o.user_id = u.id AND o.status = 'completed') AS total_amount
	FROM users u
	WHERE u.status = 1
		AND u.id IN (SELECT user_id FROM user_groups WHERE group_id = 5)
	ORDER BY u.name`

	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("SELECT 项数量: %d", len(selectStmt.Items))

	if len(selectStmt.Items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(selectStmt.Items))
	}

	t.Logf("Item[0]: %s", selectStmt.Items[0].Text)
	t.Logf("Item[1]: %s", selectStmt.Items[1].Text)
	t.Logf("Item[2]: %s (alias: %s)", selectStmt.Items[2].Text, selectStmt.Items[2].Alias)
	t.Logf("Item[3]: %s (alias: %s)", selectStmt.Items[3].Text, selectStmt.Items[3].Alias)

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	t.Logf("WHERE: %s", selectStmt.Where.Text)

	if len(selectStmt.OrderBy) != 1 {
		t.Fatalf("expected 1 order by, got %d", len(selectStmt.OrderBy))
	}
	if selectStmt.OrderBy[0].Text != "u.name" {
		t.Errorf("expected order by text='u.name', got '%s'", selectStmt.OrderBy[0].Text)
	}
}

// ========== JOIN 测试 ==========

func TestPgMultipleJoins(t *testing.T) {
	sql := "SELECT u.id, o.amount, p.name FROM users u LEFT JOIN orders o ON u.id = o.user_id INNER JOIN products p ON o.product_id = p.id WHERE u.status = 1"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("JOIN 数量: %d", len(selectStmt.Joins))

	if len(selectStmt.Joins) != 2 {
		t.Fatalf("expected 2 joins, got %d", len(selectStmt.Joins))
	}

	if selectStmt.Joins[0].Table.Name != "orders" {
		t.Errorf("expected first join table='orders', got '%s'", selectStmt.Joins[0].Table.Name)
	}
	if selectStmt.Joins[1].Table.Name != "products" {
		t.Errorf("expected second join table='products', got '%s'", selectStmt.Joins[1].Table.Name)
	}

	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	if selectStmt.Where.Text != "u.status = 1" {
		t.Errorf("expected WHERE text='u.status = 1', got '%s'", selectStmt.Where.Text)
	}
}

func TestPgRightJoin(t *testing.T) {
	sql := "SELECT * FROM users u RIGHT JOIN orders o ON u.id = o.user_id"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(selectStmt.Joins))
	}
}

func TestPgCrossJoin(t *testing.T) {
	sql := "SELECT * FROM users CROSS JOIN roles"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	if len(selectStmt.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(selectStmt.Joins))
	}
}

// ========== UNION 测试 ==========

func TestPgUnionWithOrderBy(t *testing.T) {
	sql := "SELECT id FROM users UNION SELECT id FROM admins ORDER BY 1 DESC"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("UNION 数量: %d", len(selectStmt.Unions))

	if len(selectStmt.Unions) != 1 {
		t.Fatalf("expected 1 union, got %d", len(selectStmt.Unions))
	}

	// ORDER BY 应该属于整个 UNION
	if len(selectStmt.OrderBy) != 1 {
		t.Fatalf("expected 1 order by, got %d", len(selectStmt.OrderBy))
	}
	if selectStmt.OrderBy[0].Text != "1" {
		t.Errorf("expected order by text='1', got '%s'", selectStmt.OrderBy[0].Text)
	}
	if !selectStmt.OrderBy[0].Desc {
		t.Error("expected order by DESC")
	}
}

func TestPgUnionWithLimit(t *testing.T) {
	sql := "SELECT id FROM users UNION ALL SELECT id FROM admins LIMIT 20"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	t.Logf("完整文本: %s", selectStmt.GetText())

	if selectStmt.Limit == nil {
		t.Fatal("expected LIMIT")
	}
	if selectStmt.Limit.Text != "LIMIT 20" {
		t.Errorf("expected LIMIT text='LIMIT 20', got '%s'", selectStmt.Limit.Text)
	}
	if selectStmt.Limit.Count != 20 {
		t.Errorf("expected LIMIT count=20, got %d", selectStmt.Limit.Count)
	}
}
