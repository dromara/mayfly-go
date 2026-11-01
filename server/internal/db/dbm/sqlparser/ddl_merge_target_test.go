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

// TestDdlAndMergeTarget DDL / MERGE 的目标表提取（7 方言一致）。
//
// 目的：让「指定表」类策略与表级访问控制能覆盖改结构与 upsert 语句——
// 此前 DdlStmt/MergeStmt 只有语句类型、没有表名，表级规则对它们完全失效。
// 取不到名字时必须返回空串（不猜），否则会把无关表误当成目标。
func TestDdlAndMergeTarget(t *testing.T) {
	parsers := map[string]sqlparser.SQLParser{
		"mysql":      new(mysql.MysqlParser),
		"pgsql":      new(pgsql.PgsqlParser),
		"oracle":     new(oracle.OracleParser),
		"dm":         new(dm.DmParser),
		"mssql":      new(mssql.MssqlParser),
		"sqlite":     new(sqlite.SqliteParser),
		"clickhouse": new(clickhouse.ClickhouseParser),
	}
	cases := []struct {
		sql  string
		want string
	}{
		{"CREATE TABLE IF NOT EXISTS s1.t1 (id int)", "t1"},
		{"ALTER TABLE `t2` ADD COLUMN c int", "t2"},
		{"DROP TABLE IF EXISTS t3", "t3"},
		{"CREATE INDEX idx_a ON t4 (a)", "t4"},
		{"DROP INDEX idx_b ON t5", "t5"},
		{"CREATE VIEW v1 AS SELECT 1", "v1"},
		// 非表对象：不能塞进 Target，否则「指定表」策略会误命中
		{"CREATE SCHEMA s2", ""},
		{"CREATE DATABASE db2", ""},
	}
	for name, parser := range parsers {
		for _, c := range cases {
			stmt, err := parser.Parse(c.sql)
			if err != nil {
				t.Errorf("%s: 解析 %q 出错: %v", name, c.sql, err)
				continue
			}
			ddl, ok := stmt.(*sqlstmt.DdlStmt)
			if !ok {
				t.Errorf("%s: %q 期望 DdlStmt，得到 %T", name, c.sql, stmt)
				continue
			}
			if ddl.Target != c.want {
				t.Errorf("%s: %q 目标表 = %q，期望 %q", name, c.sql, ddl.Target, c.want)
			}
		}

		// MERGE 目标表
		stmt, err := parser.Parse("MERGE INTO t6 USING s6 ON (t6.id = s6.id) WHEN MATCHED THEN UPDATE SET t6.a = 1")
		if err != nil {
			t.Errorf("%s: MERGE 解析出错: %v", name, err)
			continue
		}
		merge, ok := stmt.(*sqlstmt.MergeStmt)
		if !ok {
			t.Errorf("%s: MERGE 期望 MergeStmt，得到 %T", name, stmt)
			continue
		}
		if merge.Target != "t6" {
			t.Errorf("%s: MERGE 目标表 = %q，期望 t6", name, merge.Target)
		}
	}
}
