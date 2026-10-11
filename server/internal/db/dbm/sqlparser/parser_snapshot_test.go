package sqlparser

// SQL 解析行为快照（特征化测试）——方言解析收敛到 base 的行为基线。
//
// 为什么需要它：四方言的 SELECT/DML 解析曾是四份近似拷贝，漂移会以「静默少解析一段」
// 的形式出现（如 dm 缺 NATURAL 分支丢掉整个 JOIN、USING 泄漏成 FROM 表）。单测断言
// 只覆盖写测试时想到的字段，漂移常发生在未断言的字段上。快照把整棵 AST 的规范化
// 结构固定下来，任何字段的行为变化都会在 diff 里显形，从而让「重构=行为保持」可证。
//
// 用法：
//
//	go test ./internal/db/dbm/sqlparser/ -run TestParserSnapshot          # 校验零 diff
//	go test ./internal/db/dbm/sqlparser/ -run TestParserSnapshot -update  # 重新生成基线
//
// 规范化约定（为了 diff 只反映真实行为变化，不反映噪声）：
//   - 所有文本字段折叠连续空白并 Trim，故缩进/换行差异不产生 diff
//   - 标识符引用符原样保留（`x` / "x" / x 是不同行为，必须可见）
//   - 表达式同时记录原文本与 Pratt 树（树为 nil 即「降级为原文」，是显式状态而非缺失）
import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

var updateSnapshot = flag.Bool("update", false, "重新生成 SQL 解析行为快照基线")

// snapshotDir 快照基线目录（相对本测试包）
const snapshotDir = "testdata/snapshot"

// allDialects 参与快照的 4 个方言名（与 allParsers 的 key 一致）
var allDialects = []string{"mysql", "pgsql", "oracle", "dm"}

// except 返回除 named 之外的全部方言名，用于标注「除某方言外都原生支持」
func except(named ...string) []string {
	out := make([]string, 0, len(allDialects))
	for _, d := range allDialects {
		if !contains(named, d) {
			out = append(out, d)
		}
	}
	return out
}

