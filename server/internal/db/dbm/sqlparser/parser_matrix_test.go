package sqlparser

// 跨方言复杂SQL解析矩阵测试：4个方言解析器对同一SQL清单必须解析出一致的核心结构
import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/dm"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/internal/db/dbm/sqlparser/oracle"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

func allParsers() map[string]sqlparserParser {
	return map[string]sqlparserParser{
		"mysql":  new(mysql.MysqlParser),
		"pgsql":  new(pgsql.PgsqlParser),
		"dm":     new(dm.DmParser),
		"oracle": new(oracle.OracleParser),
	}
}

type sqlparserParser = interface {
	Parse(stmt string) (sqlstmt.Stmt, error)
}

// selectSpec 复杂SELECT的核心结构期望（各方言一致）
type selectSpec struct {
	items    int  // select项数（位置级脱敏依赖项数准确性）
	from     int  // FROM表数
	joins    int  // JOIN数
	where    bool // 是否有WHERE
	unions   int  // UNION段数
	distinct bool
	// itemAlias[i] 校验第i个select项的别名（空则跳过校验）
	itemAlias map[int]string
	starItem  bool // 是否存在 * 项
}

// TestUniversalComplexSelect 各方言必须解析出一致核心结构的复杂SELECT清单
func TestUniversalComplexSelect(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		spec selectSpec
	}{
		{
			name: "多表JOIN聚合完整链路",
			sql:  "SELECT DISTINCT d.name, COUNT(o.id) AS cnt FROM users u JOIN orders o ON o.uid = u.id LEFT JOIN dept d ON d.id = u.did WHERE u.age > 18 AND o.amount > 100 GROUP BY d.name HAVING COUNT(o.id) > 2 ORDER BY cnt DESC",
			spec: selectSpec{items: 2, from: 1, joins: 2, where: true, distinct: true, itemAlias: map[int]string{1: "cnt"}},
		},
		{
			name: "派生表",
			sql:  "SELECT * FROM (SELECT id, name FROM users WHERE age > 18) t WHERE t.name LIKE 'a%'",
			spec: selectSpec{items: 1, from: 1, where: true, starItem: true},
		},
		{
			name: "UNION",
			sql:  "SELECT name FROM users UNION SELECT name FROM admins",
			spec: selectSpec{items: 1, from: 1, unions: 1},
		},
		{
			name: "UNION ALL混合",
			sql:  "SELECT name FROM users UNION ALL SELECT name FROM admins UNION SELECT name FROM guests",
			spec: selectSpec{items: 1, from: 1, unions: 2},
		},
		{
			name: "CASE WHEN与函数",
			sql:  "SELECT CASE WHEN score >= 90 THEN 'A' WHEN score >= 60 THEN 'B' ELSE 'C' END AS grade, COALESCE(phone, 'N/A') FROM students",
			spec: selectSpec{items: 2, from: 1, itemAlias: map[int]string{0: "grade"}},
		},
		{
			name: "EXISTS子查询",
			sql:  "SELECT u.* FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.uid = u.id)",
			spec: selectSpec{items: 1, from: 1, where: true, starItem: true},
		},
		{
			name: "IN与标量子查询",
			sql:  "SELECT id, name FROM users WHERE id IN (SELECT uid FROM orders WHERE amount > (SELECT AVG(amount) FROM orders))",
			spec: selectSpec{items: 2, from: 1, where: true},
		},
		{
			name: "CAST函数内AS不得误判别名",
			sql:  "SELECT CAST(age AS CHAR) FROM users",
			spec: selectSpec{items: 1, from: 1, itemAlias: map[int]string{0: ""}},
		},
		{
			name: "聚合嵌套CASE",
			sql:  "SELECT COUNT(*) AS total, SUM(CASE WHEN status = 1 THEN amount ELSE 0 END) AS amt FROM orders",
			spec: selectSpec{items: 2, from: 1, itemAlias: map[int]string{0: "total", 1: "amt"}},
		},
		{
			name: "多行SQL",
			sql:  "SELECT phone\n  FROM users\n  WHERE age > 18\n  ORDER BY id",
			spec: selectSpec{items: 1, from: 1, where: true},
		},
		{
			name: "CROSS JOIN",
			sql:  "SELECT a.id FROM users a CROSS JOIN admins b",
			spec: selectSpec{items: 1, from: 1, joins: 1, itemAlias: map[int]string{0: ""}},
		},
		{
			name: "逗号JOIN",
			sql:  "SELECT u.name, o.amount FROM users u, orders o WHERE o.uid = u.id",
			spec: selectSpec{items: 2, from: 2, where: true},
		},
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
				t.Errorf("[%s] %s: 期望SelectStmt，得到%T(%v)", name, c.name, stmt, stmt.StmtKind())
				continue
			}
			s := c.spec
			if len(sel.Items) != s.items {
				t.Errorf("[%s] %s: items=%d, 期望%d (%+v)", name, c.name, len(sel.Items), s.items, sel.Items)
			}
			if len(sel.From) != s.from {
				t.Errorf("[%s] %s: from=%d, 期望%d (%+v)", name, c.name, len(sel.From), s.from, sel.From)
			}
			if len(sel.Joins) != s.joins {
				t.Errorf("[%s] %s: joins=%d, 期望%d", name, c.name, len(sel.Joins), s.joins)
			}
			if (sel.Where != nil) != s.where {
				t.Errorf("[%s] %s: where存在性=%v, 期望%v", name, c.name, sel.Where != nil, s.where)
			}
			if len(sel.Unions) != s.unions {
				t.Errorf("[%s] %s: unions=%d, 期望%d", name, c.name, len(sel.Unions), s.unions)
			}
			if sel.Distinct != s.distinct {
				t.Errorf("[%s] %s: distinct=%v, 期望%v", name, c.name, sel.Distinct, s.distinct)
			}
			hasStar := false
			for i, it := range sel.Items {
				if it.IsStar() {
					hasStar = true
				}
				if want, ok := s.itemAlias[i]; ok && it.Alias != want {
					t.Errorf("[%s] %s: item[%d].alias=%q, 期望%q", name, c.name, i, it.Alias, want)
				}
			}
			if hasStar != s.starItem {
				t.Errorf("[%s] %s: star项存在性=%v, 期望%v", name, c.name, hasStar, s.starItem)
			}
		}
	}
}

