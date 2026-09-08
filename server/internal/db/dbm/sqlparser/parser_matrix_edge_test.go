package sqlparser

// 跨方言解析边界探测矩阵：DML复杂形态（INSERT...SELECT/ON DUPLICATE/UPDATE JOIN/
// 多表DELETE）、子查询作列、方言特有操作符（::/->>/@/||/(+)/ROWNUM）、
// 引号别名与关键字名引用。探测各方言解析的结构必须准确
import (
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// kindOf 各方言对SQL的StmtKind判定必须一致
func kindOf(t *testing.T, p sqlparserParser, sql string) sqlstmt.Kind {
	t.Helper()
	stmt, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("解析失败 %q: %v", sql, err)
	}
	return stmt.StmtKind()
}

// TestEdgeDmlKind DML复杂形态的Kind判定（各方言必须一致）
func TestEdgeDmlKind(t *testing.T) {
	// [SQL, 除oracle外全支持（oracle无UPDATE JOIN/ON DUPLICATE/多表DELETE）]
	cases := []struct {
		name string
		sql  string
		kind sqlstmt.Kind
		// skipOracle: oracle方言不支持该语法（语法层面合法但形态不同），跳过
		skipOracle bool
	}{
		{"INSERT多行含转义引号", "INSERT INTO t (a, b) VALUES (1, 'x'), (2, 'it''s')", sqlstmt.KindInsert, false},
		{"INSERT表达式值", "INSERT INTO t (a, b) VALUES (NOW(), CONCAT('x', 'y'))", sqlstmt.KindInsert, false},
		{"INSERT...SELECT", "INSERT INTO t (a, b) SELECT x, y FROM s WHERE z = 1", sqlstmt.KindInsert, false},
		{"UPDATE多列含表达式", "UPDATE t SET a = 1, b = 'x''y', c = c + 1 WHERE id = 2", sqlstmt.KindUpdate, false},
		{"DELETE WHERE IN", "DELETE FROM t WHERE id IN (1, 2, 3)", sqlstmt.KindDelete, false},
		{"UPDATE无WHERE", "UPDATE t SET a = 1", sqlstmt.KindUpdate, false},
		{"INSERT...ON DUPLICATE KEY UPDATE", "INSERT INTO t (a) VALUES (1) ON DUPLICATE KEY UPDATE b = 2", sqlstmt.KindInsert, true},
		{"INSERT...ON CONFLICT DO NOTHING", "INSERT INTO t (a) VALUES (1) ON CONFLICT (a) DO NOTHING", sqlstmt.KindInsert, true},
		{"UPDATE带别名JOIN", "UPDATE t1 a JOIN t2 b ON a.id = b.id SET a.x = 1 WHERE b.y = 2", sqlstmt.KindUpdate, true},
		{"DELETE多表JOIN", "DELETE a FROM t1 a JOIN t2 b ON a.id = b.id WHERE b.x = 1", sqlstmt.KindDelete, true},
	}
	for _, c := range cases {
		for name, p := range allParsers() {
			if c.skipOracle && name == "oracle" {
				continue
			}
			if got := kindOf(t, p, c.sql); got != c.kind {
				t.Errorf("[%s] %s: kind=%s, 期望%s", name, c.name, got, c.kind)
			}
		}
	}
}