// snapshotCorpus 快照覆盖的 SQL 语料。
// 选取原则：每条都针对一个「解析器必须做出正确结构决策」的点，且覆盖各方言的分歧面
// （分页写法、层次查询、冲突子句、行限制、标识符引用形态）。同一份语料喂给 4 个方言，
// 各方言各得一份基线，方言间差异因此可见、可审。
//
// dialects 标注该形态被哪些方言**原生支持**（见 corpusScope）。喂给不支持它的方言时，
// 宽容解析的降级结果只记录进快照、不做断言——降级不是缺陷，把缺陷当降级才是。
var snapshotCorpus = []struct {
	id  string
	sql string
}{
	// ---------- 投影与标识符形态 ----------
	{"proj-star", "SELECT * FROM t1"},
	{"proj-table-star", "SELECT a.*, b.id FROM t1 a, t2 b"},
	{"proj-alias-as", "SELECT id AS ident, name n2 FROM users u"},
	{"proj-quoted-ident", `SELECT "id", "name" AS n FROM "users"`},
	{"proj-expr-alias", "SELECT COUNT(*) AS cnt, MAX(price) mx FROM orders o"},
	{"proj-case-when", "SELECT CASE WHEN a > 1 THEN 'x' ELSE 'y' END AS c FROM t"},
	{"proj-cast-as", "SELECT CAST(a AS CHAR) AS c FROM t"},
	{"proj-reserved-alias", "SELECT id AS `left`, name AS full FROM t"},
	{"proj-concat", "SELECT a || '-' || b AS combo FROM t"},
	{"proj-window-degraded", "SELECT ROW_NUMBER() OVER (PARTITION BY a ORDER BY b) rn FROM t"},

	// ---------- FROM / JOIN ----------
	{"from-comma-multi", "SELECT a.id FROM t1 a, t2 b, t3 c WHERE a.id = b.id"},
	{"from-schema-qualified", "SELECT id FROM mydb.users"},
	{"from-minus-as-name", "SELECT a FROM minus"},
	{"from-start-as-name", "SELECT a FROM start"},
	{"from-using-as-name", "SELECT a FROM using WHERE using = 1"},
	{"from-mysql-keyword-as-name", "SELECT a FROM ignore WHERE ignore = 1"},
	{"insert-mysql-keyword-as-column", "INSERT INTO t (ignore, quick) VALUES (1, 2)"},
	{"from-derived", "SELECT x.id FROM (SELECT id FROM t1 WHERE id > 0) x"},
	{"from-derived-union", "SELECT x.id FROM (SELECT id FROM t1 UNION SELECT id FROM t2) x WHERE x.id > 1"},
	{"join-inner-on", "SELECT a.id FROM t1 a JOIN t2 b ON a.id = b.id"},
	{"join-left-outer-on", "SELECT a.id FROM t1 a LEFT OUTER JOIN t2 b ON a.id = b.id AND b.st = 1"},
	{"join-right-on", "SELECT a.id FROM t1 a RIGHT JOIN t2 b ON a.id = b.id"},
	{"join-full-on", "SELECT a.id FROM t1 a FULL OUTER JOIN t2 b ON a.id = b.id"},
	{"join-cross", "SELECT a.id FROM t1 a CROSS JOIN t2 b"},
	{"join-straight", "SELECT a.id FROM t1 a STRAIGHT_JOIN t2 b ON a.id = b.id WHERE a.id > 1"},
	{"join-natural", "SELECT a.id FROM t1 a NATURAL JOIN t2 b WHERE a.id > 5"},
	{"join-using", "SELECT a.id FROM t1 a INNER JOIN t2 b USING (id) WHERE a.id > 5"},
	{"join-chain", "SELECT a.id FROM t1 a JOIN t2 b ON a.id = b.id LEFT JOIN t3 c ON c.id = b.id JOIN t4 d ON d.cid = c.id"},
	{"join-derived-on-derived", "SELECT x.id FROM (SELECT id FROM t1) x JOIN (SELECT id FROM t2) y ON x.id = y.id"},
	{"join-on-subquery", "SELECT a.id FROM t1 a JOIN t2 b ON a.id = b.id AND b.typ IN (SELECT typ FROM cfg WHERE on = 1)"},

	// ---------- WHERE / GROUP / HAVING / ORDER ----------
	{"where-and-or-not", "SELECT id FROM t WHERE (a = 1 OR b = 2) AND NOT c = 3"},
	{"where-in-list", "SELECT id FROM t WHERE id IN (1, 2, 3)"},
	{"where-not-in-subquery", "SELECT id FROM t WHERE id NOT IN (SELECT id FROM t2 WHERE s = 1)"},
	{"where-exists", "SELECT id FROM t WHERE EXISTS (SELECT 1 FROM t2 WHERE t2.id = t.id)"},
	{"where-between", "SELECT id FROM t WHERE created BETWEEN '2026-01-01' AND '2026-12-31'"},
	{"where-like-escape", "SELECT id FROM t WHERE name LIKE '%a\\_%' ESCAPE '\\'"},
	{"where-isnull", "SELECT id FROM t WHERE a IS NULL AND b IS NOT NULL"},
	{"where-cmp-operator", "SELECT id FROM t WHERE a <> 1 AND b != 2 AND c <= 3 AND d >= 4"},
	{"where-arithmetic", "SELECT id FROM t WHERE a + b * 2 - c / 3 > 10"},
	{"where-isolation-level", "SELECT id FROM t WHERE name = 'a''b'"},
	{"where-subquery-scalar", "SELECT id FROM t WHERE price > (SELECT AVG(price) FROM t2)"},
	{"where-rownum", "SELECT id FROM t WHERE ROWNUM <= 10 AND st = 1"},
	{"group-having", "SELECT d, COUNT(1) c FROM t GROUP BY d HAVING COUNT(1) > 2"},
	{"order-multi", "SELECT id FROM t ORDER BY a ASC, b DESC, c"},
	{"order-expr", "SELECT id FROM t ORDER BY a + b DESC"},
	{"order-ordinal", "SELECT id, name FROM t ORDER BY 1, 2"},

	// ---------- 分页：各方言分歧面 ----------
	{"page-limit-count", "SELECT id FROM t LIMIT 10"},
	{"page-limit-offset", "SELECT id FROM t LIMIT 10 OFFSET 20"},
	{"page-limit-comma", "SELECT id FROM t LIMIT 20, 10"},
	{"page-offset-first", "SELECT id FROM t OFFSET 20 ROWS FETCH NEXT 10 ROWS ONLY"},
	{"page-fetch-first-only", "SELECT id FROM t FETCH FIRST 5 ROWS ONLY"},
	{"page-limit-all", "SELECT id FROM t LIMIT ALL"},
	{"page-select-top", "SELECT TOP 10 id FROM t"},
	{"page-with-order", "SELECT id FROM t ORDER BY id DESC LIMIT 5 OFFSET 5"},

	// ---------- 集合操作 ----------
	{"set-union", "SELECT a FROM t1 UNION SELECT a FROM t2"},
	{"set-union-all", "SELECT a FROM t1 UNION ALL SELECT a FROM t2"},
	{"set-except", "SELECT a FROM t1 EXCEPT SELECT a FROM t2"},
	{"set-intersect", "SELECT a FROM t1 INTERSECT SELECT a FROM t2"},
	{"set-minus", "SELECT a FROM t1 MINUS SELECT a FROM t2"},
	{"set-three-way-tail-order", "SELECT a FROM t1 UNION SELECT a FROM t2 UNION SELECT a FROM t3 ORDER BY a LIMIT 3"},
	{"set-parenthesized", "(SELECT a FROM t1) UNION (SELECT a FROM t2)"},
	{"select-into-outfile", "SELECT a FROM t INTO OUTFILE '/tmp/x.txt'"},
	{"select-into-table", "SELECT id INTO bak_users FROM users WHERE id > 1"},

	// ---------- 层次查询（oracle/dm） ----------
	{"hier-connect-by", "SELECT id, name FROM t START WITH mgr IS NULL CONNECT BY PRIOR id = mgr"},
	{"hier-where-then-start", "SELECT id FROM t WHERE st = 1 START WITH mgr IS NULL CONNECT BY PRIOR id = mgr"},
	{"hier-order-siblings", "SELECT id FROM t START WITH mgr IS NULL CONNECT BY PRIOR id = mgr ORDER SIBLINGS BY name"},

	// ---------- 锁与返回 ----------
	{"lock-for-update", "SELECT id FROM t WHERE id = 1 FOR UPDATE"},
	{"lock-for-update-nowait", "SELECT id FROM t WHERE id = 1 FOR UPDATE NOWAIT"},
	{"lock-for-update-skip", "SELECT id FROM t WHERE id = 1 FOR UPDATE SKIP LOCKED"},
	{"dml-returning", "UPDATE t SET a = 1 WHERE id = 1 RETURNING id"},

	// ---------- INSERT ----------
	{"insert-values-multi-row", "INSERT INTO t (id, name) VALUES (1, 'a'), (2, 'b')"},
	{"insert-no-column-list", "INSERT INTO t VALUES (1, 'a')"},
	{"insert-schema-qualified", `INSERT INTO mydb."T" ("Id", "Name") VALUES (1, 'a')`},
	{"insert-backtick-quoted", "INSERT INTO mydb.`T` (`Id`, `Name`) VALUES (1, 'a')"},
	{"insert-select", "INSERT INTO t (id) SELECT id FROM t2 WHERE st = 1"},
	{"insert-all", "INSERT ALL INTO t1 (a) VALUES (1) INTO t2 (b) VALUES (2) SELECT 1 FROM dual"},
	{"insert-delayed", "INSERT DELAYED IGNORE INTO t (a) VALUES (1)"},
	{"insert-values-returning", "INSERT INTO t (a) VALUES (1) RETURNING id"},
	{"insert-on-duplicate", "INSERT INTO t (id, name) VALUES (1, 'a') ON DUPLICATE KEY UPDATE name = 'b'"},
	{"insert-on-conflict", "INSERT INTO t (id, name) VALUES (1, 'a') ON CONFLICT (id) DO UPDATE SET name = 'b'"},
	{"insert-function-value", "INSERT INTO t (id, ct) VALUES (1, NOW())"},
	{"insert-subquery-value", "INSERT INTO t (id) VALUES ((SELECT MAX(id) FROM t2))"},

	// ---------- UPDATE ----------
	{"update-single", "UPDATE t SET a = 1 WHERE id = 2"},
	{"update-multi-set", "UPDATE t SET a = 1, b = 'x', c = NULL WHERE id = 2"},
	{"update-expr-value", "UPDATE t SET cnt = cnt + 1 WHERE id = 2"},
	{"update-alias-set", "UPDATE t x SET x.a = 1 WHERE x.id = 2"},
	{"update-quoted-col", `UPDATE "T" SET "Id" = 1 WHERE "Id" = 2`},
	{"update-subquery-set", "UPDATE t SET a = (SELECT MAX(b) FROM t2) WHERE id = 2"},
	{"update-limit", "UPDATE t SET a = 1 WHERE st = 1 LIMIT 10"},
	{"update-order-limit", "UPDATE t SET a = 1 WHERE st = 1 ORDER BY id LIMIT 10"},
	{"update-join-mysql", "UPDATE t1 a JOIN t2 b ON a.id = b.id SET a.x = b.y WHERE b.st = 1"},
	{"update-from-source", "UPDATE orders o SET total = s.amt FROM sums s WHERE s.id = o.id"},
	{"update-order-limit-no-where", "UPDATE t SET a = 1 ORDER BY id LIMIT 1"},
	{"update-low-priority-ignore", "UPDATE LOW_PRIORITY IGNORE t SET a = 1 WHERE id = 1"},
	{"update-no-where", "UPDATE t SET a = 1"},
	{"update-schema-qualified", "UPDATE mydb.users SET name = 'x' WHERE id = 1"},

	// ---------- DELETE ----------
	{"delete-simple", "DELETE FROM t WHERE id = 2"},
	{"delete-alias-where", "DELETE FROM t AS x WHERE x.id = 2"},
	{"delete-limit", "DELETE FROM t WHERE st = 1 LIMIT 5"},
	{"delete-no-where", "DELETE FROM t"},
	{"delete-using-mysql", "DELETE a FROM t1 a USING t2 b WHERE b.id = a.id"},
	{"delete-multi-table-mysql", "DELETE a, b FROM t1 a JOIN t2 b ON a.id = b.id WHERE a.st = 1"},
	{"delete-modifiers-mysql", "DELETE LOW_PRIORITY QUICK IGNORE FROM t WHERE id = 1"},
	{"delete-subquery", "DELETE FROM t WHERE id IN (SELECT id FROM t2 WHERE st = 1)"},
	{"delete-schema-qualified", "DELETE FROM mydb.users WHERE id = 1"},

	// ---------- 非 DML（分类相关） ----------
	{"ddl-create", "CREATE TABLE t (id INT, name VARCHAR(20))"},
	{"ddl-alter", "ALTER TABLE t ADD COLUMN age INT"},
	{"ddl-drop", "DROP TABLE IF EXISTS t"},
	{"ddl-truncate", "TRUNCATE TABLE t"},
	{"cte-with", "WITH c AS (SELECT id FROM t) SELECT id FROM c"},
	{"stmt-show", "SHOW TABLES"},
	{"stmt-unknown", "VACUUM ANALYZE t"},
}