// TestCteParse CTE（WITH）语句：各方言识别为KindWith且保留主查询文本
func TestCteParse(t *testing.T) {
	sqls := []string{
		"WITH cte AS (SELECT id, name FROM users) SELECT c.id, c.name FROM cte c WHERE c.id > 1",
		"WITH a AS (SELECT 1 AS x), b AS (SELECT 2 AS y) SELECT * FROM a, b",
	}
	for _, sql := range sqls {
		for name, p := range allParsers() {
			stmt, err := p.Parse(sql)
			if err != nil {
				t.Errorf("[%s] CTE解析失败: %v", name, err)
				continue
			}
			if stmt.StmtKind() != sqlstmt.KindWith {
				t.Errorf("[%s] CTE期望KindWith，得到%s", name, stmt.StmtKind())
			}
		}
	}
}

// TestDialectSpecificSyntax 方言特有语法（引用符/分页）解析正确
func TestDialectSpecificSyntax(t *testing.T) {
	cases := []struct {
		name    string
		dialect string
		sql     string
		check   func(*testing.T, sqlstmt.Stmt)
	}{
		{
			name: "mysql反引号", dialect: "mysql",
			sql: "SELECT `name`, `age` FROM `users` WHERE `id` = 1",
			check: func(t *testing.T, stmt sqlstmt.Stmt) {
				sel := stmt.(*sqlstmt.SelectStmt)
				if len(sel.Items) != 2 || sel.Items[0].ColumnName != "name" || sel.Items[1].ColumnName != "age" {
					t.Errorf("反引号列名解析错误: %+v", sel.Items)
				}
				if len(sel.From) != 1 || sel.From[0].Name != "users" {
					t.Errorf("反引号表名解析错误: %+v", sel.From)
				}
			},
		},
		{
			name: "pgsql双引号", dialect: "pgsql",
			sql: `SELECT "Name" FROM "Users" WHERE "ID" = 1`,
			check: func(t *testing.T, stmt sqlstmt.Stmt) {
				sel := stmt.(*sqlstmt.SelectStmt)
				if len(sel.Items) != 1 || sel.Items[0].ColumnName != "Name" {
					t.Errorf("双引号列名解析错误: %+v", sel.Items)
				}
				if len(sel.From) != 1 || sel.From[0].Name != "Users" {
					t.Errorf("双引号表名解析错误: %+v", sel.From)
				}
			},
		},
		{
			name: "pgsql分页", dialect: "pgsql",
			sql: "SELECT id FROM users ORDER BY id LIMIT 10 OFFSET 5",
			check: func(t *testing.T, stmt sqlstmt.Stmt) {
				sel := stmt.(*sqlstmt.SelectStmt)
				if sel.Limit == nil {
					t.Error("LIMIT应被解析")
				}
			},
		},
		{
			name: "oracle分页", dialect: "oracle",
			sql: "SELECT id FROM users ORDER BY id FETCH FIRST 10 ROWS ONLY",
			check: func(t *testing.T, stmt sqlstmt.Stmt) {
				sel := stmt.(*sqlstmt.SelectStmt)
				if len(sel.Items) != 1 || sel.From[0].Name != "users" {
					t.Errorf("FETCH FIRST分页解析错误: %+v %+v", sel.Items, sel.From)
				}
			},
		},
		{
			name: "dm分页", dialect: "dm",
			sql: "SELECT id FROM users ORDER BY id LIMIT 10",
			check: func(t *testing.T, stmt sqlstmt.Stmt) {
				sel := stmt.(*sqlstmt.SelectStmt)
				if sel.Limit == nil {
					t.Error("dm的LIMIT应被解析")
				}
			},
		},
	}
	for _, c := range cases {
		stmt, err := allParsers()[c.dialect].Parse(c.sql)
		if err != nil {
			t.Errorf("[%s] 解析失败: %v", c.name, err)
			continue
		}
		c.check(t, stmt)
	}
}