// TestEdgeSelectItems 子查询作列/CASE嵌套/负数等items结构探测
func TestEdgeSelectItems(t *testing.T) {
	cases := []struct {
		name  string
		sql   string
		items int
		from  int
		where bool
	}{
		{"子查询作列", "SELECT (SELECT MAX(id) FROM u) AS mx, id FROM t", 2, 1, false},
		{"CASE内嵌子查询", "SELECT CASE WHEN a THEN (SELECT x FROM u WHERE u.id = t.id) ELSE 0 END AS v FROM t", 1, 1, false},
		{"NOT EXISTS", "SELECT id FROM t WHERE NOT EXISTS (SELECT 1 FROM u)", 1, 1, true},
		{"NOT IN与IS NOT NULL", "SELECT id FROM t WHERE a NOT IN (1, 2) AND b IS NOT NULL", 1, 1, true},
		{"负数与一元运算", "SELECT -1 AS v, id FROM t WHERE a > -5", 2, 1, true},
		{"GROUP BY位置引用", "SELECT dept, COUNT(*) FROM t GROUP BY 1 ORDER BY 2", 2, 1, false},
		{"IN子查询双层括号", "SELECT id FROM t WHERE id IN ((SELECT id FROM u))", 1, 1, true},
		{"字符串含括号逗号", "SELECT id FROM t WHERE note = 'contains (a, b) here'", 1, 1, true},
	}
	for _, c := range cases {
		for name, p := range allParsers() {
			stmt, err := p.Parse(c.sql)
			if err != nil {
				t.Errorf("[%s] %s: 解析失败: %v", name, c.name, err)
				continue
			}
			sel, ok := stmt.(*sqlstmt.SelectStmt)
			if !ok {
				t.Errorf("[%s] %s: 期望SelectStmt，得到%T(%s)", name, c.name, stmt, stmt.StmtKind())
				continue
			}
			if len(sel.Items) != c.items {
				t.Errorf("[%s] %s: items=%d, 期望%d", name, c.name, len(sel.Items), c.items)
			}
			if len(sel.From) != c.from {
				t.Errorf("[%s] %s: from=%d, 期望%d", name, c.name, len(sel.From), c.from)
			}
			if (sel.Where != nil) != c.where {
				t.Errorf("[%s] %s: where=%v, 期望%v", name, c.name, sel.Where != nil, c.where)
			}
		}
	}
}

// TestEdgeQuotedKeywordIdentifiers 引号内的关键字名与引号别名（列名/表名恰为关键字形状）
func TestEdgeQuotedKeywordIdentifiers(t *testing.T) {
	cases := []struct {
		name    string
		dialect string
		sql     string
		col     string
	}{
		{"mysql反引号关键字列名", "mysql", "SELECT `left` FROM t", "left"},
		{"mysql反引号关键字表名", "mysql", "SELECT id FROM `order`", ""},
		{"pgsql双引号关键字列名", "pgsql", `SELECT "left" FROM t`, "left"},
		{"pgsql双引号关键字表名", "pgsql", `SELECT id FROM "order"`, ""},
		{"oracle双引号关键字列名", "oracle", `SELECT "left" FROM t`, "left"},
		{"dm双引号关键字列名", "dm", `SELECT "left" FROM t`, "left"},
	}
	for _, c := range cases {
		stmt, err := allParsers()[c.dialect].Parse(c.sql)
		if err != nil {
			t.Errorf("[%s] %s: 解析失败: %v", c.dialect, c.name, err)
			continue
		}
		if c.col == "" {
			continue
		}
		sel, ok := stmt.(*sqlstmt.SelectStmt)
		if !ok {
			t.Errorf("[%s] %s: 期望SelectStmt，得到%T", c.dialect, c.name, stmt)
			continue
		}
		if len(sel.Items) != 1 || sel.Items[0].ColumnName != c.col {
			t.Errorf("[%s] %s: items=%+v, 期望列名%q", c.dialect, c.name, sel.Items, c.col)
		}
	}
}

// TestEdgeDialectOperators 方言特有操作符（pg ::/->>、oracle (+)与ROWNUM、mysql @变量、||拼接）
func TestEdgeDialectOperators(t *testing.T) {
	cases := []struct {
		name    string
		dialect string
		sql     string
		items   int
		from    int
	}{
		{"pg类型转换::", "pgsql", "SELECT id::text FROM t", 1, 1},
		{"pg嵌套::链", "pgsql", "SELECT id::text::varchar(10) AS v, n FROM t", 2, 1},
		{"pg json操作符->>", "pgsql", `SELECT data->>'name' FROM t`, 1, 1},
		{"pg json操作符->", "pgsql", "SELECT data->'tags'->>0 AS first_tag FROM t", 1, 1},
		{"oracle(+)外连接", "oracle", "SELECT a.id FROM ta a, tb b WHERE a.id = b.id(+)", 1, 2},
		{"oracle ROWNUM", "oracle", "SELECT id FROM t WHERE ROWNUM <= 10", 1, 1},
		{"oracle || 拼接", "oracle", "SELECT a || b AS x FROM t", 1, 1},
		{"dm || 拼接", "dm", "SELECT a || b AS x FROM t", 1, 1},
		{"mysql @变量", "mysql", "SELECT @v, id FROM t", 2, 1},
		{"mysql !=", "mysql", "SELECT id FROM t WHERE a != 1", 1, 1},
		{"pgsql != 与 <>", "pgsql", "SELECT id FROM t WHERE a <> 1", 1, 1},
		{"oracle NVL多参", "oracle", "SELECT NVL(a, b, c) AS v FROM t", 1, 1},
	}
	for _, c := range cases {
		stmt, err := allParsers()[c.dialect].Parse(c.sql)
		if err != nil {
			t.Errorf("[%s] %s: 解析失败: %v", c.dialect, c.name, err)
			continue
		}
		sel, ok := stmt.(*sqlstmt.SelectStmt)
		if !ok {
			t.Errorf("[%s] %s: 期望SelectStmt，得到%T", c.dialect, c.name, stmt)
			continue
		}
		if len(sel.Items) != c.items {
			t.Errorf("[%s] %s: items=%d, 期望%d (%+v)", c.dialect, c.name, len(sel.Items), c.items, sel.Items)
		}
		if len(sel.From) != c.from {
			t.Errorf("[%s] %s: from=%d, 期望%d", c.dialect, c.name, len(sel.From), c.from)
		}
	}
}