// corpusScope 语料的方言适用面例外表：仅登记「并非所有方言都原生支持」的条目，
// 未登记的条目默认四方言通用。单一真源：完整消费不变量与跨方言结构比对都从此表取范围。
//
// 注意 pg 解析器同时服务 SQLite/ClickHouse/MSSQL，所以 pg 行丢失某个行限制形态
// 并不只是「该方言不支持所以无所谓」——它会连带让共用语法的 fail-closed 守卫失效。
var corpusScope = map[string][]string{
	"proj-concat":              except("mysql"), // mysql 默认模式下 || 是逻辑或而非拼接
	"where-rownum":             {"oracle", "dm"},
	"page-limit-count":         except("oracle"),
	"page-limit-offset":        except("oracle"),
	"page-limit-comma":         {"mysql", "dm"},
	"page-offset-first":        except("mysql"),
	"page-fetch-first-only":    except("mysql"),
	"page-limit-all":           {"pgsql"},
	"page-select-top":          {"dm"},
	"page-with-order":          except("oracle"),
	"set-minus":                {"oracle", "dm"},
	"set-three-way-tail-order": except("oracle"),
	"hier-connect-by":          {"oracle", "dm"},
	"lock-for-update-nowait":   {"oracle", "dm"},
	"lock-for-update-skip":     except("mysql"),
	"dml-returning":            {"pgsql", "oracle"},
	"insert-on-duplicate":      {"mysql"},
	"insert-on-conflict":       {"pgsql"},
	// 双引号作标识符引用符是 pg/oracle/dm 的语义；mysql 默认模式下 "T" 是字符串字面量
	"insert-schema-qualified": {"pgsql", "oracle", "dm"},
	"insert-backtick-quoted":  {"mysql"},
	"update-limit":            {"mysql", "dm"},
	"update-order-limit":      {"mysql", "dm"},
	"update-join-mysql":       {"mysql"},
	"delete-alias-where":      {"pgsql", "dm"},
	"delete-limit":            {"mysql", "dm"},
	"delete-using-mysql":      {"mysql"},
	"stmt-show":               {"mysql"},
	"join-straight":           {"mysql"},
	"from-minus-as-name":      {"mysql", "pgsql"},
	"from-start-as-name":      {"mysql", "pgsql"},
	// 非保留字关键字作裸表名/列名：各方言都必须能解析出自己那一侧（USING 在 T-SQL/CH 是裸名，
	// QUICK/DELAYED 只在 mysql 登记；MINUS 只在 Oracle 系是保留字）
	// USING/IGNORE/QUICK 只在 mysql（及 pg 的 USING）是保留字，登记在各方言自己一侧后，
	// 另一个方言里它们仍是可作裸表名/列名的普通标识符——下面三条分别钉这个「不互相污染」的方向：
	// T-SQL/CH 里 USING 非保留字，pg/oracle/达梦 里 mysql 的那几个修饰词仍是普通名字
	"from-using-as-name":             {"mssql", "clickhouse"},
	"from-mysql-keyword-as-name":     {"pgsql", "oracle", "dm"},
	"insert-mysql-keyword-as-column": {"pgsql"},
	// JOIN..USING 只有把 USING 视为保留字/支持该写法的方言才谈得上完整消费
	"join-using": {"mysql", "pgsql", "oracle", "dm"},
	// INTO 写出目的地：mysql 的 INTO OUTFILE 在语句尾，pg/SQL Server 的 SELECT INTO 在投影之后
	"select-into-outfile":         {"mysql"},
	"select-into-table":           {"pgsql", "mssql"},
	"hier-where-then-start":       {"oracle", "dm"},
	"hier-order-siblings":         {"oracle", "dm"},
	"update-from-source":          {"pgsql"},
	"update-order-limit-no-where": {"mysql", "dm"},
	"update-low-priority-ignore":  {"mysql"},
	"delete-multi-table-mysql":    {"mysql"},
	"delete-modifiers-mysql":      {"mysql"},
	"insert-all":                  {"oracle", "dm"},
	"insert-delayed":              {"mysql"},
	"insert-values-returning":     {"pgsql", "oracle"},
}

