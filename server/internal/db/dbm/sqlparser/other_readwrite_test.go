package sqlparser_test

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/clickhouse"
	"mayfly-go/internal/db/dbm/sqlparser/dm"
	"mayfly-go/internal/db/dbm/sqlparser/mssql"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/internal/db/dbm/sqlparser/oracle"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlite"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// TestOtherStmtReadWriteSplit 未建模语句的读写分流：写语句不得再落只读通道。
//
// 这是本轮修复的核心回归锁——LOAD DATA / COPY FROM / CALL 一旦被判成 other-read，
// 就会走 checkSQLTrigger("read") 与「审计记为 Query」的只读分支，绕过 DML 审批。
func TestOtherStmtReadWriteSplit(t *testing.T) {
	parsers := map[string]sqlparser.SQLParser{
		"mysql":      new(mysql.MysqlParser),
		"pgsql":      new(pgsql.PgsqlParser),
		"oracle":     new(oracle.OracleParser),
		"dm":         new(dm.DmParser),
		"mssql":      new(mssql.MssqlParser),
		"sqlite":     new(sqlite.SqliteParser),
		"clickhouse": new(clickhouse.ClickhouseParser),
	}
	splitter := sqlparser.NewSplitter(tokenizer.MysqlConfig)

	cases := []struct {
		sql  string
		want sqlstmt.StmtType
	}{
		{"SHOW TABLES", sqlstmt.StmtTypeOtherRead},
		{"SHOW CREATE TABLE t", sqlstmt.StmtTypeOtherRead},
		{"EXPLAIN SELECT 1", sqlstmt.StmtTypeOtherRead},
		{"DESC t", sqlstmt.StmtTypeOtherRead},
		{"SET @a = 1", sqlstmt.StmtTypeOtherRead},
		{"BEGIN", sqlstmt.StmtTypeOtherRead},
		{"COMMIT", sqlstmt.StmtTypeOtherRead},
		{"VACUUM t", sqlstmt.StmtTypeOtherRead},
		// 真执行内层语句：不能当只读放行
		{"EXPLAIN ANALYZE SELECT 1", sqlstmt.StmtTypeOtherWrite},
		// 写数据的未建模语句
		{"CALL p_proc(1)", sqlstmt.StmtTypeOtherWrite},
		{"LOAD DATA INFILE '/tmp/a.csv' INTO TABLE t", sqlstmt.StmtTypeOtherWrite},
		{"COPY t FROM '/tmp/a.csv'", sqlstmt.StmtTypeOtherWrite},
		{"DO SLEEP(1)", sqlstmt.StmtTypeOtherWrite},
		{"PREPARE s FROM 'SELECT 1'", sqlstmt.StmtTypeOtherWrite},
	}
	for name, parser := range parsers {
		for _, c := range cases {
			stmt, err := parser.Parse(c.sql)
			got := sqlparser.Classify(parser, splitter, c.sql, stmt, err)
			if got != c.want {
				t.Errorf("%s: %q 分类为 %q，期望 %q", name, c.sql, got, c.want)
			}
		}
	}
}

// TestUnknownLeadingKeywordIsNotReadOnly 完全未知的首词必须返回空串（由执行层按可能写处理），
// 不能返回只读类型——否则新方言/新语法会以「未知」为名溜进只读通道。
func TestUnknownLeadingKeywordIsNotReadOnly(t *testing.T) {
	parser := new(mysql.MysqlParser)
	splitter := sqlparser.NewSplitter(tokenizer.MysqlConfig)
	sql := "SOME_FUTURE_STATEMENT x"
	stmt, err := parser.Parse(sql)
	got := sqlparser.Classify(parser, splitter, sql, stmt, err)
	if got == sqlstmt.StmtTypeOtherRead || got == sqlstmt.StmtTypeSelect {
		t.Fatalf("未知语句 %q 被判为只读类型 %q", sql, got)
	}
}

// TestMultiStatementIsNotReadOnly 多语句串不得按首句判只读。
//
// 解析器只解析到首个分号（各方言单语句语义），若调用方违约传入整串，
// "SELECT 1; DELETE FROM t" 曾会被判为 select 而整体交给驱动执行（PG simple query 会真删）。
func TestMultiStatementIsNotReadOnly(t *testing.T) {
	parser := new(mysql.MysqlParser)
	splitter := sqlparser.NewSplitter(tokenizer.MysqlConfig)
	for _, sql := range []string{"SELECT 1; DELETE FROM t", "SELECT 1; DROP TABLE t"} {
		stmt, err := parser.Parse(sql)
		got := sqlparser.Classify(parser, splitter, sql, stmt, err)
		if got == sqlstmt.StmtTypeSelect || got == sqlstmt.StmtTypeOtherRead {
			t.Fatalf("多语句 %q 被判为只读 %q", sql, got)
		}
	}
	// 单条带尾分号的语句不受影响
	stmt, err := parser.Parse("SELECT 1;")
	if got := sqlparser.Classify(parser, splitter, "SELECT 1;", stmt, err); got != sqlstmt.StmtTypeSelect {
		t.Fatalf("单条语句被误判: %q", got)
	}
}

// TestDdlRenamePairs 表级改名对：建 AST 侧真源，替代 application 层的文本正则。
//
// 关键反例：`RENAME COLUMN a TO b` 是列改名，绝不能被当成表改名（否则表级策略在列名上误命中）。
func TestDdlRenamePairs(t *testing.T) {
	parser := new(mysql.MysqlParser)

	stmt, err := parser.Parse("RENAME TABLE a TO b, c TO d")
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	ddl := stmt.(*sqlstmt.DdlStmt)
	if len(ddl.Renames) != 2 || ddl.Renames[0] != (sqlstmt.Rename{From: "a", To: "b"}) || ddl.Renames[1] != (sqlstmt.Rename{From: "c", To: "d"}) {
		t.Fatalf("多对改名解析错误: %+v", ddl.Renames)
	}

	stmt, err = parser.Parse("ALTER TABLE t1 RENAME TO t2")
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	ddl = stmt.(*sqlstmt.DdlStmt)
	if len(ddl.Renames) != 1 || ddl.Renames[0] != (sqlstmt.Rename{From: "t1", To: "t2"}) {
		t.Fatalf("ALTER 改名解析错误: %+v", ddl.Renames)
	}

	// 列改名不得产生表改名对
	stmt, err = parser.Parse("ALTER TABLE t1 RENAME COLUMN c1 TO c2")
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	ddl = stmt.(*sqlstmt.DdlStmt)
	if len(ddl.Renames) != 0 {
		t.Fatalf("列改名被误当成表改名: %+v", ddl.Renames)
	}
}
