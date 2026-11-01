package sqlparser_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/mssql"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/internal/db/dbm/sqlparser/oracle"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// TestShapeReport 生成「解析行为看板」：每条代表性 SQL 在关键方言下的
// 类型 / 表数 / join 数 / limit / where 有无，落到 testdata/snapshot/shape_report.md。
//
// 快照 golden 是逐字段文本 diff，评审时不易读；本看板给出一眼可扫的结构摘要，
// 用于「改了解析器到底影响了哪些语句形态」这类人工评审。测试每次重写该文件，
// 因此它永远是当前行为的事实记录（不参与失败判定，避免文档型噪音）。
func TestShapeReport(t *testing.T) {
	corpus := []string{
		"SELECT a, b FROM t1 JOIN t2 ON t1.id = t2.id WHERE t1.x > 0 ORDER BY a LIMIT 10",
		"SELECT * FROM (SELECT id FROM t) x WHERE id IN (SELECT id FROM s)",
		"WITH c AS (SELECT 1) SELECT * FROM c",
		"WITH d AS (DELETE FROM t RETURNING id) SELECT * FROM d",
		"SELECT COUNT(*) FROM t u GROUP BY u.k HAVING COUNT(*) > 1",
		"SELECT * FROM t1 UNION ALL SELECT * FROM t2",
		"INSERT INTO t (id, v) VALUES (1, 2)",
		"INSERT INTO t (id, v) SELECT id, v FROM s",
		"UPDATE t SET v = 1 WHERE id = 2",
		"UPDATE t1 JOIN t2 ON t1.id = t2.id SET t1.v = t2.v",
		"DELETE FROM t WHERE id = 1",
		"MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.v = s.v",
		"EXPLAIN SELECT * FROM t",
		"CREATE TABLE IF NOT EXISTS s1.t1 (id int, v varchar(20))",
		"ALTER TABLE t1 ADD COLUMN c int",
		"DROP TABLE IF EXISTS t1",
		"RENAME TABLE a TO b, c TO d",
		"TRUNCATE TABLE t1",
		"CALL p_proc(1)",
		"LOAD DATA INFILE '/tmp/a.csv' INTO TABLE t1",
	}
	parsers := []struct {
		name   string
		parser sqlparser.SQLParser
	}{
		{"mysql", new(mysql.MysqlParser)},
		{"pgsql", new(pgsql.PgsqlParser)},
		{"oracle", new(oracle.OracleParser)},
		{"mssql", new(mssql.MssqlParser)},
	}

	var b strings.Builder
	b.WriteString("# SQL 解析行为看板\n\n")
	b.WriteString("> 由 `TestShapeReport` 生成，反映当前解析行为。列含义：type=语句类型，tables=表清单数，joins=JOIN 数，limit=分页，where=有无条件。\n\n")
	b.WriteString("| SQL | 方言 | type | tables | joins | limit | where | cols |\n|---|---|---|---|---|---|---|---|\n")
	for _, sql := range corpus {
		display := sql
		if len(display) > 62 {
			display = display[:62] + "…"
		}
		display = strings.ReplaceAll(display, "|", "\\|")
		for _, p := range parsers {
			stmt, err := p.parser.Parse(sql)
			if err != nil {
				fmt.Fprintf(&b, "| `%s` | %s | PARSE_ERROR | - | - | - | - | - |\n", display, p.name)
				continue
			}
			typ := sqlparser.Classify(p.parser, sqlparser.NewSplitter(dialectConfig(p.name)), sql, stmt, err)
			tables, joins, limit, where, cols := shapeFields(stmt)
			fmt.Fprintf(&b, "| `%s` | %s | %s | %d | %d | %s | %v | %d |\n", display, p.name, typ, tables, joins, limit, where, cols)
		}
	}

	report := b.String()
	if strings.Contains(report, "PARSE_ERROR") {
		t.Errorf("看板出现解析失败:\n%s", report)
	}
	if err := os.MkdirAll("testdata/snapshot", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile("testdata/snapshot/shape_report.md", []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
}

func shapeFields(stmt sqlstmt.Stmt) (tables, joins int, limit string, where bool, cols int) {
	switch v := stmt.(type) {
	case *sqlstmt.SelectStmt:
		tables, joins = len(v.From), len(v.Joins)
		if v.Limit != nil {
			limit = fmt.Sprintf("%d/%d", v.Limit.Offset, v.Limit.Count)
		}
		where = v.Where != nil
	case *sqlstmt.UpdateStmt:
		tables = len(v.Tables)
		if v.Limit != nil {
			limit = fmt.Sprintf("%d/%d", v.Limit.Offset, v.Limit.Count)
		}
		where = v.Where != nil
	case *sqlstmt.DeleteStmt:
		tables = len(v.Tables)
		if v.Limit != nil {
			limit = fmt.Sprintf("%d/%d", v.Limit.Offset, v.Limit.Count)
		}
		where = v.Where != nil
	case *sqlstmt.InsertStmt:
		tables = 1
	case *sqlstmt.DdlStmt:
		cols = len(v.Columns)
	}
	return
}

// dialectConfig 取方言词法配置（切割器需要与解析器一致的注释/引号语义）。
func dialectConfig(dialect string) tokenizer.DialectConfig {
	switch dialect {
	case "pgsql":
		return tokenizer.PgConfig
	case "oracle":
		return tokenizer.OracleConfig
	case "mssql":
		return tokenizer.MssqlConfig
	default:
		return tokenizer.MysqlConfig
	}
}