// scopeOf 返回该语料条目适用的方言集合
func scopeOf(id string) []string {
	if s, ok := corpusScope[id]; ok {
		return s
	}
	return allDialects
}

// TestParserSnapshot 对每个方言把整条语料的解析结果 dump 成规范化文本并与基线比对。
func TestParserSnapshot(t *testing.T) {
	for dialect, parser := range allParsers() {
		dialect, parser := dialect, parser
		t.Run(dialect, func(t *testing.T) {
			t.Parallel()
			var got strings.Builder
			for _, c := range snapshotCorpus {
				fmt.Fprintf(&got, "### %s\n%s\n", c.id, c.sql)
				stmt, err := parser.Parse(c.sql)
				if err != nil {
					fmt.Fprintf(&got, "  error=%v\n\n", err)
					continue
				}
				if stmt == nil {
					fmt.Fprintf(&got, "  <nil stmt>\n\n")
					continue
				}
				dumpStmt(&got, stmt, "  ", 0)
				got.WriteString("\n")
			}
			assertGolden(t, dialect, got.String())
		})
	}
}

// TestParserSnapshotNoSilentLoss 断言每个方言对其**原生支持**的形态必须消费完整条语句。
//
// 静默丢段的共同特征都是 base.Text 短于输入（解析器提前停在某个子句关键字上）。
// 结构断言只覆盖被想到的字段，而这条不变量对所有子句一视同仁。
func TestParserSnapshotNoSilentLoss(t *testing.T) {
	for dialect, parser := range allParsers() {
		dialect, parser := dialect, parser
		t.Run(dialect, func(t *testing.T) {
			t.Parallel()
			for _, c := range snapshotCorpus {
				if !contains(scopeOf(c.id), dialect) {
					continue // 非该方言原生形态，降级结果只进快照不做断言
				}
				stmt, err := parser.Parse(c.sql)
				if err != nil || stmt == nil {
					t.Errorf("[%s/%s] 解析失败: err=%v stmt=%v", dialect, c.id, err, stmt)
					continue
				}
				base := normText(stmt.GetText())
				want := strings.TrimSuffix(normText(c.sql), ";")
				if base != want {
					t.Errorf("[%s/%s] 语句未被完整消费（存在静默丢段）\n got=%q\nwant=%q", dialect, c.id, base, want)
				}
				// 文本完整并不代表结构建出来了：语句在此停下却一个对象都没登记时，
				// 文本照样「看起来完整」（TextFrom 含当前未消费的那个 token），
				// 而下游拿不到表/条件就是静默失效（脱敏漏遮列、准入按无表放行）
				assertMinimumStructure(t, dialect, c.sql, stmt)
			}
		})
	}
}

