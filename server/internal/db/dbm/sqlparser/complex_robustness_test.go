package sqlparser_test

import (
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/clickhouse"
	"mayfly-go/internal/db/dbm/sqlparser/dm"
	"mayfly-go/internal/db/dbm/sqlparser/mssql"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/internal/db/dbm/sqlparser/oracle"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlite"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// TestComplexSQLRobustness 边角输入的健壮性与准确性：注释/字符串/大小写/参数占位/
// 不完整语句/超长 IN/深层括号/Unicode 标识符。目标：**不 panic、不挂起、表清单不被污染**。
func TestComplexSQLRobustness(t *testing.T) {
	parsers := []struct {
		name   string
		parser sqlparser.SQLParser
		cfg    tokenizer.DialectConfig
	}{
		{"mysql", new(mysql.MysqlParser), tokenizer.MysqlConfig},
		{"pgsql", new(pgsql.PgsqlParser), tokenizer.PgConfig},
		{"oracle", new(oracle.OracleParser), tokenizer.OracleConfig},
		{"dm", new(dm.DmParser), tokenizer.DmConfig},
		{"mssql", new(mssql.MssqlParser), tokenizer.MssqlConfig},
		{"sqlite", new(sqlite.SqliteParser), tokenizer.SQLiteConfig},
		{"clickhouse", new(clickhouse.ClickhouseParser), tokenizer.ClickhouseConfig},
	}

	longIn := "1" + strings.Repeat(",1", 999)
	cases := []struct {
		sql        string
		wantTables []string // nil = 不校验（只要求不 panic / 不污染）
	}{
		{"SELECT * FROM t WHERE a = 'x;DROP TABLE y'", []string{"t"}},
		{"SELECT * FROM t /* DROP TABLE y */ WHERE a = 1", []string{"t"}},
		{"/* lead */ SELECT * FROM t -- tail\nWHERE a = 1", []string{"t"}},
		{"select * from t", []string{"t"}},

		{"SELECT * FROM T WHERE A = 1", []string{"t"}},
		{"SELECT $1, $2 FROM t WHERE a = $3", []string{"t"}},
		{"SELECT ? FROM t WHERE a = ?", []string{"t"}},
		{"SELECT * FROM t WHERE a = -1.5e10", []string{"t"}},
		{"SELECT * FROM t WHERE a IN (" + longIn + ")", []string{"t"}},
		{"SELECT * FROM ((((((SELECT * FROM t)))))) x", nil},
		{"INSERT INTO t VALUES (1), (2), (3)", []string{"t"}},
		{"SELECT * FROM t1 JOIN (t2 JOIN t3 ON t2.id = t3.id) ON t1.id = t2.id", nil},
		{"SELECT * FROM t WHERE a = 1; -- 末尾注释", []string{"t"}},
		// 不完整 / 空 / 纯注释：不得 panic（返回 nil 或 Other 均可）
		{"", nil},
		{";", nil},
		{"-- only comment", nil},
		{"SELECT * FROM", nil},
		{"SELECT * FROM t WHERE", nil},
		{"INSERT INTO", nil},
		{"<<<>>>", nil},
	}

	// 引用符是方言能力：反引号只在 mysql/sqlite/clickhouse 是标识符引用符，
	// pg/oracle/dm 用双引号、mssql 用方括号。用各方言的合法写法分别断言。
	quoted := map[string]struct {
		sql  string
		want []string
	}{
		"mysql":      {"SELECT * FROM `中文表`", []string{"中文表"}},
		"sqlite":     {"SELECT * FROM `中文表`", []string{"中文表"}},
		"clickhouse": {"SELECT * FROM `中文表`", []string{"中文表"}},
		"pgsql":      {`SELECT * FROM "中文表"`, []string{"中文表"}},
		"oracle":     {`SELECT * FROM "中文表"`, []string{"中文表"}},
		"dm":         {`SELECT * FROM "中文表"`, []string{"中文表"}},
		"mssql":      {"SELECT * FROM [中文表]", []string{"中文表"}},
	}

	for _, p := range parsers {
		splitter := sqlparser.NewSplitter(p.cfg)
		all := append([]struct {
			sql        string
			wantTables []string
		}{}, cases...)
		if q, ok := quoted[p.name]; ok {
			all = append(all, struct {
				sql        string
				wantTables []string
			}{q.sql, q.want})
		}
		for _, c := range all {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s panic on %q: %v", p.name, c.sql, r)
					}
				}()
				stmt, err := p.parser.Parse(c.sql)
				typ := sqlparser.Classify(p.parser, splitter, c.sql, stmt, err)
				if c.wantTables == nil {
					return
				}
				if stmt == nil {
					t.Errorf("%s %q 解析结果为空（期望有表 %v）", p.name, c.sql, c.wantTables)
					return
				}
				got := tablesOf(stmt)
				if !sameSet(got, c.wantTables) {
					t.Errorf("%s 表清单错误: %q -> %v，期望 %v (type=%s)", p.name, c.sql, got, c.wantTables, typ)
				}
			}()
		}
	}
}

// FuzzParserNoPanic 随机输入下各解析器不得 panic。
//
// 语料种子取自上面与 complex_accuracy/sqlparser 既有用例的高频形态；fuzz 会做字节级变异，
// 主要用来兜住「畸形输入触发越界/死循环」这类人工语料难以穷举的问题。
func FuzzParserNoPanic(f *testing.F) {
	seeds := []string{
		"SELECT * FROM t WHERE a = 1",
		"SELECT * FROM t1 JOIN t2 ON t1.id = t2.id",
		"WITH c AS (SELECT 1) SELECT * FROM c",
		"INSERT INTO t (a) VALUES (1)",
		"UPDATE t SET a = 1 WHERE id = 2",
		"DELETE FROM t WHERE id = 1",
		"MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.v = s.v",
		"CREATE TABLE t (id int PRIMARY KEY, v varchar(20) NOT NULL DEFAULT 'x')",
		"ALTER TABLE t ADD COLUMN c int",
		"DROP TABLE IF EXISTS t",
		"RENAME TABLE a TO b, c TO d",
		"EXPLAIN ANALYZE DELETE FROM t",
		"CALL p(1)",
		"SELECT * FROM t WHERE a IN (SELECT id FROM s)",
		"SELECT * FROM `t` /* c */ WHERE a = 'x;y'",
		"SELECT TOP (10) PERCENT * FROM t",
		"SELECT * FROM t OFFSET 5 ROWS FETCH NEXT 10 ROWS ONLY",
		"/*!40101 SET NAMES utf8mb4 */",
		"SELECT \"a\" FROM \"t\"",
		"SELECT [a] FROM [dbo].[t]",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	parsers := []sqlparser.SQLParser{
		new(mysql.MysqlParser), new(pgsql.PgsqlParser), new(oracle.OracleParser),
		new(dm.DmParser), new(mssql.MssqlParser), new(sqlite.SqliteParser),
		new(clickhouse.ClickhouseParser),
	}
	f.Fuzz(func(t *testing.T, sql string) {
		if len(sql) > 4096 {
			t.Skip()
		}
		for _, p := range parsers {
			// 解析器内部已有 recover；此处断言的是「不 panic 且能在有限时间内返回」
			_, _ = p.Parse(sql)
		}
	})
}
