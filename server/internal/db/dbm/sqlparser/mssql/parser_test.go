package mssql_test

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/mssql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// TestMssqlThinDialect T-SQL 特有词法（[方括号] 标识符）与分页改写。
//
// 薄方言此前无独立测试文件，只靠 sqlparser 根目录的通用矩阵覆盖；
// 这里钉住「方括号表名进 From」「]] 转义」「OFFSET-FETCH 改写」三条 T-SQL 特征。
func TestMssqlThinDialect(t *testing.T) {
	parser := new(mssql.MssqlParser)

	stmt, err := parser.Parse("SELECT [user name] FROM [dbo].[t order] WHERE [id] > 0")
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok {
		t.Fatalf("want SelectStmt, got %T", stmt)
	}
	if len(sel.From) == 0 || sel.From[0].Name != "t order" {
		t.Fatalf("方括号表名提取错误: %+v", sel.From)
	}

	// ]] 为转义的 ]
	stmt, err = parser.Parse("SELECT * FROM [t]]x]")
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	sel = stmt.(*sqlstmt.SelectStmt)
	if len(sel.From) == 0 || sel.From[0].Name != "t]x" {
		t.Fatalf("]] 转义解析错误: %+v", sel.From)
	}

	sql, err := parser.RewritePagination("SELECT * FROM t", 10, 5)
	if err != nil {
		t.Fatalf("rewrite err: %v", err)
	}
	if sql == "SELECT * FROM t" {
		t.Fatalf("分页改写未生效: %s", sql)
	}
}
