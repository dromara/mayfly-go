package oracle

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== INSERT 测试 ==========

func TestOracleInsertBasic(t *testing.T) {
	sql := "INSERT INTO users (name, age) VALUES ('John', 30)"
	parser := newParser(sql)
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

func TestOracleInsertMultipleRows(t *testing.T) {
	sql := "INSERT INTO users (name, age) VALUES ('John', 30)"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	if insertStmt.GetText() != sql {
		t.Errorf("expected text='%s'", sql)
	}
}

func TestOracleInsertReturning(t *testing.T) {
	sql := "INSERT INTO users (name, age) VALUES ('John', 30) RETURNING id, name"
	parser := newParser(sql)
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

func TestOracleUpdateBasic(t *testing.T) {
	sql := "UPDATE users SET name = \"John\" WHERE id = 1"
	parser := newParser(sql)
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

func TestOracleUpdateMultipleColumns(t *testing.T) {
	sql := "UPDATE users SET name = 'John', age = 30, email = 'john@example.com' WHERE id = 1"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if updateStmt.Where == nil || updateStmt.Where.Text != "id = 1" {
		t.Errorf("expected WHERE='id = 1'")
	}
}

func TestOracleUpdateReturning(t *testing.T) {
	sql := "UPDATE users SET status = 0 WHERE id = 1 RETURNING id, name, status"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if updateStmt.Where == nil || updateStmt.Where.Text != "id = 1" {
		t.Errorf("expected WHERE='id = 1'")
	}
}

func TestOracleUpdateRowNum(t *testing.T) {
	// Oracle 特有的 ROWNUM 用法
	sql := "UPDATE users SET status = 0 WHERE ROWNUM <= 10 AND status = 1"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if updateStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	t.Logf("UPDATE WHERE with ROWNUM: %s", updateStmt.Where.Text)
}

// ========== DELETE 测试 ==========

func TestOracleDeleteBasic(t *testing.T) {
	sql := "DELETE FROM users WHERE id = 1"
	parser := newParser(sql)
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

func TestOracleDeleteReturning(t *testing.T) {
	sql := "DELETE FROM users WHERE status = 0 RETURNING id, name"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil || deleteStmt.Where.Text != "status = 0" {
		t.Errorf("expected WHERE='status = 0'")
	}
}

func TestOracleDeleteRowNum(t *testing.T) {
	// Oracle 特有的 ROWNUM 用法
	sql := "DELETE FROM users WHERE ROWNUM <= 100 AND status = 0"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	t.Logf("DELETE WHERE with ROWNUM: %s", deleteStmt.Where.Text)
}

// ========== MERGE INTO 测试 (Oracle 特有) ==========

func TestOracleMergeInto(t *testing.T) {
	sql := `MERGE INTO users u USING (SELECT 1 AS id, 'John' AS name FROM dual) s ON (u.id = s.id) WHEN MATCHED THEN UPDATE SET u.name = s.name WHEN NOT MATCHED THEN INSERT (id, name) VALUES (s.id, s.name)`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// MERGE INTO 可能被解析为其他类型
	if stmt == nil {
		t.Fatalf("expected stmt not nil")
	}
	t.Logf("MERGE stmt type: %T", stmt)
	t.Logf("MERGE text: %s", stmt.GetText())
}

// ========== 复杂 INSERT 测试 ==========

func TestOracleInsertWithDoubleQuotes(t *testing.T) {
	sql := `INSERT INTO "users" ("name", "age", "email") VALUES ('John', 30, 'john@example.com')`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	if insertStmt.GetText() != sql {
		t.Errorf("expected text='%s'", sql)
	}
}

func TestOracleInsertWithSpecialChars(t *testing.T) {
	sql := `INSERT INTO "logs" ("message", "level") VALUES ('Error: connection failed!', 'ERROR')`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	t.Logf("INSERT text: %s", insertStmt.GetText())
}

func TestOracleInsertReturningComplex(t *testing.T) {
	sql := `INSERT INTO "users" ("name", "email") VALUES ('John', 'john@example.com') RETURNING "id", "name", "created_at"`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	t.Logf("INSERT RETURNING: %s", insertStmt.GetText())
}

// ========== 复杂 UPDATE 测试 ==========

func TestOracleUpdateWithDoubleQuotes(t *testing.T) {
	sql := `UPDATE "users" SET "name" = 'John', "age" = 30 WHERE "id" = 1`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if len(updateStmt.Tables) != 1 || updateStmt.Tables[0].Name != "users" {
		t.Errorf("expected table='users', got '%s'", updateStmt.Tables[0].Name)
	}
}

func TestOracleUpdateWithComplexWhere(t *testing.T) {
	sql := `UPDATE "orders" SET "status" = 'cancelled' WHERE "status" = 'pending' AND "created_at" < TO_DATE('2024-01-01', 'YYYY-MM-DD')`
	parser := newParser(sql)
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

func TestOracleUpdateReturningComplex(t *testing.T) {
	sql := `UPDATE "users" SET "status" = 0 WHERE "id" = 1 RETURNING "id", "name", "status"`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if updateStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
}

func TestOracleUpdateWithSubquery(t *testing.T) {
	sql := `UPDATE "users" SET "total" = (SELECT SUM("amount") FROM "orders" WHERE "user_id" = "users"."id") WHERE "status" = 'active'`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if len(updateStmt.Tables) != 1 {
		t.Fatal("expected table")
	}
}

func TestOracleUpdateRowNumComplex(t *testing.T) {
	sql := `UPDATE "users" SET "status" = 0 WHERE ROWNUM <= 10 AND "status" = 1`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if updateStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
}

// ========== 复杂 DELETE 测试 ==========

func TestOracleDeleteWithDoubleQuotes(t *testing.T) {
	sql := `DELETE FROM "users" WHERE "id" = 1`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if len(deleteStmt.Tables) != 1 || deleteStmt.Tables[0].Name != "users" {
		t.Errorf("expected table='users'")
	}
}

func TestOracleDeleteReturningComplex(t *testing.T) {
	sql := `DELETE FROM "users" WHERE "status" = 0 RETURNING "id", "name"`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
}

func TestOracleDeleteWithSubquery(t *testing.T) {
	sql := `DELETE FROM "users" WHERE "id" NOT IN (SELECT DISTINCT "user_id" FROM "orders")`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
}

func TestOracleDeleteRowNumComplex(t *testing.T) {
	sql := `DELETE FROM "logs" WHERE ROWNUM <= 1000 AND "level" = 'DEBUG'`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if deleteStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
}

// ========== Schema.Table 测试 ==========

func TestOracleInsertWithSchema(t *testing.T) {
	sql := `INSERT INTO "SCOTT"."users" ("name", "age") VALUES ('John', 30)`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	if insertStmt.Table.Schema != "SCOTT" {
		t.Errorf("expected schema='SCOTT', got '%s'", insertStmt.Table.Schema)
	}
	if insertStmt.Table.Name != "users" {
		t.Errorf("expected table='users', got '%s'", insertStmt.Table.Name)
	}
}

func TestOracleUpdateWithSchema(t *testing.T) {
	sql := `UPDATE "SCOTT"."t_db" SET "name" = 'test' WHERE "id" = 5`
	parser := newParser(sql)
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
	if updateStmt.Tables[0].Schema != "SCOTT" {
		t.Errorf("expected schema='SCOTT', got '%s'", updateStmt.Tables[0].Schema)
	}
	if updateStmt.Tables[0].Name != "t_db" {
		t.Errorf("expected table='t_db', got '%s'", updateStmt.Tables[0].Name)
	}
	if updateStmt.Where == nil {
		t.Errorf("expected WHERE")
	}
}

func TestOracleDeleteWithSchema(t *testing.T) {
	sql := `DELETE FROM "SCOTT"."logs" WHERE "created_at" < TO_DATE('2024-01-01', 'YYYY-MM-DD')`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deleteStmt := stmt.(*sqlstmt.DeleteStmt)
	if len(deleteStmt.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(deleteStmt.Tables))
	}
	if deleteStmt.Tables[0].Schema != "SCOTT" {
		t.Errorf("expected schema='SCOTT', got '%s'", deleteStmt.Tables[0].Schema)
	}
	if deleteStmt.Tables[0].Name != "logs" {
		t.Errorf("expected table='logs', got '%s'", deleteStmt.Tables[0].Name)
	}
	if deleteStmt.Where == nil {
		t.Errorf("expected WHERE")
	}
}

// ========== 复杂 DDL 测试 ==========

func TestOracleDDLCreateTableWithQuotes(t *testing.T) {
	sql := `CREATE TABLE "users" ("id" INT PRIMARY KEY, "name" VARCHAR2(50) NOT NULL)`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ddlStmt := stmt.(*sqlstmt.DdlStmt)
	if ddlStmt.DdlKind != "CREATE" {
		t.Errorf("expected DdlKind='CREATE'")
	}
}

func TestOracleDDLAlterTableAddColumn(t *testing.T) {
	sql := `ALTER TABLE "users" ADD ("email" VARCHAR2(255))`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ddlStmt := stmt.(*sqlstmt.DdlStmt)
	if ddlStmt.DdlKind != "ALTER" {
		t.Errorf("expected DdlKind='ALTER'")
	}
}

func TestOracleDDLDropCascade(t *testing.T) {
	sql := `DROP TABLE "users" CASCADE CONSTRAINTS`
	parser := newParser(sql)
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

func TestOracleDMLWithSingleLineComment(t *testing.T) {
	// Oracle 单行注释 --
	sql := "-- 更新用户信息\nUPDATE \"users\" SET \"name\" = 'John' WHERE \"id\" = 1"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if len(updateStmt.Tables) != 1 || updateStmt.Tables[0].Name != "users" {
		t.Errorf("expected table='users'")
	}
	t.Logf("UPDATE with -- comment: %s", updateStmt.GetText())
}

func TestOracleDMLWithMultiLineComment(t *testing.T) {
	// 多行注释 /* */
	sql := "/* 删除过期数据 */ DELETE FROM \"logs\" WHERE \"created_at\" < SYSDATE - 90"
	parser := newParser(sql)
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

func TestOracleDMLWithInlineComment(t *testing.T) {
	// 行内注释
	sql := `SELECT "id", "name" /* 用户名 */ FROM "users" WHERE "status" = 1`
	parser := newParser(sql)
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

func TestOracleDMLWithMultipleComments(t *testing.T) {
	// 多个注释
	sql := `-- 查询订单
/* 只查询已支付的 */
SELECT "id", "amount" FROM "orders" 
WHERE "status" = 'PAID' AND "created_at" > TO_DATE('2024-01-01', 'YYYY-MM-DD')
ORDER BY "created_at" DESC`
	parser := newParser(sql)
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

func TestOracleInsertWithComment(t *testing.T) {
	sql := "-- 插入新用户\nINSERT INTO \"users\" (\"name\", \"email\") VALUES ('John', 'john@example.com')"
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	insertStmt := stmt.(*sqlstmt.InsertStmt)
	t.Logf("INSERT with comment: %s", insertStmt.GetText())
}

func TestOracleUpdateWithComment(t *testing.T) {
	sql := `/* 批量更新 */
UPDATE "orders" SET "status" = 'CANCELLED' 
WHERE "status" = 'PENDING' -- 只更新待处理订单
AND "created_at" < SYSDATE - 30`
	parser := newParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	updateStmt := stmt.(*sqlstmt.UpdateStmt)
	if updateStmt.Where == nil {
		t.Fatal("expected WHERE")
	}
	t.Logf("UPDATE with multiple comments: %s", updateStmt.GetText())
}
