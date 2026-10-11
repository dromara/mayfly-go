package mysql

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== INSERT 测试 ==========

func TestInsertBasic(t *testing.T) {
	sql := "INSERT INTO users (name, age) VALUES ('John', 30)"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt, ok := stmt.(*sqlstmt.InsertStmt)
	if !ok {
		t.Fatal("expected InsertStmt")
	}
	if insertStmt.GetText() != sql {
		t.Errorf("expected text='%s', got '%s'", sql, insertStmt.GetText())
	}
}

func TestInsertMultipleRows(t *testing.T) {
	sql := "INSERT INTO users (name, age) VALUES ('John', 30), ('Jane', 25), ('Bob', 35)"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	if insertStmt.GetText() != sql {
		t.Errorf("expected text='%s'", sql)
	}
}

// ========== UPDATE 测试 ==========

func TestUpdateBasic(t *testing.T) {
	sql := "UPDATE users SET name = 'John' WHERE id = 1"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)

	if updateStmt.GetText() != sql {
		t.Errorf("expected text='%s', got '%s'", sql, updateStmt.GetText())
	}
	if len(updateStmt.Tables) != 1 || updateStmt.Tables[0].Name != "users" {
		t.Errorf("expected table='users'")
	}
	if updateStmt.Where == nil || updateStmt.Where.Text != "id = 1" {
		t.Errorf("expected WHERE='id = 1'")
	}
}

func TestUpdateMultipleColumns(t *testing.T) {
	sql := "UPDATE users SET name = 'John', age = 30, email = 'john@example.com' WHERE id = 1"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if updateStmt.Where == nil || updateStmt.Where.Text != "id = 1" {
		t.Errorf("expected WHERE='id = 1'")
	}
}

func TestUpdateWithOrderByLimit(t *testing.T) {
	sql := "UPDATE users SET status = 0 WHERE status = 1 ORDER BY id LIMIT 10"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)

	if updateStmt.Where == nil || updateStmt.Where.Text != "status = 1" {
		t.Errorf("expected WHERE='status = 1'")
	}
	// LIMIT 必须被结构化捕获：带行限制的 UPDATE 只影响 WHERE 匹配集的非确定子集，
	// 采集层据此 fail-closed，否则会为未被更新的行生成覆盖式回滚
	if updateStmt.Limit == nil || updateStmt.Limit.Count != 10 {
		t.Errorf("expected Limit.Count=10, got %+v", updateStmt.Limit)
	}
	// WHERE.Text 不得吞入尾随的 ORDER BY/LIMIT（回查 SELECT 逐字回放 WHERE.Text）
	if updateStmt.Where.Text != "status = 1" {
		t.Errorf("WHERE.Text polluted by ORDER BY/LIMIT: %q", updateStmt.Where.Text)
	}
}

func TestUpdateNoLimitIsNil(t *testing.T) {
	stmt, err := NewParser("UPDATE users SET name = 'John' WHERE id = 1").Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if u := stmt.(*sqlstmt.UpdateStmt); u.Limit != nil {
		t.Errorf("expected nil Limit for UPDATE without LIMIT, got %+v", u.Limit)
	}
}

// ========== DELETE 测试 ==========

func TestDeleteBasic(t *testing.T) {
	sql := "DELETE FROM users WHERE id = 1"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)

	if deleteStmt.GetText() != sql {
		t.Errorf("expected text='%s', got '%s'", sql, deleteStmt.GetText())
	}
	if len(deleteStmt.Tables) != 1 || deleteStmt.Tables[0].Name != "users" {
		t.Errorf("expected table='users'")
	}
	if deleteStmt.Where == nil || deleteStmt.Where.Text != "id = 1" {
		t.Errorf("expected WHERE='id = 1'")
	}
}

func TestDeleteMultipleTables(t *testing.T) {
	sql := "DELETE t1, t2 FROM users t1 JOIN orders t2 ON t1.id = t2.user_id WHERE t1.status = 0"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)

	if len(deleteStmt.Tables) < 1 {
		t.Errorf("expected at least 1 table")
	}
}