// TestEdgeCteInDml CTE与DML组合（pg: WITH...INSERT/UPDATE/DELETE，dm兼容）
func TestEdgeCteInDml(t *testing.T) {
	for name, p := range allParsers() {
		if name == "mysql" {
			continue // mysql不支持CTE写DML
		}
		if _, err := p.Parse("WITH moved AS (DELETE FROM t WHERE a = 1 RETURNING id) INSERT INTO log SELECT id FROM moved"); err != nil {
			t.Errorf("[%s] WITH...INSERT解析失败: %v", name, err)
		}
	}
}

// TestEdgeKeywordAliasPositions 关键字形状的显式别名（AS后跟关键字形状标识符）不得破坏解析
func TestEdgeKeywordAliasPositions(t *testing.T) {
	// AS "引号别名" 各方言都支持；AS后裸关键字形别名（order/left）在部分方言非法，
	// 仅要求不崩（返回任意Stmt或error，不得panic/死循环）
	for _, sql := range []string{
		`SELECT id AS "order" FROM t`,
		`SELECT id AS "user name" FROM t`,
		"SELECT id AS cnt2, name AS full2 FROM t",
	} {
		for name, p := range allParsers() {
			stmt, err := p.Parse(sql)
			if err != nil {
				t.Errorf("[%s] %q 解析失败: %v", name, sql, err)
				continue
			}
			if stmt == nil {
				t.Errorf("[%s] %q 解析结果为nil", name, sql)
			}
		}
	}
}