// TestDmlParse DML解析：各方言结构化解析一致
func TestDmlParse(t *testing.T) {
	cases := []struct {
		name  string
		sql   string
		kind  sqlstmt.Kind
		check func(*testing.T, sqlstmt.Stmt)
	}{
		{
			name: "INSERT多值", sql: "INSERT INTO users (id, name) VALUES (1, 'tom'), (2, 'jerry')", kind: sqlstmt.KindInsert,
			check: func(t *testing.T, stmt sqlstmt.Stmt) {
				ins := stmt.(*sqlstmt.InsertStmt)
				if ins.Table.Name != "users" || len(ins.Columns) != 2 {
					t.Errorf("INSERT解析错误: table=%+v cols=%v", ins.Table, ins.Columns)
				}
			},
		},
		{
			name: "INSERT SELECT", sql: "INSERT INTO users SELECT * FROM tmp_users", kind: sqlstmt.KindInsert,
			check: func(t *testing.T, stmt sqlstmt.Stmt) {
				if stmt.(*sqlstmt.InsertStmt).Table.Name != "users" {
					t.Error("INSERT...SELECT目标表解析错误")
				}
			},
		},
		{
			name: "UPDATE多赋值", sql: "UPDATE users SET name = 'x', age = age + 1 WHERE id = 1", kind: sqlstmt.KindUpdate,
			check: func(t *testing.T, stmt sqlstmt.Stmt) {
				upd := stmt.(*sqlstmt.UpdateStmt)
				if len(upd.Tables) != 1 || upd.Tables[0].Name != "users" || len(upd.Set) != 2 || upd.Where == nil {
					t.Errorf("UPDATE解析错误: tables=%+v set=%d where=%v", upd.Tables, len(upd.Set), upd.Where != nil)
				}
			},
		},
		{
			name: "DELETE子查询", sql: "DELETE FROM users WHERE id IN (SELECT uid FROM blacklists)", kind: sqlstmt.KindDelete,
			check: func(t *testing.T, stmt sqlstmt.Stmt) {
				del := stmt.(*sqlstmt.DeleteStmt)
				if len(del.Tables) != 1 || del.Tables[0].Name != "users" || del.Where == nil {
					t.Errorf("DELETE解析错误: tables=%+v where=%v", del.Tables, del.Where != nil)
				}
			},
		},
	}
	for _, c := range cases {
		for name, p := range allParsers() {
			stmt, err := p.Parse(c.sql)
			if err != nil {
				t.Errorf("[%s] %s: 解析失败: %v", name, c.name, err)
				continue
			}
			if stmt.StmtKind() != c.kind {
				t.Errorf("[%s] %s: 期望%s，得到%s", name, c.name, c.kind, stmt.StmtKind())
				continue
			}
			c.check(t, stmt)
		}
	}
}

