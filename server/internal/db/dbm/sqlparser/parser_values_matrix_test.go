package sqlparser

// 跨方言 INSERT VALUES 字面量提取矩阵：INSERT 回滚靠这份字面量原文定位并删除刚插入的行，
// 任何一格提取失真（丢引号、改转义、把函数当字面量）都会让回滚 DELETE 命中错误的行或 0 行，
// 而语句照样「执行成功」。既有矩阵只断言列数/语句类型，这里逐格钉住字面量原文与「整体放弃」边界。
//
// 覆盖全部方言：mysql/pgsql/oracle/dm 各有独立 parser，mssql 包装 pgsql、sqlite 直接用 pgsql
// （见各自 dialect.go 的 GetSQLParser），故 pgsql 一例即代表 mssql 与 sqlite。

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/mssql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// valueParsers 在 allParsers（mysql/pgsql/dm/oracle）基础上补 mssql，凑齐全部方言的 INSERT 解析路径
func valueParsers() map[string]sqlparserParser {
	parsers := allParsers()
	parsers["mssql"] = mssql.NewParser()
	return parsers
}

func mustParseInsert(t *testing.T, p sqlparserParser, sql string) *sqlstmt.InsertStmt {
	t.Helper()
	stmt, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("parse %q failed: %v", sql, err)
	}
	ins, ok := stmt.(*sqlstmt.InsertStmt)
	if !ok {
		t.Fatalf("parse %q: got %T, want *sqlstmt.InsertStmt", sql, stmt)
	}
	return ins
}

// TestUniversalInsertValueRowsLiterals 各方言 parser 必须把 VALUES 里的纯字面量逐格提取成原文：
// 字符串含外层引号与原样转义、数字保形状、NULL 置 Null 标记。这是 INSERT 回滚 DELETE 条件的唯一来源
func TestUniversalInsertValueRowsLiterals(t *testing.T) {
	const nullCell = "\x00NULL" // 期望 Null=true 的哨兵（不与任何真实字面量原文相撞）
	cases := []struct {
		name string
		sql  string
		want [][]string
	}{
		{"整数与字符串", "INSERT INTO t (id, name) VALUES (1, 'tom')", [][]string{{"1", "'tom'"}}},
		{"多行", "INSERT INTO t (id, name) VALUES (1, 'a'), (2, 'b')", [][]string{{"1", "'a'"}, {"2", "'b'"}}},
		{"转义单引号原文保留", "INSERT INTO t (a) VALUES ('it''s')", [][]string{{"'it''s'"}}},
		{"NULL 置标记", "INSERT INTO t (a) VALUES (NULL)", [][]string{{nullCell}}},
		{"负数", "INSERT INTO t (a) VALUES (-5)", [][]string{{"-5"}}},
		{"小数", "INSERT INTO t (a) VALUES (1.5)", [][]string{{"1.5"}}},
		{"多列混合", "INSERT INTO t (a, b, c) VALUES (1, 'x', NULL)", [][]string{{"1", "'x'", nullCell}}},
	}
	for name, p := range valueParsers() {
		name, p := name, p
		t.Run(name, func(t *testing.T) {
			for _, tc := range cases {
				ins := mustParseInsert(t, p, tc.sql)
				if len(ins.ValueRows) != len(tc.want) {
					t.Fatalf("[%s] %q: got %d value rows, want %d (%+v)", name, tc.sql, len(ins.ValueRows), len(tc.want), ins.ValueRows)
				}
				for r, wantRow := range tc.want {
					gotRow := ins.ValueRows[r]
					if len(gotRow) != len(wantRow) {
						t.Fatalf("[%s] %q row %d: got %d cells, want %d (%+v)", name, tc.sql, r, len(gotRow), len(wantRow), gotRow)
					}
					for c, wantText := range wantRow {
						cell := gotRow[c]
						if wantText == nullCell {
							if !cell.Null {
								t.Errorf("[%s] %q cell(%d,%d): want Null=true, got Text=%q", name, tc.sql, r, c, cell.Text)
							}
							continue
						}
						if cell.Null {
							t.Errorf("[%s] %q cell(%d,%d): want Text=%q, got Null=true", name, tc.sql, r, c, wantText)
							continue
						}
						if cell.Text != wantText {
							t.Errorf("[%s] %q cell(%d,%d): got Text=%q, want %q", name, tc.sql, r, c, cell.Text, wantText)
						}
					}
				}
			}
		})
	}
}

// TestUniversalInsertValueRowsRejectsNonLiterals 任何一格不是可确定求值的纯字面量时，整个 VALUES
// 必须放弃（ValueRows=nil）：函数/表达式/子查询/占位符的最终写入值只有执行后才知道，猜一个近似值
// 写进回滚 DELETE 的定位条件，轻则命中 0 行、重则删错行——宁可不出快照，也不能出错快照
func TestUniversalInsertValueRowsRejectsNonLiterals(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"函数", "INSERT INTO t (a) VALUES (NOW())"},
		{"算术表达式", "INSERT INTO t (a) VALUES (1 + 1)"},
		{"子查询", "INSERT INTO t (a) VALUES ((SELECT MAX(x) FROM u))"},
		{"字面量与函数混排", "INSERT INTO t (a, b) VALUES (1, NOW())"},
		{"非法数字形状", "INSERT INTO t (a) VALUES (1.2.3)"},
	}
	for name, p := range valueParsers() {
		name, p := name, p
		t.Run(name, func(t *testing.T) {
			for _, tc := range cases {
				ins := mustParseInsert(t, p, tc.sql)
				if ins.ValueRows != nil {
					t.Errorf("[%s] %q: 非纯字面量必须整体放弃(ValueRows=nil)，却得到 %+v", name, tc.sql, ins.ValueRows)
				}
			}
		})
	}
}

// TestUniversalUpdateSetColumnNames 各方言 parser 必须提取 UPDATE 的 SET 裸列名（剥掉引号）：
// 旧值采集回查与回滚生成都按这份列名定位，列名错位会让旧值采空或把旧值写回错误的列
func TestUniversalUpdateSetColumnNames(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want []string
	}{
		{"多列赋值", "UPDATE t SET name = 'x', age = 30 WHERE id = 1", []string{"name", "age"}},
		{"单列赋值", "UPDATE t SET val = 1 WHERE id = 1", []string{"val"}},
		{"赋值为表达式仍取列名", "UPDATE t SET a = a + 1, b = UPPER('x') WHERE id = 1", []string{"a", "b"}},
	}
	for name, p := range valueParsers() {
		name, p := name, p
		t.Run(name, func(t *testing.T) {
			for _, tc := range cases {
				stmt, err := p.Parse(tc.sql)
				if err != nil {
					t.Fatalf("[%s] parse %q: %v", name, tc.sql, err)
				}
				upd, ok := stmt.(*sqlstmt.UpdateStmt)
				if !ok {
					t.Fatalf("[%s] parse %q: got %T, want *sqlstmt.UpdateStmt", name, tc.sql, stmt)
				}
				if len(upd.Set) != len(tc.want) {
					t.Fatalf("[%s] %q: got %d assignments, want %d", name, tc.sql, len(upd.Set), len(tc.want))
				}
				for i, wantCol := range tc.want {
					if got := upd.Set[i].Column; got != wantCol {
						t.Errorf("[%s] %q set[%d].Column = %q, want %q", name, tc.sql, i, got, wantCol)
					}
				}
			}
		})
	}
}
