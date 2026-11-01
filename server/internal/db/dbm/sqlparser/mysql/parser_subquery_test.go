package mysql

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

func TestSubqueryInFrom(t *testing.T) {
	// FROM 子句中的子查询
	sql := "SELECT * FROM (SELECT id, name FROM users WHERE status = 1) AS u WHERE u.id > 10"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if stmt == nil {
		t.Fatalf("expected stmt not nil")
	}

	selectStmt, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok {
		t.Fatal("expected SelectStmt")
	}

	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("FROM 表数量: %d", len(selectStmt.From))

	// 验证 FROM 子句
	if len(selectStmt.From) != 1 {
		t.Fatalf("expected 1 table in FROM, got %d", len(selectStmt.From))
	}

	// FROM 应该是子查询
	fromText := selectStmt.From[0].Name
	t.Logf("FROM[0] Name: %s", fromText)
	t.Logf("FROM[0] Alias: %s", selectStmt.From[0].Alias)

	if selectStmt.From[0].Alias != "u" {
		t.Errorf("expected alias='u', got '%s'", selectStmt.From[0].Alias)
	}

	// 验证外层 WHERE
	if selectStmt.Where == nil {
		t.Fatal("expected outer WHERE clause")
	}
	t.Logf("外层 WHERE: %s", selectStmt.Where.Text)
	if selectStmt.Where.Text != "u.id > 10" {
		t.Errorf("expected outer WHERE text='u.id > 10', got '%s'", selectStmt.Where.Text)
	}
}

func TestSubqueryInWhere(t *testing.T) {
	// WHERE 子句中的子查询（IN）
	sql := "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders WHERE amount > 100)"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("WHERE: %s", selectStmt.Where.Text)

	// WHERE 应该包含整个 IN 子句
	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	expectedWhere := "id IN (SELECT user_id FROM orders WHERE amount > 100)"
	if selectStmt.Where.Text != expectedWhere {
		t.Errorf("expected WHERE text='%s', got '%s'", expectedWhere, selectStmt.Where.Text)
	}
}

func TestSubqueryInSelect(t *testing.T) {
	// SELECT 列中的子查询
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

	// 应该有 3 个 SELECT 项
	if len(selectStmt.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(selectStmt.Items))
	}

	// 第三个项应该是子查询
	if selectStmt.Items[2].Alias != "order_count" {
		t.Errorf("expected item[2] alias='order_count', got '%s'", selectStmt.Items[2].Alias)
	}
}

func TestNestedSubquery(t *testing.T) {
	// 嵌套子查询
	sql := "SELECT * FROM (SELECT * FROM (SELECT id FROM users WHERE status = 1) AS inner_u) AS outer_u WHERE outer_u.id > 5"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("外层 WHERE: %s", selectStmt.Where.Text)

	// 验证外层 WHERE
	if selectStmt.Where == nil {
		t.Fatal("expected outer WHERE")
	}
	if selectStmt.Where.Text != "outer_u.id > 5" {
		t.Errorf("expected WHERE text='outer_u.id > 5', got '%s'", selectStmt.Where.Text)
	}
}

func TestSubqueryWithUnion(t *testing.T) {
	// 子查询包含 UNION
	sql := "SELECT * FROM (SELECT id FROM users UNION SELECT id FROM admins) AS all_users WHERE all_users.id > 10"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("FROM 数量: %d", len(selectStmt.From))

	// 验证外层 WHERE
	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	if selectStmt.Where.Text != "all_users.id > 10" {
		t.Errorf("expected WHERE text='all_users.id > 10', got '%s'", selectStmt.Where.Text)
	}
}

func TestCorrelatedSubquery(t *testing.T) {
	// 相关子查询
	sql := "SELECT u.id, u.name FROM users u WHERE u.id IN (SELECT o.user_id FROM orders o WHERE o.user_id = u.id AND o.amount > 50)"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)

	t.Logf("完整文本: %s", selectStmt.GetText())
	t.Logf("WHERE: %s", selectStmt.Where.Text)

	// WHERE 应该包含相关子查询
	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	expectedWhere := "u.id IN (SELECT o.user_id FROM orders o WHERE o.user_id = u.id AND o.amount > 50)"
	if selectStmt.Where.Text != expectedWhere {
		t.Errorf("expected WHERE text='%s', got '%s'", expectedWhere, selectStmt.Where.Text)
	}
}

func TestSubqueryWithExists(t *testing.T) {
	// EXISTS 子查询
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

func TestMultipleSubqueries(t *testing.T) {
	// 多个子查询
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

func TestSubqueryWithLimit(t *testing.T) {
	// 子查询包含 LIMIT
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

func TestComplexSubquery(t *testing.T) {
	// 复杂子查询场景
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

	// 验证 SELECT 项
	if len(selectStmt.Items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(selectStmt.Items))
	}

	// 验证各个项
	t.Logf("Item[0]: %s", selectStmt.Items[0].Text)
	t.Logf("Item[1]: %s", selectStmt.Items[1].Text)
	t.Logf("Item[2]: %s (alias: %s)", selectStmt.Items[2].Text, selectStmt.Items[2].Alias)
	t.Logf("Item[3]: %s (alias: %s)", selectStmt.Items[3].Text, selectStmt.Items[3].Alias)

	// 验证 WHERE
	if selectStmt.Where == nil {
		t.Fatal("expected WHERE clause")
	}
	t.Logf("WHERE: %s", selectStmt.Where.Text)

	// 验证 ORDER BY
	if len(selectStmt.OrderBy) != 1 {
		t.Fatalf("expected 1 order by, got %d", len(selectStmt.OrderBy))
	}
	if selectStmt.OrderBy[0].Text != "u.name" {
		t.Errorf("expected order by text='u.name', got '%s'", selectStmt.OrderBy[0].Text)
	}
}