// TestEdgeRound2 第二轮高难度探测：真实分页包装SQL/括号UNION/聚合DISTINCT/关键字形列名等
func TestEdgeRound2(t *testing.T) {
	cases := []struct {
		name    string
		dialect string // 空=全方言
		sql     string
		items   int
		from    int
		where   bool
	}{
		// oracle经典三段分页包装（分页功能真实生成的SQL形态，必须可完整解析）
		{"oracle三段分页包装", "oracle",
			"SELECT * FROM (SELECT t_.*, ROWNUM rn FROM (SELECT id, name FROM t ORDER BY id) t_ WHERE ROWNUM <= 10) WHERE rn > 0",
			1, 1, true},
		{"嵌套派生表两层", "", "SELECT a.id FROM (SELECT id FROM (SELECT id FROM t WHERE x = 1) m WHERE m.id > 2) a", 1, 1, false},
		{"聚合DISTINCT", "", "SELECT COUNT(DISTINCT dept) AS dc, SUM(salary) FROM t", 2, 1, false},
		{"SELECT DISTINCT多列", "", "SELECT DISTINCT a, b FROM t", 2, 1, false},
		{"CASE内IS NULL", "", "SELECT CASE WHEN a IS NULL THEN 1 ELSE 2 END AS v FROM t", 1, 1, false},
		{"UPDATE SET子查询", "", "UPDATE t SET a = (SELECT MAX(x) FROM u) WHERE id = 1", 0, 0, false},
		{"DELETE IN子查询", "", "DELETE FROM t WHERE id IN (SELECT id FROM u)", 0, 0, false},
		{"VALUES内CASE", "", "INSERT INTO t (a) VALUES (CASE WHEN 1 THEN 2 ELSE 3 END)", 0, 0, false},
		{"LIMIT逗号分页(mysql)", "mysql", "SELECT id FROM t LIMIT 5, 10", 1, 1, false},
		{"LIMIT ALL(pg)", "pgsql", "SELECT id FROM t LIMIT ALL OFFSET 5", 1, 1, false},
		{"LIKE ESCAPE", "", "SELECT id FROM t WHERE name LIKE 'a!%' ESCAPE '!'", 1, 1, true},
		{"OR链与括号分组", "", "SELECT id FROM t WHERE (a = 1 OR b = 2) AND (c = 3 OR d LIKE 'x%')", 1, 1, true},
	}
	for _, c := range cases {
		dialects := []string{"mysql", "pgsql", "dm", "oracle"}
		if c.dialect != "" {
			dialects = []string{c.dialect}
		}
		for _, d := range dialects {
			stmt, err := allParsers()[d].Parse(c.sql)
			if err != nil {
				t.Errorf("[%s] %s: 解析失败: %v", d, c.name, err)
				continue
			}
			if c.items == 0 {
				continue // DML用例仅要求不崩
			}
			sel, ok := stmt.(*sqlstmt.SelectStmt)
			if !ok {
				t.Errorf("[%s] %s: 期望SelectStmt，得到%T(%s)", d, c.name, stmt, stmt.StmtKind())
				continue
			}
			if len(sel.Items) != c.items {
				t.Errorf("[%s] %s: items=%d, 期望%d (%+v)", d, c.name, len(sel.Items), c.items, sel.Items)
			}
			if len(sel.From) != c.from {
				t.Errorf("[%s] %s: from=%d, 期望%d", d, c.name, len(sel.From), c.from)
			}
			if (sel.Where != nil) != c.where {
				t.Errorf("[%s] %s: where=%v, 期望%v", d, c.name, sel.Where != nil, c.where)
			}
		}
	}
}

// TestEdgeKeywordLikeColumns 关键字形裸列名在非保留字方言（oracle/dm）的WHERE/ORDER BY中
// 不得截断表达式（IsExprEnd的JOIN前缀关键字边界误伤探测）
func TestEdgeKeywordLikeColumns(t *testing.T) {
	// oracle/dm中 left/right/full 非保留字，可作裸列名
	for _, d := range []string{"oracle", "dm"} {
		stmt, err := allParsers()[d].Parse("SELECT id FROM t WHERE left = 1 AND right = 2")
		if err != nil {
			t.Errorf("[%s] WHERE关键字形列名解析失败: %v", d, err)
			continue
		}
		sel, ok := stmt.(*sqlstmt.SelectStmt)
		if !ok || sel.Where == nil {
			t.Errorf("[%s] WHERE关键字形列名: where应被识别: %T", d, stmt)
			continue
		}
		// 表达式完整性：LEFT/RIGHT边界误伤会截断AND右半
		if wt := sel.Where.Text; !strings.Contains(wt, "right = 2") {
			t.Errorf("[%s] WHERE表达式被截断: %q", d, wt)
		}
		stmt2, err := allParsers()[d].Parse("SELECT id FROM t ORDER BY left, id")
		if err != nil {
			t.Errorf("[%s] ORDER BY关键字形列名解析失败: %v", d, err)
			continue
		}
		sel2 := stmt2.(*sqlstmt.SelectStmt)
		if len(sel2.OrderBy) < 2 {
			t.Errorf("[%s] ORDER BY项数=%d, 期望2（left不得被当边界截断）", d, len(sel2.OrderBy))
		}
	}
}

// TestEdgeParenUnion 括号包裹的UNION段与UNION后ORDER BY
func TestEdgeParenUnion(t *testing.T) {
	for name, p := range allParsers() {
		stmt, err := p.Parse("(SELECT a FROM t) UNION (SELECT b FROM u) ORDER BY 1")
		if err != nil {
			t.Errorf("[%s] 括号UNION解析失败: %v", name, err)
			continue
		}
		sel, ok := stmt.(*sqlstmt.SelectStmt)
		if !ok {
			t.Errorf("[%s] 期望SelectStmt，得到%T", name, stmt)
			continue
		}
		if len(sel.Unions) != 1 {
			t.Errorf("[%s] 括号UNION应识别1段: %+v", name, sel.Unions)
		}
	}
}

