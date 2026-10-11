package sqlparser

// 方言特有形态的建模正确性断言（与快照互补）：
// 快照锁「行为不变」，本文件锁「行为是对的」——每个形态都断言到具体数值/枚举/表数，
// 而不是只断言非空或项数。分页行数与涉及表数会直接决定同步准入与回滚采集的 fail-closed 判定，
// 断言必须有牙（用变异法验证过：把对应解析分支撤掉，用例会红）。

import (
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// parseSel 解析并断言为 SelectStmt
func parseSel(t *testing.T, dialect, sql string) *sqlstmt.SelectStmt {
	t.Helper()
	stmt, err := allParsers()[dialect].Parse(sql)
	if err != nil {
		t.Fatalf("[%s] %s 解析失败: %v", dialect, sql, err)
	}
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok {
		t.Fatalf("[%s] %s 期望 SelectStmt，得到 %T", dialect, sql, stmt)
	}
	return sel
}

// TestKeywordTableIsDialectScoped 词法能力表必须只登记该方言的保留字，不跨方言污染。
//
// 一个词被登记成关键字，就意味着它在该方言的「只能是名称」的位置上要靠 IsUsableName 的结构判据
// 才不会被拒；反过来若把它误登记进全局表，别的方言里这个非保留字的裸表名/列名就会解析失败
// —— 表拿不到、脱敏与回滚静默失效。这里直接钉配置本身
func TestKeywordTableIsDialectScoped(t *testing.T) {
	mysqlOnly := []string{"IGNORE", "QUICK", "DELAYED", "LOW_PRIORITY", "HIGH_PRIORITY", "STRAIGHT_JOIN"}
	oracleOnly := []string{"MINUS"}

	for _, word := range oracleOnly {
		if tokenizer.MysqlConfig.IsKeyword(word) {
			t.Errorf("%s 是 Oracle 系保留字，不应污染 mysql", word)
		}
	}
	for _, word := range mysqlOnly {
		if !tokenizer.MysqlConfig.IsKeyword(word) {
			t.Errorf("mysql 应把本方言保留字 %s 登记为关键字", word)
		}
		if tokenizer.OracleConfig.IsKeyword(word) {
			t.Errorf("%s 是 mysql 保留字，不应污染 oracle（会吃掉裸标识符 %s）", word, word)
		}
		if tokenizer.PgConfig.IsKeyword(word) {
			t.Errorf("%s 是 mysql 保留字，不应污染 pgsql", word)
		}
	}
	for _, word := range oracleOnly {
		if !tokenizer.OracleConfig.IsKeyword(word) || !tokenizer.DmConfig.IsKeyword(word) {
			t.Errorf("Oracle 系应把 %s 登记为关键字（集合操作）", word)
		}
		if tokenizer.PgConfig.IsKeyword(word) || tokenizer.MysqlConfig.IsKeyword(word) {
			t.Errorf("%s 不是 %s 的保留字，登记后该方言里它无法再作裸表名", word, "pg/mysql")
		}
	}
	// USING：支持 JOIN..USING 的方言登记，T-SQL/ClickHouse 保持非保留字
	for _, cfg := range []struct {
		name    string
		config  tokenizer.DialectConfig
		keyword bool
	}{
		{"mysql", tokenizer.MysqlConfig, true},
		{"pgsql", tokenizer.PgConfig, true},
		{"oracle", tokenizer.OracleConfig, true},
		{"dm", tokenizer.DmConfig, true},
		{"sqlite", tokenizer.SQLiteConfig, true},
		{"mssql", tokenizer.MssqlConfig, false},
		{"clickhouse", tokenizer.ClickhouseConfig, false},
	} {
		if got := cfg.config.IsKeyword("USING"); got != cfg.keyword {
			t.Errorf("[%s] USING 关键字登记=%v 期望%v", cfg.name, got, cfg.keyword)
		}
	}
	// 层次查询能力位只声明在 Oracle 系：否则 T-SQL 的 `FROM start WITH (NOLOCK)` 会被当层次子句头
	for _, cfg := range []struct {
		name   string
		config tokenizer.DialectConfig
		want   bool
	}{
		{"oracle", tokenizer.OracleConfig, true},
		{"dm", tokenizer.DmConfig, true},
		{"mysql", tokenizer.MysqlConfig, false},
		{"pgsql", tokenizer.PgConfig, false},
		{"mssql", tokenizer.MssqlConfig, false},
	} {
		if cfg.config.Hierarchic != cfg.want {
			t.Errorf("[%s] Hierarchic 能力位=%v 期望%v", cfg.name, cfg.config.Hierarchic, cfg.want)
		}
	}
}

// TestRowLimitModeled 各方言原生分页写法都必须解析出行数与偏移量。
//
// 只记 Text 不记 Count 等于没建模：消费方（分页改写、同步准入）读的是 Count/Offset。
func TestRowLimitModeled(t *testing.T) {
	cases := []struct {
		name       string
		dialects   []string
		sql        string
		wantCount  int
		wantOffset int
	}{
		{"limit-count", []string{"mysql", "pgsql", "dm"}, "SELECT id FROM t LIMIT 10", 10, 0},
		{"limit-offset", []string{"mysql", "pgsql", "dm"}, "SELECT id FROM t LIMIT 10 OFFSET 20", 10, 20},
		{"limit-comma", []string{"mysql", "dm"}, "SELECT id FROM t LIMIT 20, 10", 10, 20},
		// 标准 OFFSET..FETCH：pg/oracle/达梦都原生支持
		{"offset-fetch", []string{"pgsql", "oracle", "dm"}, "SELECT id FROM t OFFSET 20 ROWS FETCH NEXT 10 ROWS ONLY", 10, 20},
		{"fetch-first", []string{"pgsql", "oracle", "dm"}, "SELECT id FROM t FETCH FIRST 5 ROWS ONLY", 5, 0},
		{"offset-only", []string{"pgsql", "oracle", "dm"}, "SELECT id FROM t OFFSET 7 ROWS", 0, 7},
		// SELECT TOP（达梦/SQL Server 形态）
		{"top", []string{"dm"}, "SELECT TOP 10 id FROM t", 10, 0},
		// 跨方言的宽容消费：LIMIT 出现在非原方言也不能把整段丢在解析之外
		{"limit-on-oracle-lenient", []string{"oracle"}, "SELECT id FROM t LIMIT 10", 10, 0},
	}
	for _, c := range cases {
		for _, d := range c.dialects {
			sel := parseSel(t, d, c.sql)
			if sel.Limit == nil {
				t.Errorf("[%s/%s] 行限制未被建模（limit=nil），分页/准入会按「无限制」处理: %s", d, c.name, c.sql)
				continue
			}
			if sel.Limit.Count != c.wantCount || sel.Limit.Offset != c.wantOffset {
				t.Errorf("[%s/%s] 行限制=%+v，期望 count=%d offset=%d", d, c.name, *sel.Limit, c.wantCount, c.wantOffset)
			}
		}
	}
}

// TestDmlRowLimitGuardModeled UPDATE/DELETE 的尾随行限制必须被结构化捕获。
//
// 这是采集层的 fail-closed 依据：行限制只作用于 WHERE 匹配集的一个子集，
// 漏掉就会回查到全部匹配行，为未被本句影响的行生成覆盖式回滚（DELETE 场景还会重插未删的行）。
func TestDmlRowLimitGuardModeled(t *testing.T) {
	cases := []struct {
		name     string
		dialects []string
		sql      string
		wantRows int // 期望登记的涉及表数
		wantLim  bool
		// 无 WHERE 的行限制形态（mysql 允许 ORDER BY/LIMIT 不带 WHERE）
	}{
		{"update-limit", []string{"mysql", "dm"}, "UPDATE t SET a = 1 WHERE st = 1 LIMIT 10", 1, true},
		{"update-order-limit", []string{"mysql", "dm"}, "UPDATE t SET a = 1 WHERE st = 1 ORDER BY id LIMIT 10", 1, true},
		// 无 WHERE：值扫描必须在 ORDER 处结束，否则 LIMIT 被吞进赋值值文本
		{"update-order-limit-no-where", []string{"mysql", "dm"}, "UPDATE t SET a = 1 ORDER BY id LIMIT 1", 1, true},
		{"delete-limit", []string{"mysql", "dm"}, "DELETE FROM t WHERE st = 1 LIMIT 5", 1, true},
		{"update-limit-on-pg-lenient", []string{"pgsql"}, "UPDATE t SET a = 1 WHERE st = 1 LIMIT 3", 1, true},
		{"update-set-value-keeps-expr", []string{"pgsql"}, "UPDATE t SET a = (SELECT MAX(b) FROM t2), c = 2 WHERE id = 1", 1, false},
	}
	for _, c := range cases {
		for _, d := range c.dialects {
			stmt, err := allParsers()[d].Parse(c.sql)
			if err != nil {
				t.Fatalf("[%s/%s] 解析失败: %v", d, c.name, err)
			}
			var gotLim *sqlstmt.Limit
			var tables []sqlstmt.TableRef
			switch v := stmt.(type) {
			case *sqlstmt.UpdateStmt:
				gotLim, tables, err = v.Limit, v.Tables, nil
				if len(v.Set) == 0 && c.name != "delete-limit" {
					t.Errorf("[%s/%s] SET 赋值未被解析出来", d, c.name)
				}
			case *sqlstmt.DeleteStmt:
				gotLim, tables = v.Limit, v.Tables
			default:
				t.Fatalf("[%s/%s] 期望 Update/Delete，得到 %T", d, c.name, stmt)
			}
			if err != nil {
				t.Fatalf("[%s/%s] %v", d, c.name, err)
			}
			if len(tables) != c.wantRows {
				t.Errorf("[%s/%s] 涉及表数=%d 期望%d（%+v）", d, c.name, len(tables), c.wantRows, tables)
			}
			if (gotLim != nil) != c.wantLim {
				t.Errorf("[%s/%s] 行限制捕获=%v 期望%v（漏掉即采集层 fail-closed 守卫失效）", d, c.name, gotLim != nil, c.wantLim)
			}
		}
	}
}

// TestMultiTableDmlRegistersAllInvolvedTables 多表 DML 必须把来源表一并登记，
// 让「不是单表」成为可核对的结构事实；采集层据此跳过旧值采集，
// 而不是靠「解析中断使 Where 为空」这种副作用碰巧兜住
func TestMultiTableDmlRegistersAllInvolvedTables(t *testing.T) {
	cases := []struct {
		name     string
		dialect  string
		sql      string
		wantFrom []string // 期望登记到的表名（顺序敏感）
		wantCond bool     // WHERE 是否仍被完整解析
	}{
		{"update-join", "mysql", "UPDATE t1 a JOIN t2 b ON a.id = b.id SET a.x = b.y WHERE b.st = 1", []string{"t1", "t2"}, true},
		{"update-comma", "mysql", "UPDATE t1 a, t2 b SET a.x = b.y WHERE a.id = b.id", []string{"t1", "t2"}, true},
		{"update-from", "pgsql", "UPDATE orders o SET total = s.amt FROM sums s WHERE s.id = o.id", []string{"orders", "sums"}, true},
		{"delete-using", "pgsql", "DELETE FROM orders o USING users u WHERE o.user_id = u.id", []string{"orders", "users"}, true},
		{"delete-using", "mysql", "DELETE a FROM t1 a USING t2 b WHERE b.id = a.id", []string{"a", "t1", "t2"}, true},
		{"delete-multi", "mysql", "DELETE a, b FROM t1 a JOIN t2 b ON a.id = b.id WHERE a.st = 1", []string{"a", "b", "t1", "t2"}, true},
		{"delete-simple", "mysql", "DELETE FROM t WHERE id = 1", []string{"t"}, true},
	}
	for _, c := range cases {
		stmt, err := allParsers()[c.dialect].Parse(c.sql)
		if err != nil {
			t.Fatalf("[%s/%s] 解析失败: %v", c.dialect, c.name, err)
		}
		var tables []sqlstmt.TableRef
		var where *sqlstmt.Expr
		switch v := stmt.(type) {
		case *sqlstmt.UpdateStmt:
			tables, where = v.Tables, v.Where
		case *sqlstmt.DeleteStmt:
			tables, where = v.Tables, v.Where
		default:
			t.Fatalf("[%s/%s] 期望 Update/Delete，得到 %T", c.dialect, c.name, stmt)
		}
		if len(tables) != len(c.wantFrom) {
			t.Fatalf("[%s/%s] 涉及表数=%d 期望%d（%+v）", c.dialect, c.name, len(tables), len(c.wantFrom), tables)
		}
		for i, want := range c.wantFrom {
			if tables[i].Name != want {
				t.Errorf("[%s/%s] 表#%d=%q 期望%q", c.dialect, c.name, i, tables[i].Name, want)
			}
		}
		if (where != nil) != c.wantCond {
			t.Errorf("[%s/%s] WHERE 解析=%v 期望%v", c.dialect, c.name, where != nil, c.wantCond)
		}
	}
}

// TestHierarchicClauseConsumed 层次查询子句必须被完整消费且不吞掉相邻子句。
//
// Oracle 文档语序里 START WITH 可在 WHERE 之后，也允许 CONNECT BY 在前；
// START/CONNECT/PRIOR 在 Oracle 系不是保留字，所以这些位置靠「词对匹配 + 子句边界」而不是关键字登记
func TestHierarchicClauseConsumed(t *testing.T) {
	cases := []struct {
		name string
		sql  string
	}{
		{"start-then-connect", "SELECT id, name FROM t START WITH mgr IS NULL CONNECT BY PRIOR id = mgr"},
		{"connect-then-start", "SELECT id FROM t CONNECT BY PRIOR id = mgr START WITH mgr IS NULL"},
		{"nocycle", "SELECT id FROM t START WITH mgr IS NULL CONNECT BY NOCYCLE PRIOR id = mgr"},
		{"after-where", "SELECT id FROM t WHERE st = 1 START WITH mgr IS NULL CONNECT BY PRIOR id = mgr"},
		{"with-order", "SELECT id FROM t START WITH mgr IS NULL CONNECT BY PRIOR id = mgr ORDER BY id"},
		{"order-siblings", "SELECT id FROM t START WITH mgr IS NULL CONNECT BY PRIOR id = mgr ORDER SIBLINGS BY name"},
		{"with-pagination", "SELECT id FROM t START WITH mgr IS NULL CONNECT BY PRIOR id = mgr OFFSET 5 ROWS FETCH NEXT 10 ROWS ONLY"},
	}
	for _, d := range []string{"oracle", "dm"} {
		for _, c := range cases {
			sel := parseSel(t, d, c.sql)
			if got, want := normText(sel.GetText()), normText(c.sql); got != want {
				t.Errorf("[%s/hier-%s] 层次子句未完整消费\n got=%q\nwant=%q", d, c.name, got, want)
			}
		}
	}
	// START WITH 之后仍要能拿到分页行数（层次子句不能把尾部子句挤掉）
	sel := parseSel(t, "oracle", cases[len(cases)-1].sql)
	if sel.Limit == nil || sel.Limit.Count != 10 || sel.Limit.Offset != 5 {
		t.Errorf("[oracle/hier-with-pagination] 层次子句后的行限制丢失: %+v", sel.Limit)
	}
	// ORDER SIBLINGS BY 的排序列必须落到 OrderBy
	sel = parseSel(t, "oracle", "SELECT id FROM t START WITH mgr IS NULL CONNECT BY PRIOR id = mgr ORDER SIBLINGS BY name")
	if len(sel.OrderBy) != 1 || sel.OrderBy[0].Text != "name" {
		t.Errorf("[oracle/order-siblings] 排序未被建模: %+v", sel.OrderBy)
	}
}

// TestLockingClauseConsumed FOR UPDATE 及其修饰词必须整段消费（各方言共用同一骨架实现）
func TestLockingClauseConsumed(t *testing.T) {
	cases := []struct {
		name     string
		dialects []string
		sql      string
	}{
		{"for-update", allDialects, "SELECT id FROM t WHERE id = 1 FOR UPDATE"},
		{"for-share", []string{"pgsql"}, "SELECT id FROM t WHERE id = 1 FOR SHARE"},
		{"of-nowait", []string{"oracle", "dm", "pgsql"}, "SELECT id FROM t WHERE id = 1 FOR UPDATE OF t NOWAIT"},
		{"skip-locked", []string{"oracle", "dm", "pgsql"}, "SELECT id FROM t WHERE id = 1 FOR UPDATE SKIP LOCKED"},
		{"wait-n", []string{"oracle", "dm"}, "SELECT id FROM t WHERE id = 1 FOR UPDATE WAIT 3"},
	}
	for _, c := range cases {
		for _, d := range c.dialects {
			sel := parseSel(t, d, c.sql)
			if got, want := normText(sel.GetText()), normText(c.sql); got != want {
				t.Errorf("[%s/%s] 锁子句未完整消费（残留会让 RETURNING/文本回放失真）\n got=%q\nwant=%q", d, c.name, got, want)
			}
			if sel.Where == nil {
				t.Errorf("[%s/%s] WHERE 丢失", d, c.name)
			}
		}
	}
}

// TestJoinKindModeled 连接类型必须落到枚举，USING/CROSS/NATURAL/STRAIGHT_JOIN 都不得泄漏成 FROM 表
func TestJoinKindModeled(t *testing.T) {
	cases := []struct {
		name       string
		dialects   []string
		sql        string
		wantFrom   int
		wantJoins  int
		wantKind   sqlstmt.JoinKind
		wantOnExpr bool
	}{
		{"straight-join", []string{"mysql"}, "SELECT a.id FROM t1 a STRAIGHT_JOIN t2 b ON a.id = b.id", 1, 1, sqlstmt.JoinKindInner, true},
		{"natural", allDialects, "SELECT a.id FROM t1 a NATURAL JOIN t2 b WHERE a.id > 5", 1, 1, sqlstmt.JoinKindNatural, false},
		{"cross", allDialects, "SELECT a.id FROM t1 a CROSS JOIN t2 b", 1, 1, sqlstmt.JoinKindCross, false},
		{"using", allDialects, "SELECT a.id FROM t1 a INNER JOIN t2 b USING (id)", 1, 1, sqlstmt.JoinKindInner, false},
		{"left-outer", allDialects, "SELECT a.id FROM t1 a LEFT OUTER JOIN t2 b ON a.id = b.id", 1, 1, sqlstmt.JoinKindLeft, true},
		{"full", allDialects, "SELECT a.id FROM t1 a FULL JOIN t2 b ON a.id = b.id", 1, 1, sqlstmt.JoinKindFull, true},
	}
	for _, c := range cases {
		for _, d := range c.dialects {
			sel := parseSel(t, d, c.sql)
			if len(sel.From) != c.wantFrom {
				t.Errorf("[%s/%s] FROM 表数=%d 期望%d（连接词或条件列泄漏成表？）: %+v", d, c.name, len(sel.From), c.wantFrom, sel.From)
			}
			if len(sel.Joins) != c.wantJoins {
				t.Errorf("[%s/%s] JOIN 数=%d 期望%d", d, c.name, len(sel.Joins), c.wantJoins)
				continue
			}
			if sel.Joins[0].Kind != c.wantKind {
				t.Errorf("[%s/%s] 连接类型=%d 期望%d", d, c.name, sel.Joins[0].Kind, c.wantKind)
			}
			if (sel.Joins[0].On != nil) != c.wantOnExpr {
				t.Errorf("[%s/%s] ON 存在性=%v 期望%v", d, c.name, sel.Joins[0].On != nil, c.wantOnExpr)
			}
			if sel.Where != nil && c.name == "natural" && normText(sel.Where.Text) != "a.id > 5" {
				t.Errorf("[%s/%s] NATURAL JOIN 后的 WHERE 被改写: %q", d, c.name, sel.Where.Text)
			}
		}
	}
}

// TestDerivedTableContract 派生表的表引用契约：Name 只含括号表达式本身，别名单独在 Alias。
//
// 脱敏血缘按「Name 含 ( 就重新解析内层 SQL」工作（db_mask.addTableRef），
// 别名混进 Name 会让内层文本多带一段；曾四方言都把别名写进 Name
func TestDerivedTableContract(t *testing.T) {
	const sql = "SELECT x.id FROM (SELECT id FROM t1 WHERE id > 0) x WHERE x.id > 1"
	for _, d := range allDialects {
		sel := parseSel(t, d, sql)
		if len(sel.From) != 1 {
			t.Fatalf("[%s] FROM 表数=%d 期望1: %+v", d, len(sel.From), sel.From)
		}
		ref := sel.From[0]
		if ref.Name != "(SELECT id FROM t1 WHERE id > 0)" {
			t.Errorf("[%s] 派生表 Name 不应含别名或后继子句: %q", d, ref.Name)
		}
		if ref.Alias != "x" {
			t.Errorf("[%s] 派生表 Alias=%q 期望 x", d, ref.Alias)
		}
		if !strings.HasPrefix(ref.Name, "(") {
			t.Errorf("[%s] 派生表 Name 应以括号起始（血缘按此特征识别派生表）: %q", d, ref.Name)
		}
		if sel.Where == nil || normText(sel.Where.Text) != "x.id > 1" {
			t.Errorf("[%s] 外层 WHERE 丢失: %+v", d, sel.Where)
		}
	}
}

// TestUnionOperandContract 集合操作：操作数文本、分支级 ORDER BY/LIMIT 上提、UNION 段文本
func TestUnionOperandContract(t *testing.T) {
	sel := parseSel(t, "pgsql", "SELECT a FROM t1 UNION ALL SELECT a FROM t2 ORDER BY a LIMIT 3")
	if len(sel.Unions) != 1 {
		t.Fatalf("UNION 段数=%d 期望1", len(sel.Unions))
	}
	u := sel.Unions[0]
	if u.Op != "UNION" || !u.All {
		t.Errorf("UNION 段=%s all=%v，期望 UNION all=true", u.Op, u.All)
	}
	// 段文本是该子句的原文 span（含随后被上提到语句级的 ORDER BY/LIMIT）
	if !strings.HasPrefix(u.Text, "UNION ALL SELECT a FROM t2") {
		t.Errorf("UNION 段文本=%q，应以操作符起始并含操作数", u.Text)
	}
	if u.Select == nil {
		t.Fatal("UNION 操作数未建树")
	}
	// 操作数文本必须覆盖完整的 `SELECT a FROM t2`：旧实现在构造子句体时过早回填 Base.Text，
	// 使分支文本永久停在子句中间（曾实测为 "SELECT a FROM"）
	if !strings.HasPrefix(u.Select.GetText(), "SELECT a FROM t2") {
		t.Errorf("UNION 操作数文本=%q（未覆盖完整分支）", u.Select.GetText())
	}
	// 尾部的全局排序与分页属于整条语句，必须上提到外层
	if len(sel.OrderBy) != 1 || sel.OrderBy[0].Text != "a" {
		t.Errorf("全局 ORDER BY 未上提: %+v", sel.OrderBy)
	}
	if sel.Limit == nil || sel.Limit.Count != 3 {
		t.Errorf("全局 LIMIT 未上提: %+v", sel.Limit)
	}
	if len(u.Select.OrderBy) != 0 || u.Select.Limit != nil {
		t.Errorf("分支内不应保留已被上提的 ORDER BY/LIMIT: order=%+v limit=%+v", u.Select.OrderBy, u.Select.Limit)
	}
}

// TestNonReservedWordsStillWorkAsNames 能力表登记的边界不能吃掉这些词作裸标识符的合法用法。
//
// 这是把词登记为关键字/边界时必须守住的反面约束
func TestNonReservedWordsStillWorkAsNames(t *testing.T) {
	// minus 只在 Oracle 系是保留字：pg/mysql 里它是普通表名
	for _, d := range []string{"pgsql", "mysql"} {
		sel := parseSel(t, d, "SELECT a FROM minus")
		if len(sel.From) != 1 || sel.From[0].Name != "minus" {
			t.Errorf("[%s] 表名 minus 解析异常: %+v", d, sel.From)
		}
	}
	// start 在 Oracle 系也不是保留字：作表名、列名都要能用
	for _, d := range allDialects {
		sel := parseSel(t, d, "SELECT start FROM start")
		if len(sel.From) != 1 || sel.From[0].Name != "start" {
			t.Errorf("[%s] 表名 start 解析异常: %+v", d, sel.From)
		}
		if len(sel.Items) != 1 || sel.Items[0].ColumnName != "start" {
			t.Errorf("[%s] 列名 start 解析异常: %+v", d, sel.Items)
		}
	}
	// 隐式别名位上的非保留字关键字（full 是连接词前缀，但可作 AS 后的别名）
	sel := parseSel(t, "mysql", "SELECT id AS `left`, name AS full FROM t")
	if len(sel.Items) != 2 || sel.Items[0].Alias != "left" {
		t.Errorf("[mysql] 保留字形态的别名解析异常: %+v", sel.Items)
	}
}

// TestStarItemContract 星号项：Kind 表达「全部列」，ColumnName 不得再填 `*`（噪声），
// t.* 的限定符进 TableAlias
func TestStarItemContract(t *testing.T) {
	for _, d := range allDialects {
		sel := parseSel(t, d, "SELECT *, t.* FROM t")
		if len(sel.Items) != 2 {
			t.Fatalf("[%s] 星号项数=%d: %+v", d, len(sel.Items), sel.Items)
		}
		for i, it := range sel.Items {
			if !it.IsStar() {
				t.Errorf("[%s] item[%d] Kind=%d 期望星号", d, i, it.Kind)
			}
			if it.ColumnName != "" {
				t.Errorf("[%s] item[%d] 星号项不应有 ColumnName=%q", d, i, it.ColumnName)
			}
		}
		if sel.Items[1].TableAlias != "t" {
			t.Errorf("[%s] t.* 的 TableAlias=%q 期望 t", d, sel.Items[1].TableAlias)
		}
	}
}

// TestInsertConflictForms 各 upsert 形态都要标记冲突处理，采集层据此跳过插入快照
func TestInsertConflictForms(t *testing.T) {
	cases := []struct {
		dialect string
		sql     string
		want    bool
	}{
		{"mysql", "INSERT INTO t (a) VALUES (1)", false},
		{"mysql", "REPLACE INTO t (a) VALUES (1)", true},
		{"mysql", "INSERT IGNORE INTO t (a) VALUES (1)", true},
		{"mysql", "INSERT DELAYED IGNORE INTO t (a) VALUES (1)", true},
		{"mysql", "INSERT INTO t (a) VALUES (1) ON DUPLICATE KEY UPDATE a = 2", true},
		{"pgsql", "INSERT INTO t (a) VALUES (1) ON CONFLICT (a) DO NOTHING", true},
		{"pgsql", "INSERT INTO t (a) VALUES (1) RETURNING id", false},
		{"oracle", "INSERT INTO t (a) VALUES (1)", false},
		{"oracle", "INSERT ALL INTO t1 (a) VALUES (1) INTO t2 (b) VALUES (2) SELECT 1 FROM dual", false},
	}
	for _, c := range cases {
		stmt, err := allParsers()[c.dialect].Parse(c.sql)
		if err != nil {
			t.Fatalf("[%s] %s 解析失败: %v", c.dialect, c.sql, err)
		}
		ins, ok := stmt.(*sqlstmt.InsertStmt)
		if !ok {
			t.Fatalf("[%s] %s 期望 InsertStmt，得到 %T", c.dialect, c.sql, stmt)
		}
		if ins.HasConflictClause != c.want {
			t.Errorf("[%s] %s 冲突标记=%v 期望%v", c.dialect, c.sql, ins.HasConflictClause, c.want)
		}
		if got, want := normText(ins.GetText()), normText(c.sql); got != want {
			t.Errorf("[%s] %s 未被完整消费\n got=%q\nwant=%q", c.dialect, c.sql, got, want)
		}
	}
	// INSERT ALL 是多表插入：不登记单一目标表，采集层因此不会猜一张表去删行
	stmt, _ := allParsers()["oracle"].Parse("INSERT ALL INTO t1 (a) VALUES (1) INTO t2 (b) VALUES (2) SELECT 1 FROM dual")
	if ins := stmt.(*sqlstmt.InsertStmt); ins.Table.Name != "" || ins.ValueRows != nil {
		t.Errorf("[oracle] INSERT ALL 不应登记单表或字面量行: table=%+v rows=%v", ins.Table, ins.ValueRows)
	}
}

// TestItemTextKeepsQuoteContract SelectItem.Text 保留原文（含引用符），去引用后的名字在 ColumnName/Alias。
// 整体 Unquote 会把 `u`.`phone` 破坏成 u`.`phone，污染限定符分段
func TestItemTextKeepsQuoteContract(t *testing.T) {
	sel := parseSel(t, "mysql", "SELECT `u`.`phone` FROM t u")
	if len(sel.Items) != 1 {
		t.Fatalf("项数=%d", len(sel.Items))
	}
	it := sel.Items[0]
	if it.Text != "`u`.`phone`" {
		t.Errorf("Text 应保留引用符: %q", it.Text)
	}
	if it.ColumnName != "phone" || it.TableAlias != "u" {
		t.Errorf("语义字段应已去引用: col=%q tbl=%q", it.ColumnName, it.TableAlias)
	}
	sel = parseSel(t, "pgsql", `SELECT "u"."PHONE" FROM t u`)
	if sel.Items[0].Text != `"u"."PHONE"` || sel.Items[0].ColumnName != "PHONE" {
		t.Errorf("[pgsql] 双引号列项解析异常: %+v", sel.Items[0])
	}
}
