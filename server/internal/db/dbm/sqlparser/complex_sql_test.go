package sqlparser_test

import (
	"fmt"
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
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

func TestComplexSQLMatrix(t *testing.T) {
	groups := []struct {
		dialect string
		parser  sqlparser.SQLParser
		sqls    []string
	}{
		{"pg", new(pgsql.PgsqlParser), []string{
			`WITH RECURSIVE t(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM t WHERE n < 10) SELECT * FROM t`,
			`SELECT o.id, ROW_NUMBER() OVER (PARTITION BY o.uid ORDER BY o.amt DESC) rn FROM orders o WHERE o.st IN (SELECT st FROM x WHERE y IN (SELECT z FROM y))`,
			`SELECT a FROM t1 UNION ALL SELECT b FROM t2 ORDER BY 1 LIMIT 10 OFFSET 5`,
			`SELECT a FROM t1 INTERSECT SELECT a FROM t2`,
			`SELECT * FROM t1 JOIN LATERAL (SELECT * FROM t2 WHERE t2.id = t1.id) s ON true`,
			`SELECT k, sum(v) FILTER (WHERE v > 0) FROM m GROUP BY GROUPING SETS ((k), ())`,
			`INSERT INTO t (id, v) VALUES (1, 2) ON CONFLICT (id) DO UPDATE SET v = excluded.v RETURNING id`,
			`UPDATE t SET v = s.v FROM s WHERE s.id = t.id`,
			`DELETE FROM t USING s WHERE s.id = t.id`,
			`SELECT payload->>'name', payload @> '{"a":1}' FROM j WHERE payload ? 'k'`,
			`CREATE FUNCTION f() RETURNS int AS $$ BEGIN RETURN 1; END $$ LANGUAGE plpgsql`,
		}},
		{"mysql", new(mysql.MysqlParser), []string{
			`UPDATE t1 JOIN t2 ON t1.id = t2.id SET t1.v = t2.v WHERE t2.x = 1 LIMIT 10`,
			`SELECT STRAIGHT_JOIN a.* FROM a USE INDEX (idx_a) JOIN b ON b.id = a.id LIMIT 5`,
			`CREATE PROCEDURE p() BEGIN UPDATE t SET v = 1; DELETE FROM s; END`,
			`PREPARE stmt FROM 'SELECT 1'; EXECUTE stmt`,
			`SELECT * FROM t WHERE a IN (1,2,3) AND (b = 1 OR c BETWEEN 1 AND 2) ORDER BY d DESC LIMIT 100`,
		}},
		{"oracle", new(oracle.OracleParser), []string{
			`SELECT * FROM e, d WHERE e.id = d.id(+) AND e.st = 1`,
			`SELECT id, LEVEL FROM t START WITH pid IS NULL CONNECT BY PRIOR id = pid ORDER SIBLINGS BY name`,
			`MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.v = s.v WHEN NOT MATCHED THEN INSERT (id, v) VALUES (s.id, s.v)`,
			`SELECT * FROM (SELECT a.*, ROWNUM rn FROM (SELECT * FROM t ORDER BY id) a WHERE ROWNUM <= 20) WHERE rn > 10`,
		}},
		{"mssql", new(mssql.MssqlParser), []string{
			`SELECT TOP 10 [a].[id] FROM [dbo].[t] AS [a] WHERE [a].[v] > 0 ORDER BY [a].[id]`,
			`SELECT * FROM t ORDER BY id OFFSET 20 ROWS FETCH NEXT 10 ROWS ONLY`,
			`SELECT * FROM t1 CROSS APPLY (SELECT TOP 1 * FROM t2 WHERE t2.id = t1.id) x`,
		}},
		{"clickhouse", new(clickhouse.ClickhouseParser), []string{
			"SELECT * FROM t FINAL PREWHERE a > 0 WHERE b = 1 ARRAY JOIN arr AS x LIMIT 10",
			"SELECT k, count() FROM t GROUP BY k WITH TOTALS",
		}},
		{"sqlite", new(sqlite.SqliteParser), []string{
			`INSERT OR REPLACE INTO t (id) VALUES (1) RETURNING id`,
			`WITH c AS (SELECT 1) SELECT * FROM c`,
		}},
		{"dm", new(dm.DmParser), []string{
			`SELECT * FROM t WHERE id IN (SELECT id FROM s WHERE x > 0) LIMIT 10`,
		}},
	}

	for _, g := range groups {
		for _, sql := range g.sqls {
			stmt, err := g.parser.Parse(sql)
			status := "OK"
			if err != nil {
				status = "ERR:" + err.Error()
				t.Errorf("%s 复杂语句解析失败: %q -> %v", g.dialect, sql, err)
			}
			if stmt == nil {
				t.Errorf("%s 复杂语句解析结果为空: %q", g.dialect, sql)
			}
			desc := "nil"
			if stmt != nil {
				desc = describe(stmt)
			}
			t.Logf("%-9s %-6s %-95s %s", g.dialect, status, truncateSQL(sql), desc)
		}
	}
}

func describe(stmt sqlstmt.Stmt) string {
	switch v := stmt.(type) {
	case *sqlstmt.SelectStmt:
		names := make([]string, 0, len(v.From))
		for _, f := range v.From {
			names = append(names, f.Name)
		}
		parts := []string{"Select", "from=" + fmt.Sprint(len(v.From)) + "[" + strings.Join(names, "|") + "]", "joins=" + fmt.Sprint(len(v.Joins)), "unions=" + fmt.Sprint(len(v.Unions)), "items=" + fmt.Sprint(len(v.Items))}
		if v.Where != nil {
			parts = append(parts, "where")
		}
		if v.Limit != nil {
			parts = append(parts, fmt.Sprintf("limit=%d/%d", v.Limit.Offset, v.Limit.Count))
		}
		return strings.Join(parts, ",")
	case *sqlstmt.InsertStmt:
		return "Insert table=" + v.Table.Name
	case *sqlstmt.UpdateStmt:
		return fmt.Sprintf("Update tables=%d set=%d", len(v.Tables), len(v.Set))
	case *sqlstmt.DeleteStmt:
		return fmt.Sprintf("Delete tables=%d", len(v.Tables))
	case *sqlstmt.MergeStmt:
		return "Merge target=" + v.Target
	case *sqlstmt.DdlStmt:
		return "Ddl " + v.DdlKind + " target=" + v.Target
	case *sqlstmt.ShowStmt:
		return "Show"
	case *sqlstmt.OtherStmt:
		return fmt.Sprintf("Other readOnly=%v", v.ReadOnly)
	}
	return fmt.Sprintf("%T", stmt)
}

func truncateSQL(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 92 {
		return s[:92] + "…"
	}
	return s
}
