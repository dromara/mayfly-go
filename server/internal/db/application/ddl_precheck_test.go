package application

// DDL 预检规则：每条风险都必须能被单独触发，且「不可逆」与「耗时」分属不同等级。
// 这里只测纯函数——读表规模那段是一条 SQL，动它就得连库，不适合放在单测里。

import (
	"testing"
)

func precheckOne(sql string, stats map[string]*DdlTableStat) []DdlFinding {
	kind, tables := ddlKindOf(sql)
	return ddlPrecheck([]DdlStatement{{SQL: sql, Kind: kind, Tables: tables}}, stats)
}

func hasFinding(findings []DdlFinding, code string) (DdlFinding, bool) {
	for _, finding := range findings {
		if finding.Code == code {
			return finding, true
		}
	}
	return DdlFinding{}, false
}

func TestDdlKindOfExtractsTable(t *testing.T) {
	cases := []struct {
		sql   string
		kind  string
		table string
	}{
		{"ALTER TABLE `mayfly_go`.`t_a` ADD COLUMN c int", "alter_table", "t_a"},
		{"alter table if exists t_b drop column x", "alter_table", "t_b"},
		{"DROP TABLE IF EXISTS t_c", "drop_table", "t_c"},
		{"TRUNCATE TABLE t_d", "truncate_table", "t_d"},
		{"truncate t_e", "truncate_table", "t_e"},
		{"CREATE TABLE t_f (id int)", "create_table", "t_f"},
		{"CREATE UNIQUE INDEX idx_a ON t_g (a)", "create_index", "t_g"},
		{"RENAME TABLE t_h TO t_i", "rename_table", "t_h"},
		{"SELECT * FROM t_j", "", ""},
	}
	for _, c := range cases {
		kind, tables := ddlKindOf(c.sql)
		if kind != c.kind {
			t.Fatalf("%q kind = %q, want %q", c.sql, kind, c.kind)
		}
		if c.table != "" && (len(tables) == 0 || tables[0] != c.table) {
			t.Fatalf("%q tables = %v, want %q", c.sql, tables, c.table)
		}
	}
}

func TestDdlPrecheckDangerRules(t *testing.T) {
	stats := map[string]*DdlTableStat{"t_a": {Name: "t_a", Rows: 10, HasPk: true}}

	findings := precheckOne("DROP TABLE t_a", stats)
	finding, ok := hasFinding(findings, "drop_table")
	if !ok || finding.Level != DdlRiskDanger {
		t.Fatalf("drop table must be danger, got %+v", findings)
	}

	findings = precheckOne("TRUNCATE TABLE t_a", stats)
	if finding, ok = hasFinding(findings, "truncate_table"); !ok || finding.Level != DdlRiskDanger {
		t.Fatalf("truncate must be danger, got %+v", findings)
	}

	findings = precheckOne("ALTER TABLE t_a DROP COLUMN c", stats)
	if finding, ok = hasFinding(findings, "drop_column"); !ok || finding.Level != DdlRiskDanger {
		t.Fatalf("drop column must be danger, got %+v", findings)
	}
}

func TestDdlPrecheckWarnRules(t *testing.T) {
	// 大表 + 无主键 + 改类型 + 加索引：四条 warn 都要出，且不能升级成 danger
	stats := map[string]*DdlTableStat{"t_big": {Name: "t_big", Rows: ddlLargeTableRows, HasPk: false}}
	findings := precheckOne("ALTER TABLE t_big MODIFY COLUMN c varchar(200), ADD INDEX idx_c (c)", stats)

	for _, code := range []string{"large_table", "no_primary_key", "type_change", "add_index"} {
		finding, ok := hasFinding(findings, code)
		if !ok {
			t.Fatalf("missing finding %q, got %+v", code, findings)
		}
		if finding.Level != DdlRiskWarn {
			t.Fatalf("%q level = %q, want warn", code, finding.Level)
		}
	}
	// 大表发现要带上行数，界面才能显示「你说它大，多大」
	large, _ := hasFinding(findings, "large_table")
	if large.Args == nil || large.Args["rows"] != ddlLargeTableRows {
		t.Fatalf("large_table must carry rows, got %+v", large.Args)
	}
}

func TestDdlPrecheckSmallTableHasNoScaleNoise(t *testing.T) {
	// 小表 + 有主键 + 只是加列：不该有任何 warn/danger，避免规则噪声让人忽略真风险
	stats := map[string]*DdlTableStat{"t_small": {Name: "t_small", Rows: 3, HasPk: true}}
	findings := precheckOne("ALTER TABLE t_small ADD COLUMN c int", stats)
	for _, finding := range findings {
		if finding.Level != DdlRiskInfo {
			t.Fatalf("small safe change must not warn, got %+v", findings)
		}
	}
}

func TestDdlPrecheckUnknownTableIsInfo(t *testing.T) {
	// 表读不到（名字写错或还没建）：给 info 提示，但不谎报规模类风险
	findings := precheckOne("ALTER TABLE t_missing ADD COLUMN c int", map[string]*DdlTableStat{})
	finding, ok := hasFinding(findings, "table_unknown")
	if !ok || finding.Level != DdlRiskInfo {
		t.Fatalf("unknown table must be info, got %+v", findings)
	}
	if _, ok := hasFinding(findings, "large_table"); ok {
		t.Fatal("unknown table must not report large_table")
	}
	// 建表语句的目标表本来就不存在，不该被当成「名字写错」
	if findings = precheckOne("CREATE TABLE t_new (id int)", map[string]*DdlTableStat{}); len(findings) != 0 {
		t.Fatalf("create table must not report unknown table, got %+v", findings)
	}
}

func TestSplitTopLevelSql(t *testing.T) {
	cases := []struct {
		name   string
		script string
		want   int
	}{
		{"两条语句", "ALTER TABLE t_a ADD c int; DROP TABLE t_b;", 2},
		{"引号内的分号不切", "UPDATE t SET c='a;b'; ALTER TABLE t_a ADD c int", 2},
		{"反引号内的分号不切", "ALTER TABLE `t;a` ADD c int; SELECT 1", 2},
		{"行注释", "-- a; b\nALTER TABLE t_a ADD c int", 1},
		{"块注释", "/* a; b */ ALTER TABLE t_a ADD c int", 1},
		{"末尾无分号", "ALTER TABLE t_a ADD c int", 1},
		{"空脚本", "   \n\t ", 0},
		{"未闭合引号保守整段", "ALTER TABLE t SET c='x; DROP TABLE t_b", 1},
	}
	for _, c := range cases {
		if got := splitTopLevelSql(c.script); len(got) != c.want {
			t.Fatalf("%s: got %d statements (%v), want %d", c.name, len(got), got, c.want)
		}
	}
	// 切分结果里引号拼写必须原样保留（切分不得改动语句内容）
	got := splitTopLevelSql("UPDATE t SET c='a;b';")
	if len(got) != 1 || got[0] != "UPDATE t SET c='a;b'" {
		t.Fatalf("statement text must be preserved, got %v", got)
	}
}