// assertMinimumStructure 按语句类型断言「解析出了最小结构」。
//
// 专门对付「边界词被吃成别名/表名，导致后继子句整段不被登记」这类形态：
// 它不检查具体名字对不对（那是快照与形态断言的职责），只保证不会交出空壳结果
func assertMinimumStructure(t *testing.T, dialect, sql string, stmt sqlstmt.Stmt) {
	t.Helper()
	upper := outerSqlWords(sql)
	switch v := stmt.(type) {
	case *sqlstmt.SelectStmt:
		if strings.Contains(upper, " FROM ") && len(v.From)+len(v.Joins) == 0 {
			t.Errorf("[%s] 含 FROM 却未登记任何表引用（边界词被当成别名/表名？）: %s", dialect, sql)
		}
		if strings.Contains(upper, " WHERE ") && v.Where == nil {
			t.Errorf("[%s] 含 WHERE 却未建出条件: %s", dialect, sql)
		}
		if hasSetOperator(upper) && len(v.Unions) == 0 && !strings.Contains(upper, " MINUS ") {
			t.Errorf("[%s] 含集合操作却未登记分支（操作词被吃成别名？）: %s", dialect, sql)
		}
	case *sqlstmt.UpdateStmt:
		if len(v.Tables) == 0 {
			t.Errorf("[%s] UPDATE 未登记目标表: %s", dialect, sql)
		}
		if strings.Contains(upper, " WHERE ") && v.Where == nil {
			t.Errorf("[%s] UPDATE 含 WHERE 却未建出条件: %s", dialect, sql)
		}
		if strings.Contains(upper, " SET ") && len(v.Set) == 0 {
			t.Errorf("[%s] UPDATE 含 SET 却未解析出赋值: %s", dialect, sql)
		}
	case *sqlstmt.DeleteStmt:
		if len(v.Tables) == 0 {
			t.Errorf("[%s] DELETE 未登记目标表: %s", dialect, sql)
		}
		if strings.Contains(upper, " WHERE ") && v.Where == nil {
			t.Errorf("[%s] DELETE 含 WHERE 却未建出条件: %s", dialect, sql)
		}
	case *sqlstmt.InsertStmt:
		// INSERT ALL 是多表插入，按设计不登记单一目标表；MINUS 在非 Oracle 系方言是裸标识符，
		// 由各自的适用面与快照负责，不在此强求集合操作结构
		if !strings.HasPrefix(upper, "INSERT ALL") && v.Table.Name == "" {
			t.Errorf("[%s] INSERT 未登记目标表: %s", dialect, sql)
		}
	}
}