// TestDdlAndOtherDegrade DDL/管理语句：文本级降级，必须返回非nil stmt且原文保留
func TestDdlAndOtherDegrade(t *testing.T) {
	sqls := []string{
		"CREATE TABLE t_new (id INT PRIMARY KEY, name VARCHAR(64))",
		"ALTER TABLE users ADD COLUMN email VARCHAR(128)",
		"DROP INDEX idx_name ON users",
		"TRUNCATE TABLE users",
		"SHOW TABLES",
		"EXPLAIN SELECT * FROM users",
		"SET @v = 1",
		"/* leading comment */ SELECT id FROM users",
	}
	for _, sql := range sqls {
		for name, p := range allParsers() {
			stmt, err := p.Parse(sql)
			if err != nil {
				t.Errorf("[%s] %q: 解析失败: %v", name, sql, err)
				continue
			}
			if stmt == nil {
				t.Errorf("[%s] %q: 返回nil stmt", name, sql)
				continue
			}
			if stmt.GetText() == "" {
				t.Errorf("[%s] %q: 语句原文丢失", name, sql)
			}
		}
	}
}

// TestQualifiedColumnTableAlias 限定列的TableAlias必须跨方言一致填充（脱敏逐表归属的依赖契约）：
// 纯列引用/带AS别名/带隐式别名的限定列记录去引用后的表限定符；表达式项与无限定列不记录
func TestQualifiedColumnTableAlias(t *testing.T) {
	cases := []struct {
		name  string
		sql   string
		alias string // 第一个select项期望的TableAlias（""表示不应有）
	}{
		{name: "限定列", sql: "SELECT u.phone FROM users u", alias: "u"},
		{name: "限定列AS别名", sql: "SELECT u.phone AS p FROM users u", alias: "u"},
		{name: "限定列隐式别名", sql: "SELECT u.phone p FROM users u", alias: "u"},
		{name: "库名限定取表段", sql: "SELECT db.t.col FROM db.t", alias: "t"},
		{name: "无限定列", sql: "SELECT phone FROM users", alias: ""},
		{name: "函数表达式", sql: "SELECT MAX(u.phone) FROM users u", alias: ""},
		{name: "CASE表达式", sql: "SELECT CASE WHEN a > 1 THEN b END AS c FROM t", alias: ""},
	}
	for _, c := range cases {
		for name, p := range allParsers() {
			stmt, err := p.Parse(c.sql)
			if err != nil {
				t.Errorf("[%s] %s: 解析失败: %v", name, c.name, err)
				continue
			}
			sel, ok := stmt.(*sqlstmt.SelectStmt)
			if !ok || len(sel.Items) == 0 {
				t.Errorf("[%s] %s: 解析结果异常: %T", name, c.name, stmt)
				continue
			}
			if got := sel.Items[0].TableAlias; got != c.alias {
				t.Errorf("[%s] %s: TableAlias=%q, 期望%q", name, c.name, got, c.alias)
			}
		}
	}
	// 方言引用符形态
	dialectCases := []struct {
		dialect string
		sql     string
		alias   string
	}{
		{"mysql", "SELECT `u`.`phone` FROM `users` `u`", "u"},
		{"pgsql", `SELECT "u"."PHONE" FROM "Users" "u"`, "u"},
	}
	for _, c := range dialectCases {
		stmt, err := allParsers()[c.dialect].Parse(c.sql)
		if err != nil {
			t.Errorf("[%s] 引用符形态解析失败: %v", c.dialect, err)
			continue
		}
		sel := stmt.(*sqlstmt.SelectStmt)
		if got := sel.Items[0].TableAlias; got != c.alias {
			t.Errorf("[%s] TableAlias=%q, 期望%q", c.dialect, got, c.alias)
		}
	}
}
