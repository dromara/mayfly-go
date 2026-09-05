package dm

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ========== 非查询语句（DDL/DCL）分类测试 ==========
// 验证 COMMENT ON、GRANT 等非查询语句被解析为 DdlStmt，
// 由应用层走 doExecDDL（Exec 通道）执行，避免走查询通道被数据库驱动拒绝

func TestDmCommentOnTable(t *testing.T) {
	sql := "comment on table sys_user is '用户表'"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if _, ok := stmt.(*sqlstmt.DdlStmt); !ok {
		t.Fatalf("expected DdlStmt for '%s', got %T", sql, stmt)
	}
}

func TestDmCommentOnColumn(t *testing.T) {
	sql := "COMMENT ON COLUMN sys_user.name IS '用户名'"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ddlStmt, ok := stmt.(*sqlstmt.DdlStmt)
	if !ok {
		t.Fatalf("expected DdlStmt, got %T", stmt)
	}
	if ddlStmt.GetText() != sql {
		t.Errorf("expected text='%s', got '%s'", sql, ddlStmt.GetText())
	}
}

func TestDmGrantAndRevoke(t *testing.T) {
	sqls := []string{
		"GRANT SELECT ON sys_user TO dev_role",
		"REVOKE SELECT ON sys_user FROM dev_role",
	}
	for _, sql := range sqls {
		parser := NewParser(sql)
		stmt, err := parser.Parse()
		if err != nil {
			t.Fatalf("parse error for '%s': %v", sql, err)
		}
		if _, ok := stmt.(*sqlstmt.DdlStmt); !ok {
			t.Fatalf("expected DdlStmt for '%s', got %T", sql, stmt)
		}
	}
}

func TestDmTruncateAsDdl(t *testing.T) {
	sql := "TRUNCATE TABLE sys_user"
	parser := NewParser(sql)
	stmt, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if _, ok := stmt.(*sqlstmt.DdlStmt); !ok {
		t.Fatalf("expected DdlStmt, got %T", stmt)
	}
}