// TestEdgeOnFollowedByJoin ON条件后紧跟JOIN子句：ON表达式不得吞掉后续JOIN
// （IsExprEnd/SkipCondExpr拆分后的回归保护）
func TestEdgeOnFollowedByJoin(t *testing.T) {
	for name, p := range allParsers() {
		stmt, err := p.Parse("SELECT a.id FROM ta a JOIN tb b ON a.id = b.id LEFT JOIN tc c ON b.x = c.x RIGHT JOIN td d ON c.y = d.y WHERE a.z = 1")
		if err != nil {
			t.Errorf("[%s] ON后接JOIN解析失败: %v", name, err)
			continue
		}
		sel, ok := stmt.(*sqlstmt.SelectStmt)
		if !ok {
			t.Errorf("[%s] 期望SelectStmt，得到%T", name, stmt)
			continue
		}
		if len(sel.Joins) != 3 {
			t.Errorf("[%s] joins=%d, 期望3（ON表达式不得吞掉LEFT/RIGHT JOIN）(%+v)", name, len(sel.Joins), sel.Joins)
		}
		// Join.Text 不得越界包含下一JOIN的前缀词（LEFT/RIGHT）
		for ji, j := range sel.Joins {
			if strings.HasSuffix(strings.ToUpper(j.Text), " LEFT") || strings.HasSuffix(strings.ToUpper(j.Text), " RIGHT") {
				t.Errorf("[%s] Join[%d].Text越界: %q", name, ji, j.Text)
			}
		}
		if sel.Where == nil {
			t.Errorf("[%s] WHERE应被识别", name)
		}
	}
}

// TestEdgeRound3 第三轮探测：迁移链路真实生成的SQL形态（MERGE upsert/RETURNING/REPLACE）
// 与极端形态（深嵌套/长SQL），要求不panic不死循环、Kind判定正确
func TestEdgeRound3(t *testing.T) {
	// Kind判定用例：[方言, SQL, 期望Kind（空串=仅要求不崩不panic）]
	cases := []struct {
		dialect string
		sql     string
		kind    sqlstmt.Kind
	}{
		{"mysql", "REPLACE INTO t (a, b) VALUES (1, 2)", sqlstmt.KindInsert},
		{"pgsql", "DELETE FROM t WHERE id = 1 RETURNING *", sqlstmt.KindDelete},
		{"pgsql", "INSERT INTO t (a) VALUES (1) RETURNING id", sqlstmt.KindInsert},
		{"oracle", "MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.a = s.a WHEN NOT MATCHED THEN INSERT (a) VALUES (s.a)", ""},
		{"dm", "MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.a = s.a", ""},
		{"mysql", "UPDATE t1 a, t2 b SET a.x = b.x WHERE a.id = b.id", sqlstmt.KindUpdate},
		{"mysql", "INSERT INTO t (`left`, `order`) VALUES (1, 2)", sqlstmt.KindInsert},
	}
	for _, c := range cases {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("[%s] panic: %v", c.dialect, r)
				}
			}()
			stmt, err := allParsers()[c.dialect].Parse(c.sql)
			if err != nil {
				t.Errorf("[%s] 解析失败: %v", c.dialect, err)
				return
			}
			if c.kind != "" && stmt.StmtKind() != c.kind {
				t.Errorf("[%s] kind=%s, 期望%s", c.dialect, stmt.StmtKind(), c.kind)
			}
		}()
	}

	// 极端形态：100层嵌套括号与长SQL（防死循环/栈溢出）
	deep := "SELECT id FROM t WHERE a IN (" + strings.Repeat("(", 100) + "1" + strings.Repeat(")", 100) + ")"
	long := "SELECT " + strings.Join(strings.Split(strings.Repeat("c, ", 500), ", "), ", ") + " FROM t WHERE x = 'long'"
	for name, p := range allParsers() {
		for _, sql := range []string{deep, long} {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("[%s] 极端SQL panic: %v", name, r)
					}
				}()
				if _, err := p.Parse(sql); err != nil {
					t.Errorf("[%s] 极端SQL解析失败(len=%d): %v", name, len(sql), err)
				}
			}()
		}
	}
}