func TestDeleteWithOrderByLimit(t *testing.T) {
	sql := "DELETE FROM users WHERE status = 0 ORDER BY id LIMIT 100"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)

	if deleteStmt.Where == nil || deleteStmt.Where.Text != "status = 0" {
		t.Errorf("expected WHERE='status = 0'")
	}
	// LIMIT 必须被结构化捕获：DELETE … LIMIT n 只删 WHERE 匹配集的一个子集，
	// 采集回查若捞回全部匹配行，回滚会为未被删除的行生成 INSERT（撞主键或插入重复行）
	if deleteStmt.Limit == nil || deleteStmt.Limit.Count != 100 {
		t.Errorf("expected Limit.Count=100, got %+v", deleteStmt.Limit)
	}
	if deleteStmt.Where.Text != "status = 0" {
		t.Errorf("WHERE.Text polluted by ORDER BY/LIMIT: %q", deleteStmt.Where.Text)
	}
}

func TestDeleteNoLimitIsNil(t *testing.T) {
	stmt, err := NewParser("DELETE FROM users WHERE id = 1").Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if d := stmt.(*sqlstmt.DeleteStmt); d.Limit != nil {
		t.Errorf("expected nil Limit for DELETE without LIMIT, got %+v", d.Limit)
	}
}

// ========== DDL 测试 ==========

func TestDDLCreate(t *testing.T) {
	sql := "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50), age INT)"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ddlStmt := stmt.(*sqlstmt.DdlStmt)

	if ddlStmt.DdlKind != "CREATE" {
		t.Errorf("expected DdlKind='CREATE', got '%s'", ddlStmt.DdlKind)
	}
	if ddlStmt.GetText() != sql {
		t.Errorf("expected text='%s'", sql)
	}
}

func TestDDLDrop(t *testing.T) {
	sql := "DROP TABLE users"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ddlStmt := stmt.(*sqlstmt.DdlStmt)

	if ddlStmt.DdlKind != "DROP" {
		t.Errorf("expected DdlKind='DROP', got '%s'", ddlStmt.DdlKind)
	}
}

func TestDDLAlter(t *testing.T) {
	sql := "ALTER TABLE users ADD COLUMN email VARCHAR(255)"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ddlStmt := stmt.(*sqlstmt.DdlStmt)

	if ddlStmt.DdlKind != "ALTER" {
		t.Errorf("expected DdlKind='ALTER', got '%s'", ddlStmt.DdlKind)
	}
}

func TestDDLTruncate(t *testing.T) {
	sql := "TRUNCATE TABLE users"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// TRUNCATE 可能返回 DdlStmt 或其他类型
	if stmt == nil {
		t.Fatalf("expected stmt not nil")
	}
	t.Logf("TRUNCATE stmt type: %T", stmt)
	t.Logf("TRUNCATE text: %s", stmt.GetText())
}

// TestInsertConflictClause 逐形态断言 INSERT 的冲突处理标志：
// 带 REPLACE / INSERT IGNORE / ON DUPLICATE KEY UPDATE 的语句在目标行已存在时并不新增行，
// 而 INSERT 的回滚是「删掉刚插入的行」，按 VALUES 字面量生成的 DELETE 会命中并删除既有行（数据丢失），
// 故这些形态必须被标记，采集层据此 fail-closed 跳过插入快照；纯 INSERT 不标记、正常采集。
func TestInsertConflictClause(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want bool
	}{
		{"plain", "INSERT INTO users (id, name) VALUES (1, 'a')", false},
		{"multi-row", "INSERT INTO users (id, name) VALUES (1, 'a'), (2, 'b')", false},
		{"replace", "REPLACE INTO users (id, name) VALUES (1, 'a')", true},
		{"insert-ignore", "INSERT IGNORE INTO users (id, name) VALUES (1, 'a')", true},
		{"insert-ignore-no-into", "INSERT IGNORE users (id, name) VALUES (1, 'a')", true},
		{"on-duplicate", "INSERT INTO users (id, name) VALUES (1, 'a') ON DUPLICATE KEY UPDATE name = 'b'", true},
		{"insert-select-plain", "INSERT INTO users (id, name) SELECT id, name FROM src", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stmt, err := NewParser(c.sql).Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			is, ok := stmt.(*sqlstmt.InsertStmt)
			if !ok {
				t.Fatalf("expected InsertStmt, got %T", stmt)
			}
			if is.HasConflictClause != c.want {
				t.Errorf("HasConflictClause=%v, want %v", is.HasConflictClause, c.want)
			}
		})
	}
}
