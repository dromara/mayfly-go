package dm

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== INSERT 测试 ==========

func TestDmInsertBasic(t *testing.T) {
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

func TestDmInsertMultipleRows(t *testing.T) {
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

func TestDmUpdateBasic(t *testing.T) {
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
		t.Errorf("expected table='users', got '%s'", updateStmt.Tables[0].Name)
	}
	if updateStmt.Where == nil || updateStmt.Where.Text != "id = 1" {
		t.Errorf("expected WHERE='id = 1'")
	}
}

func TestDmUpdateMultipleColumns(t *testing.T) {
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

func TestDmUpdateReturning(t *testing.T) {
	sql := "UPDATE users SET status = 0 WHERE id = 1 RETURNING id, name, status"
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

// ========== DELETE 测试 ==========

func TestDmDeleteBasic(t *testing.T) {
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

func TestDmDeleteReturning(t *testing.T) {
	sql := "DELETE FROM users WHERE status = 0 RETURNING id, name"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil || deleteStmt.Where.Text != "status = 0" {
		t.Errorf("expected WHERE='status = 0'")
	}
}

// ========== 复杂 DML 测试 ==========

func TestDmComplexUpdateWithSubquery(t *testing.T) {
	sql := "UPDATE users SET total_orders = (SELECT COUNT(*) FROM orders WHERE orders.user_id = users.id) WHERE EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id)"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if len(updateStmt.Tables) != 1 || updateStmt.Tables[0].Name != "users" {
		t.Errorf("expected table='users', got '%s'", updateStmt.Tables[0].Name)
	}
}

func TestDmComplexDeleteWithSubquery(t *testing.T) {
	sql := "DELETE FROM users WHERE id NOT IN (SELECT DISTINCT user_id FROM orders)"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
}

// ========== 双引号标识符测试 ==========

func TestDmDoubleQuoteDML(t *testing.T) {
	sql := `UPDATE "users" SET "name" = 'John' WHERE "id" = 1`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	// 注意：DM 解析器对于双引号标识符可能保留引号
	if len(updateStmt.Tables) != 1 {
		t.Fatal("expected 1 table")
	}
	t.Logf("UPDATE table: %s", updateStmt.Tables[0].Name)
	t.Logf("UPDATE WHERE: %+v", updateStmt.Where)
}

// ========== 复杂 INSERT 测试 ==========

func TestDmInsertWithDoubleQuotes(t *testing.T) {
	sql := `INSERT INTO "users" ("name", "age", "email") VALUES ('John', 30, 'john@example.com')`
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

func TestDmInsertWithSpecialChars(t *testing.T) {
	sql := `INSERT INTO "logs" ("message", "level") VALUES ('Error: connection failed!', 'ERROR')`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	t.Logf("INSERT text: %s", insertStmt.GetText())
}

func TestDmInsertMultipleRowsComplex(t *testing.T) {
	// 多行插入包含特殊字符
	sql := `INSERT INTO "users" ("name", "email") VALUES ('John', 'john@example.com'), ('Jane', 'jane@test.com'), ('Bob', 'bob@demo.com')`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	t.Logf("Multiple INSERT: %s", insertStmt.GetText())
}

func TestDmInsertReturningComplex(t *testing.T) {
	sql := `INSERT INTO "users" ("name", "email") VALUES ('John', 'john@example.com') RETURNING "id", "name", "created_at"`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	t.Logf("INSERT RETURNING: %s", insertStmt.GetText())
}

// ========== 复杂 UPDATE 测试 ==========

func TestDmUpdateWithDoubleQuotes(t *testing.T) {
	sql := `UPDATE "users" SET "name" = 'John', "age" = 30 WHERE "id" = 1`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if len(updateStmt.Tables) != 1 {
		t.Fatal("expected 1 table")
	}
	t.Logf("UPDATE table: %s", updateStmt.Tables[0].Name)
}

func TestDmUpdateWithComplexWhere(t *testing.T) {
	sql := `UPDATE "orders" SET "status" = 'cancelled' WHERE "status" = 'pending' AND "created_at" < '2024-01-01' AND ("amount" < 100 OR "user_id" IS NULL)`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if updateStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	t.Logf("Complex WHERE: %s", updateStmt.Where.Text)
}

func TestDmUpdateWithFunctions(t *testing.T) {
	sql := `UPDATE "users" SET "updated_at" = NOW(), "login_count" = "login_count" + 1 WHERE "id" = 1`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	t.Logf("UPDATE with functions: %s", updateStmt.GetText())
}

func TestDmUpdateReturningComplex(t *testing.T) {
	sql := `UPDATE "users" SET "status" = 0 WHERE "id" = 1 RETURNING "id", "name", "status"`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if updateStmt.Where == nil || updateStmt.Where.Text != `"id" = 1` {
		t.Errorf("expected WHERE")
	}
}

func TestDmUpdateWithSubquery(t *testing.T) {
	sql := `UPDATE "users" SET "total" = (SELECT SUM("amount") FROM "orders" WHERE "user_id" = "users"."id") WHERE "status" = 'active'`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if len(updateStmt.Tables) != 1 || updateStmt.Tables[0].Name != "users" {
		t.Errorf("expected table='users', got '%s'", updateStmt.Tables[0].Name)
	}
}

// ========== 复杂 DELETE 测试 ==========

func TestDmDeleteWithDoubleQuotes(t *testing.T) {
	sql := `DELETE FROM "users" WHERE "id" = 1`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if len(deleteStmt.Tables) != 1 || deleteStmt.Tables[0].Name != "users" {
		t.Errorf("expected table='users'")
	}
}

func TestDmDeleteWithComplexWhere(t *testing.T) {
	sql := `DELETE FROM "logs" WHERE "created_at" < '2024-01-01' AND ("level" = 'DEBUG' OR "level" = 'INFO')`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	t.Logf("Complex DELETE WHERE: %s", deleteStmt.Where.Text)
}

func TestDmDeleteReturningComplex(t *testing.T) {
	sql := `DELETE FROM "users" WHERE "status" = 0 RETURNING "id", "name"`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil || deleteStmt.Where.Text != `"status" = 0` {
		t.Errorf("expected WHERE")
	}
}

func TestDmDeleteWithSubquery(t *testing.T) {
	sql := `DELETE FROM "users" WHERE "id" NOT IN (SELECT DISTINCT "user_id" FROM "orders")`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
}

// ========== 复杂 DDL 测试 ==========

func TestDmDDLCreateTableWithQuotes(t *testing.T) {
	sql := `CREATE TABLE "users" ("id" INT PRIMARY KEY, "name" VARCHAR(50) NOT NULL, "email" VARCHAR(255) UNIQUE)`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ddlStmt := stmt.(*sqlstmt.DdlStmt)
	if ddlStmt.DdlKind != "CREATE" {
		t.Errorf("expected DdlKind='CREATE'")
	}
}

func TestDmDDLCreateTableWithComments(t *testing.T) {
	sql := `CREATE TABLE "orders" ("id" INT COMMENT '订单ID', "amount" DECIMAL(10,2) COMMENT '金额') COMMENT='订单表'`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ddlStmt := stmt.(*sqlstmt.DdlStmt)
	t.Logf("CREATE TABLE with comments: %s", ddlStmt.GetText())
}

func TestDmDDLAlterTableAddColumn(t *testing.T) {
	sql := `ALTER TABLE "users" ADD COLUMN "email" VARCHAR(255)`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ddlStmt := stmt.(*sqlstmt.DdlStmt)
	if ddlStmt.DdlKind != "ALTER" {
		t.Errorf("expected DdlKind='ALTER'")
	}
}

func TestDmDDLDropIfExists(t *testing.T) {
	sql := `DROP TABLE IF EXISTS "users"`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ddlStmt := stmt.(*sqlstmt.DdlStmt)
	if ddlStmt.DdlKind != "DROP" {
		t.Errorf("expected DdlKind='DROP'")
	}
}

// ========== 注释风格测试 ==========

func TestDmDMLWithSingleLineComment(t *testing.T) {
	// DM 单行注释 --
	sql := "-- 更新用户信息\nUPDATE \"users\" SET \"name\" = 'John' WHERE \"id\" = 1"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if len(updateStmt.Tables) != 1 {
		t.Fatal("expected 1 table")
	}
	t.Logf("UPDATE with -- comment: %s", updateStmt.GetText())
}

func TestDmDMLWithMultiLineComment(t *testing.T) {
	// 多行注释 /* */
	sql := "/* 删除日志 */ DELETE FROM \"logs\" WHERE \"created_at\" < '2024-01-01'"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	t.Logf("DELETE with /* */ comment: %s", deleteStmt.GetText())
}

func TestDmDMLWithInlineComment(t *testing.T) {
	// 行内注释
	sql := `SELECT "id", "name" /* 用户名 */ FROM "users" WHERE "status" = 1`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if len(selectStmt.Items) != 2 {
		t.Fatalf("expected 2 items")
	}
	t.Logf("SELECT with inline comment: %s", selectStmt.GetText())
}

func TestDmDMLWithMultipleComments(t *testing.T) {
	// 多个注释
	sql := `-- 查询订单
/* 只查询已支付的 */
SELECT "id", "amount" FROM "orders" 
WHERE "status" = 'paid' AND "created_at" > '2024-01-01'
ORDER BY "created_at" DESC`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	selectStmt := stmt.(*sqlstmt.SelectStmt)
	if selectStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	t.Logf("Multiple comments: %s", selectStmt.GetText())
}

func TestDmInsertWithComment(t *testing.T) {
	sql := "-- 插入数据\nINSERT INTO \"users\" (\"name\", \"email\") VALUES ('John', 'john@example.com')"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	t.Logf("INSERT with comment: %s", insertStmt.GetText())
}

// ========== Schema.Table 测试 ==========

func TestDmInsertWithSchema(t *testing.T) {
	sql := `INSERT INTO "TEST"."users" ("name", "age") VALUES ('John', 30)`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	if insertStmt.Table.Schema != "TEST" {
		t.Errorf("expected schema='TEST', got '%s'", insertStmt.Table.Schema)
	}
	if insertStmt.Table.Name != "users" {
		t.Errorf("expected table='users', got '%s'", insertStmt.Table.Name)
	}
}

func TestDmUpdateWithSchema(t *testing.T) {
	sql := `UPDATE "TEST"."t_db" SET "name" = 'test' WHERE "id" = 5`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)

	t.Logf("UPDATE text: %s", updateStmt.GetText())
	t.Logf("UPDATE tables: %+v", updateStmt.Tables)
	t.Logf("UPDATE WHERE: %+v", updateStmt.Where)

	if len(updateStmt.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(updateStmt.Tables))
	}
	if updateStmt.Tables[0].Schema != "TEST" {
		t.Errorf("expected schema='TEST', got '%s'", updateStmt.Tables[0].Schema)
	}
	if updateStmt.Tables[0].Name != "t_db" {
		t.Errorf("expected table='t_db', got '%s'", updateStmt.Tables[0].Name)
	}
	if updateStmt.Where == nil {
		t.Errorf("expected WHERE")
	}
}

func TestDmDeleteWithSchema(t *testing.T) {
	sql := `DELETE FROM "TEST"."logs" WHERE "created_at" < '2024-01-01'`
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if len(deleteStmt.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(deleteStmt.Tables))
	}
	if deleteStmt.Tables[0].Schema != "TEST" {
		t.Errorf("expected schema='TEST', got '%s'", deleteStmt.Tables[0].Schema)
	}
	if deleteStmt.Tables[0].Name != "logs" {
		t.Errorf("expected table='logs', got '%s'", deleteStmt.Tables[0].Name)
	}
	if deleteStmt.Where == nil {
		t.Errorf("expected WHERE")
	}
}

// TestDmUpdateDeleteLimitCapture 断言达梦 UPDATE/DELETE 的尾随 LIMIT 被结构化捕获（采集层据此 fail-closed），
// 且无 LIMIT 时 Limit 为 nil。达梦 MySQL 兼容模式下 DELETE … LIMIT n 只删 WHERE 匹配集的子集，
// 若不捕获 LIMIT，旧值回查会捞回全部匹配行，回滚为未被删除的行生成 INSERT（撞主键或插入重复行）。
func TestDmUpdateDeleteLimitCapture(t *testing.T) {
	u, err := NewParser("UPDATE users SET name = 'a' WHERE id > 1 LIMIT 10").Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	us := u.(*sqlstmt.UpdateStmt)
	if us.Limit == nil || us.Limit.Count != 10 {
		t.Errorf("expected UPDATE Limit.Count=10, got %+v", us.Limit)
	}
	if us.Where == nil || us.Where.Text != "id > 1" {
		t.Errorf("WHERE.Text polluted by LIMIT: %+v", us.Where)
	}

	d, err := NewParser("DELETE FROM users WHERE id > 1 LIMIT 5").Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	ds := d.(*sqlstmt.DeleteStmt)
	if ds.Limit == nil || ds.Limit.Count != 5 {
		t.Errorf("expected DELETE Limit.Count=5, got %+v", ds.Limit)
	}

	// 无 LIMIT：必须为 nil，普通达梦 UPDATE/DELETE 不得被误判为带行限制而 fail-closed
	u2, _ := NewParser("UPDATE users SET name = 'a' WHERE id = 1").Parse()
	if l := u2.(*sqlstmt.UpdateStmt).Limit; l != nil {
		t.Errorf("expected nil Limit for UPDATE without LIMIT, got %+v", l)
	}
	d2, _ := NewParser("DELETE FROM users WHERE id = 1").Parse()
	if l := d2.(*sqlstmt.DeleteStmt).Limit; l != nil {
		t.Errorf("expected nil Limit for DELETE without LIMIT, got %+v", l)
	}
}
