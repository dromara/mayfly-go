package sqlparser_test

import (
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/clickhouse"
	"mayfly-go/internal/db/dbm/sqlparser/mssql"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// TestTableHintsAndApply 表清单不得被「表提示 / APPLY / ARRAY JOIN」污染或漏掉。
//
// 表清单一错，表级访问控制与「指定表」策略就会在错误的表上命中或漏判：
//   - MySQL 索引提示 `USE INDEX (idx)` 里的 INDEX 曾被当成 FROM 的第二张表；
//   - T-SQL CROSS/OUTER APPLY 的内层数据源曾被整体丢弃（表清单少表）；
//   - ClickHouse ARRAY JOIN 的数组列名曾被当成一张表。
func TestTableHintsAndApply(t *testing.T) {
	// MySQL 索引提示：表清单只应有 a、b
	my := new(mysql.MysqlParser)
	stmt, err := my.Parse("SELECT STRAIGHT_JOIN a.* FROM a USE INDEX (idx_a) JOIN b ON b.id = a.id LIMIT 5")
	if err != nil {
		t.Fatalf("mysql parse err: %v", err)
	}
	sel := stmt.(*sqlstmt.SelectStmt)
	names := tableNames(sel)
	if len(names) != 1 || names[0] != "a" {
		t.Fatalf("mysql FROM 被提示污染: %v", names)
	}
	if len(sel.Joins) != 1 {
		t.Fatalf("mysql JOIN 数量错误: %d", len(sel.Joins))
	}

	// T-SQL CROSS APPLY：必须产生一个 join（右侧数据源不能丢）
	ms := new(mssql.MssqlParser)
	stmt, err = ms.Parse("SELECT * FROM t1 CROSS APPLY (SELECT TOP 1 * FROM t2 WHERE t2.id = t1.id) x")
	if err != nil {
		t.Fatalf("mssql parse err: %v", err)
	}
	sel = stmt.(*sqlstmt.SelectStmt)
	if len(sel.Joins) != 1 {
		t.Fatalf("CROSS APPLY 未建模为 join: joins=%d", len(sel.Joins))
	}

	// ClickHouse：FINAL/PREWHERE/ARRAY JOIN 都不得进入表清单
	ch := new(clickhouse.ClickhouseParser)
	stmt, err = ch.Parse("SELECT * FROM t FINAL PREWHERE a > 0 WHERE b = 1 ARRAY JOIN arr AS x LIMIT 10")
	if err != nil {
		t.Fatalf("clickhouse parse err: %v", err)
	}
	sel = stmt.(*sqlstmt.SelectStmt)
	names = tableNames(sel)
	if len(names) != 1 || names[0] != "t" {
		t.Fatalf("clickhouse 表清单被污染: %v", names)
	}
	if sel.Where == nil || !strings.Contains(sel.Where.Text, "a > 0") {
		t.Fatalf("PREWHERE 条件未并入 Where: %+v", sel.Where)
	}
}

func tableNames(sel *sqlstmt.SelectStmt) []string {
	names := make([]string, 0, len(sel.From))
	for _, f := range sel.From {
		names = append(names, f.Name)
	}
	return names
}
