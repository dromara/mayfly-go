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
)

// TestDdlColumns CREATE TABLE 列定义提取（7 方言一致）。
//
// 关键约束：
//   - 表级约束项（PRIMARY KEY(...)/UNIQUE/KEY/INDEX/CONSTRAINT/FOREIGN/CHECK）不能被当成列；
//   - 表级 PRIMARY KEY(a,b) 要回填到对应列的 Primary；
//   - CREATE TABLE ... AS SELECT / LIKE 没有列清单，必须返回空而不是瞎猜；
//   - 类型文本保留原文（varchar(20)、decimal(10,2)、int unsigned），供后续展示/比对。
func TestDdlColumns(t *testing.T) {
	parsers := map[string]sqlparser.SQLParser{
		"mysql":      new(mysql.MysqlParser),
		"pgsql":      new(pgsql.PgsqlParser),
		"oracle":     new(oracle.OracleParser),
		"dm":         new(dm.DmParser),
		"mssql":      new(mssql.MssqlParser),
		"sqlite":     new(sqlite.SqliteParser),
		"clickhouse": new(clickhouse.ClickhouseParser),
	}
	for name, parser := range parsers {
		// 1) 行内约束 + 类型原文
		sql := "CREATE TABLE t1 (id int PRIMARY KEY, name varchar(20) NOT NULL DEFAULT 'x', amount decimal(10,2) NULL, code varchar(8) UNIQUE)"
		stmt, err := parser.Parse(sql)
		if err != nil {
			t.Fatalf("%s parse err: %v", name, err)
		}
		ddl := stmt.(*sqlstmt.DdlStmt)
		if len(ddl.Columns) != 4 {
			t.Fatalf("%s 列数 = %d，期望 4: %+v", name, len(ddl.Columns), ddl.Columns)
		}
		if !ddl.Columns[0].Primary || ddl.Columns[0].Type != "int" {
			t.Fatalf("%s id 列解析错误: %+v", name, ddl.Columns[0])
		}
		if ddl.Columns[1].Nullable == nil || *ddl.Columns[1].Nullable || ddl.Columns[1].Default != "'x'" {
			t.Fatalf("%s name 列解析错误: %+v", name, ddl.Columns[1])
		}
		if ddl.Columns[2].Type != "decimal(10,2)" {
			t.Fatalf("%s amount 类型原文错误: %q", name, ddl.Columns[2].Type)
		}
		if !ddl.Columns[3].Unique {
			t.Fatalf("%s code 列 UNIQUE 未识别: %+v", name, ddl.Columns[3])
		}

		// 2) 表级 PRIMARY KEY 回填 + 约束项不当列
		sql = "CREATE TABLE t2 (a int NOT NULL, b varchar(10), c int, PRIMARY KEY (a, b), KEY idx_c (c), UNIQUE KEY uk_c (c), CHECK (c > 0))"
		stmt, err = parser.Parse(sql)
		if err != nil {
			t.Fatalf("%s parse err: %v", name, err)
		}
		ddl = stmt.(*sqlstmt.DdlStmt)
		if len(ddl.Columns) != 3 {
			t.Fatalf("%s 表级约束被当成列: %+v", name, ddl.Columns)
		}
		if !ddl.Columns[0].Primary || !ddl.Columns[1].Primary || ddl.Columns[2].Primary {
			t.Fatalf("%s 表级主键回填错误: %+v", name, ddl.Columns)
		}

		// 3) CONSTRAINT 命名的主键
		sql = "CREATE TABLE t3 (a int, CONSTRAINT pk_t3 PRIMARY KEY (a))"
		stmt, err = parser.Parse(sql)
		if err != nil {
			t.Fatalf("%s parse err: %v", name, err)
		}
		ddl = stmt.(*sqlstmt.DdlStmt)
		if len(ddl.Columns) != 1 || !ddl.Columns[0].Primary {
			t.Fatalf("%s CONSTRAINT 主键未回填: %+v", name, ddl.Columns)
		}

		// 4) AS SELECT / LIKE 建表：没有列清单，不得瞎猜
		for _, noCols := range []string{"CREATE TABLE t4 AS SELECT * FROM t1", "CREATE TABLE t5 LIKE t1"} {
			stmt, err = parser.Parse(noCols)
			if err != nil {
				t.Fatalf("%s parse err: %v", name, err)
			}
			ddl = stmt.(*sqlstmt.DdlStmt)
			if len(ddl.Columns) != 0 {
				t.Fatalf("%s %q 不应产生列定义: %+v", name, noCols, ddl.Columns)
			}
		}
	}
}