// hasSetOperator 外层是否出现跨方言通用的集合操作词
// （MINUS 只在 Oracle 系是保留字，其余方言它是裸标识符，故不参与本判定）
func hasSetOperator(upper string) bool {
	return strings.Contains(upper, " UNION ") || strings.Contains(upper, " INTERSECT ") || strings.Contains(upper, " EXCEPT ")
}

// outerSqlWords 取语句的「外层文本」：剥掉括号内的内容与字符串字面量，折叠空白并转大写。
//
// 结构最小要求必须只看外层：`SELECT x.id FROM (SELECT id FROM t1 WHERE id > 0) x`
// 的内层有 WHERE、外层没有，按全文本判断会把内层的关键字算到外层头上而误报
func outerSqlWords(sql string) string {
	var b strings.Builder
	depth, inStr := 0, false
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if inStr {
			if ch == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i++ // '' 转义
					continue
				}
				inStr = false
			}
			continue
		}
		switch {
		case ch == '\'':
			inStr = true
		case ch == '(':
			depth++
		case ch == ')':
			if depth > 0 {
				depth--
			}
		case depth == 0:
			b.WriteByte(ch)
		}
	}
	return strings.ToUpper(normText(b.String())) + " "
}

// TestParserSnapshotDialectParity 同一语料在其适用方言间的结构一致性：
// 核心结构（项数/表数/JOIN 数及其类型/有无 WHERE/UNION 数）必须相同。
// 某方言比同类少解析一段（如曾因缺 NATURAL 分支丢掉整个 JOIN）正是本测试要抓的目标。
func TestParserSnapshotDialectParity(t *testing.T) {
	parsers := allParsers()
	for _, c := range snapshotCorpus {
		c := c
		scope := scopeOf(c.id)
		if len(scope) < 2 {
			continue // 单一方言原生形态，无对照方
		}
		t.Run(c.id, func(t *testing.T) {
			var refShape *shape
			var refDialect string
			for _, d := range scope {
				parser, ok := parsers[d]
				if !ok {
					continue // 该方言不在本表的比对集合内（薄方言由 parser_thin_dialects_test.go 覆盖）
				}
				stmt, err := parser.Parse(c.sql)
				if err != nil {
					t.Errorf("[%s] 解析失败: %v", d, err)
					continue
				}
				got := shapeOf(stmt)
				if refShape == nil {
					refShape, refDialect = got, d
					continue
				}
				if *got != *refShape {
					t.Errorf("[%s] 结构与 [%s] 不一致（该方言少解析了一段？）\n got=%+v\nref=%+v", d, refDialect, *got, *refShape)
				}
			}
		})
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// shape 跨方言可比的核心结构指纹
type shape struct {
	kind      string
	tables    string // FROM/JOIN/目标表的 name|schema|alias 摘要：表层错建模必须可见
	items     int
	from      int
	joins     int
	joinKinds string
	where     bool
	groupBy   int
	having    bool
	orderBy   int
	unions    int
	limit     bool
	distinct  bool
	hasTree   bool // WHERE 是否建成结构化树（而非降级原文）
}

func shapeOf(stmt sqlstmt.Stmt) *shape {
	s := &shape{kind: fmt.Sprintf("%T", stmt)}
	switch v := stmt.(type) {
	case *sqlstmt.SelectStmt:
		s.items, s.from, s.joins = len(v.Items), len(v.From), len(v.Joins)
		s.tables = tableRefsText(v.From)
		s.where, s.groupBy, s.having = v.Where != nil, len(v.GroupBy), v.Having != nil
		s.orderBy, s.unions, s.limit, s.distinct = len(v.OrderBy), len(v.Unions), v.Limit != nil, v.Distinct
		s.hasTree = v.Where != nil && v.Where.Root != nil
		kinds := make([]string, 0, len(v.Joins))
		for _, j := range v.Joins {
			kinds = append(kinds, joinKindName(j.Kind))
		}
		s.joinKinds = strings.Join(kinds, ",")
	case *sqlstmt.UpdateStmt:
		s.from, s.where, s.limit = len(v.Tables), v.Where != nil, v.Limit != nil
		s.tables = tableRefsText(v.Tables)
		s.items = len(v.Set)
		s.hasTree = v.Where != nil && v.Where.Root != nil
	case *sqlstmt.DeleteStmt:
		s.from, s.where, s.limit = len(v.Tables), v.Where != nil, v.Limit != nil
		s.tables = tableRefsText(v.Tables)
		s.hasTree = v.Where != nil && v.Where.Root != nil
	case *sqlstmt.InsertStmt:
		s.from, s.items = 1, len(v.Columns)
	}
	return s
}

// tableRefsText 表引用摘要（name|schema|alias 逐项拼接，顺序敏感）
func tableRefsText(refs []sqlstmt.TableRef) string {
	parts := make([]string, 0, len(refs))
	for _, r := range refs {
		parts = append(parts, r.Name+"|"+r.Schema+"|"+r.Alias)
	}
	return strings.Join(parts, ",")
}

func joinKindName(k sqlstmt.JoinKind) string {
	switch k {
	case sqlstmt.JoinKindInner:
		return "inner"
	case sqlstmt.JoinKindLeft:
		return "left"
	case sqlstmt.JoinKindRight:
		return "right"
	case sqlstmt.JoinKindFull:
		return "full"
	case sqlstmt.JoinKindCross:
		return "cross"
	case sqlstmt.JoinKindNatural:
		return "natural"
	}
	return fmt.Sprintf("kind(%d)", int(k))
}

// maxDumpDepth 子查询递归深度上限（防御自引用深树导致快照爆炸）
const maxDumpDepth = 4

func dumpStmt(b *strings.Builder, stmt sqlstmt.Stmt, indent string, depth int) {
	if stmt == nil {
		b.WriteString("<nil>")
		return
	}
	fmt.Fprintf(b, "%s= %T\n", indent, stmt)
	fmt.Fprintf(b, "%stext=%q\n", indent, normText(stmt.GetText()))
	if depth >= maxDumpDepth {
		b.WriteString(indent + "  (depth cap)\n")
		return
	}
	switch v := stmt.(type) {
	case *sqlstmt.SelectStmt:
		fmt.Fprintf(b, "%sdistinct=%v\n", indent, v.Distinct)
		for i, it := range v.Items {
			fmt.Fprintf(b, "%sitem[%d] kind=%d text=%q col=%q alias=%q tbl=%q\n",
				indent, i, it.Kind, normText(it.Text), normText(it.ColumnName), normText(it.Alias), normText(it.TableAlias))
		}
		for i, f := range v.From {
			dumpTableRef(b, indent, fmt.Sprintf("from[%d]", i), f)
		}
		for i, j := range v.Joins {
			fmt.Fprintf(b, "%sjoin[%d] kind=%s text=%q\n", indent, i, joinKindName(j.Kind), normText(j.Text))
			dumpTableRef(b, indent+"  ", "table", j.Table)
			dumpExpr(b, indent+"  ", "on", j.On, depth+1)
		}
		dumpExpr(b, indent, "where", v.Where, depth+1)
		for i, g := range v.GroupBy {
			fmt.Fprintf(b, "%sgroupBy[%d]=%q\n", indent, i, normText(g))
		}
		dumpExpr(b, indent, "having", v.Having, depth+1)
		for i, o := range v.OrderBy {
			fmt.Fprintf(b, "%sorderBy[%d] text=%q desc=%v\n", indent, i, normText(o.Text), o.Desc)
		}
		dumpLimit(b, indent, v.Limit)
		for i, u := range v.Unions {
			fmt.Fprintf(b, "%sunion[%d] op=%s all=%v text=%q\n", indent, i, u.Op, u.All, normText(u.Text))
			if u.Select != nil {
				dumpStmt(b, u.Select, indent+"  ", depth+1)
			}
		}
	case *sqlstmt.InsertStmt:
		dumpTableRef(b, indent, "table", v.Table)
		for i, c := range v.Columns {
			fmt.Fprintf(b, "%scolumn[%d]=%q\n", indent, i, normText(c))
		}
		for i, row := range v.ValueRows {
			cells := make([]string, 0, len(row))
			for _, cell := range row {
				if cell.Null {
					cells = append(cells, "NULL")
					continue
				}
				cells = append(cells, fmt.Sprintf("%q", normText(cell.Text)))
			}
			fmt.Fprintf(b, "%srow[%d]=[%s]\n", indent, i, strings.Join(cells, ", "))
		}
		if v.ValueRows == nil {
			b.WriteString(indent + "rows=<not-plain-literals>\n")
		}
		fmt.Fprintf(b, "%sconflict=%v\n", indent, v.HasConflictClause)
	case *sqlstmt.UpdateStmt:
		for i, tb := range v.Tables {
			dumpTableRef(b, indent, fmt.Sprintf("table[%d]", i), tb)
		}
		for i, a := range v.Set {
			fmt.Fprintf(b, "%sset[%d] col=%q valueText=%q assignText=%q\n",
				indent, i, normText(a.Column), normText(textOf(a.Value)), normText(a.Text))
		}
		dumpExpr(b, indent, "where", v.Where, depth+1)
		dumpLimit(b, indent, v.Limit)
	case *sqlstmt.DeleteStmt:
		for i, tb := range v.Tables {
			dumpTableRef(b, indent, fmt.Sprintf("table[%d]", i), tb)
		}
		dumpExpr(b, indent, "where", v.Where, depth+1)
		dumpLimit(b, indent, v.Limit)
	case *sqlstmt.DdlStmt:
		fmt.Fprintf(b, "%sddlKind=%q\n", indent, v.DdlKind)
	}
}

func dumpTableRef(b *strings.Builder, indent, label string, ref sqlstmt.TableRef) {
	fmt.Fprintf(b, "%s%s schema=%q name=%q alias=%q\n",
		indent, label, normText(ref.Schema), normText(ref.Name), normText(ref.Alias))
}

func dumpLimit(b *strings.Builder, indent string, l *sqlstmt.Limit) {
	if l == nil {
		fmt.Fprintf(b, "%slimit=<nil>\n", indent)
		return
	}
	fmt.Fprintf(b, "%slimit count=%d offset=%d text=%q\n", indent, l.Count, l.Offset, normText(l.Text))
}

func dumpExpr(b *strings.Builder, indent, label string, e *sqlstmt.Expr, depth int) {
	if e == nil {
		fmt.Fprintf(b, "%s%s=<nil>\n", indent, label)
		return
	}
	if e.Root == nil {
		fmt.Fprintf(b, "%s%s text=%q tree=<degraded>\n", indent, label, normText(e.Text))
		return
	}
	fmt.Fprintf(b, "%s%s text=%q tree=%s\n", indent, label, normText(e.Text), normText(e.Root.String()))
	dumpSubSelects(b, e.Root, indent+"  ", depth)
}

// dumpSubSelects 递归暴露表达式树里挂着的子查询（String() 只打 "(SUBQUERY)" 会藏掉内容）
func dumpSubSelects(b *strings.Builder, node sqlstmt.ExprNode, indent string, depth int) {
	if node == nil || depth >= maxDumpDepth {
		return
	}
	switch v := node.(type) {
	case *sqlstmt.InExpr:
		dumpSubSelects(b, v.Operand, indent, depth)
		for _, e := range v.List {
			dumpSubSelects(b, e, indent, depth)
		}
		if v.Subquery != nil {
			fmt.Fprintf(b, "%sinSubquery:\n", indent)
			dumpStmt(b, v.Subquery, indent+"  ", depth+1)
		}
	case *sqlstmt.ExistsExpr:
		if v.Subquery != nil {
			fmt.Fprintf(b, "%sexistsSubquery:\n", indent)
			dumpStmt(b, v.Subquery, indent+"  ", depth+1)
		}
	case *sqlstmt.SubQueryExpr:
		if v.Select != nil {
			fmt.Fprintf(b, "%sscalarSubquery:\n", indent)
			dumpStmt(b, v.Select, indent+"  ", depth+1)
		} else {
			fmt.Fprintf(b, "%sscalarSubquery degraded=%q\n", indent, normText(v.Text()))
		}
	case *sqlstmt.BinaryExpr:
		dumpSubSelects(b, v.Left, indent, depth)
		dumpSubSelects(b, v.Right, indent, depth)
	case *sqlstmt.UnaryExpr:
		dumpSubSelects(b, v.Operand, indent, depth)
	case *sqlstmt.ParenExpr:
		dumpSubSelects(b, v.Inner, indent, depth)
	case *sqlstmt.FuncCall:
		for _, a := range v.Args {
			dumpSubSelects(b, a, indent, depth)
		}
	case *sqlstmt.CaseExpr:
		dumpSubSelects(b, v.Operand, indent, depth)
		for _, w := range v.Whens {
			dumpSubSelects(b, w.When, indent, depth)
			dumpSubSelects(b, w.Then, indent, depth)
		}
		dumpSubSelects(b, v.ElseExpr, indent, depth)
	case *sqlstmt.BetweenExpr:
		dumpSubSelects(b, v.Operand, indent, depth)
		dumpSubSelects(b, v.Low, indent, depth)
		dumpSubSelects(b, v.High, indent, depth)
	case *sqlstmt.LikeExpr:
		dumpSubSelects(b, v.Operand, indent, depth)
		dumpSubSelects(b, v.Pattern, indent, depth)
		dumpSubSelects(b, v.Escape, indent, depth)
	case *sqlstmt.IsNullExpr:
		dumpSubSelects(b, v.Operand, indent, depth)
	}
}

func textOf(e *sqlstmt.Expr) string {
	if e == nil {
		return ""
	}
	return e.Text
}

// normText 折叠连续空白并去首尾空白：让快照只反映内容差异，不反映排版差异
func normText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// assertGolden 与基线比对；-update 时写回基线
func assertGolden(t *testing.T, dialect, got string) {
	t.Helper()
	path := filepath.Join(snapshotDir, dialect+".golden")
	if *updateSnapshot {
		if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
			t.Fatalf("创建快照目录失败: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("写入快照基线失败: %v", err)
		}
		t.Logf("已更新基线: %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取快照基线失败（首次请用 -update 生成）: %v", err)
	}
	if string(want) != got {
		gotLines, wantLines := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		var diffs []string
		for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
			var w, g string
			if i < len(wantLines) {
				w = wantLines[i]
			}
			if i < len(gotLines) {
				g = gotLines[i]
			}
			if w != g {
				diffs = append(diffs, fmt.Sprintf("  line %d:\n- %s\n+ %s", i+1, w, g))
			}
			if len(diffs) >= 20 {
				diffs = append(diffs, "  ...（更多 diff 被截断）")
				break
			}
		}
		t.Errorf("解析行为与基线不一致 (%s):\n%s", path, strings.Join(diffs, "\n"))
	}
}
