package sqlparser

// 跨方言复杂SQL解析进阶矩阵：窗口函数/递归CTE/集合操作/注释与字符串干扰等
// 高频易错形态。各方言解析出的核心结构必须一致，保证脱敏（items计数）与
// 数据同步校验（GroupBy/Having/OrderBy识别）在这些形态下同样准确
import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// TestWindowFunctionParse 窗口函数：OVER内逗号/关键字不得撑爆items计数
func TestWindowFunctionParse(t *testing.T) {
	cases := []struct {
		name  string
		sql   string
		items int
		from  int
	}{
		{
			name:  "ROW_NUMBER分区排序",
			sql:   "SELECT name, ROW_NUMBER() OVER (PARTITION BY dept_id ORDER BY salary DESC) AS rn FROM employees",
			items: 2, from: 1,
		},
		{
			name:  "窗口函数多参数",
			sql:   "SELECT dept, SUM(amount, 1) OVER (ORDER BY id ROWS BETWEEN 1 PRECEDING AND CURRENT ROW) FROM t",
			items: 2, from: 1,
		},
		{
			name:  "窗口函数与聚合混合",
			sql:   "SELECT COUNT(*), RANK() OVER (PARTITION BY a, b ORDER BY c) FROM orders GROUP BY a",
			items: 2, from: 1,
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
				t.Errorf("[%s] %s: 期望SelectStmt，得到%T", name, c.name, stmt)
				continue
			}
			if len(sel.Items) != c.items {
				t.Errorf("[%s] %s: items=%d, 期望%d (%+v)", name, c.name, len(sel.Items), c.items, sel.Items)
			}
			if len(sel.From) != c.from {
				t.Errorf("[%s] %s: from=%d, 期望%d", name, c.name, len(sel.From), c.from)
			}
		}
	}
}

// TestRecursiveCteParse 递归CTE各方言识别为KindWith
func TestRecursiveCteParse(t *testing.T) {
	sql := "WITH RECURSIVE sub AS (SELECT id, pid FROM tree WHERE id = 1 UNION ALL SELECT t.id, t.pid FROM tree t JOIN sub s ON t.pid = s.id) SELECT * FROM sub"
	for name, p := range allParsers() {
		stmt, err := p.Parse(sql)
		if err != nil {
			t.Errorf("[%s] 递归CTE解析失败: %v", name, err)
			continue
		}
		if stmt.StmtKind() != sqlstmt.KindWith {
			t.Errorf("[%s] 递归CTE期望KindWith，得到%s", name, stmt.StmtKind())
		}
	}
}

// TestSetOperationParse INTERSECT/EXCEPT集合操作（oracle为MINUS，单独断言）
func TestSetOperationParse(t *testing.T) {
	for name, p := range allParsers() {
		if name == "oracle" {
			continue
		}
		stmt, err := p.Parse("SELECT id FROM a INTERSECT SELECT id FROM b EXCEPT SELECT id FROM c")
		if err != nil {
			t.Errorf("[%s] INTERSECT/EXCEPT解析失败: %v", name, err)
			continue
		}
		sel, ok := stmt.(*sqlstmt.SelectStmt)
		if !ok {
			t.Errorf("[%s] 期望SelectStmt，得到%T", name, stmt)
			continue
		}
		// 主查询+2段集合操作
		if len(sel.Unions) != 2 {
			t.Errorf("[%s] unions=%d, 期望2", name, len(sel.Unions))
		}
	}
	stmt, err := allParsers()["oracle"].Parse("SELECT id FROM a MINUS SELECT id FROM b")
	if err != nil {
		t.Fatalf("[oracle] MINUS解析失败: %v", err)
	}
	if sel, ok := stmt.(*sqlstmt.SelectStmt); !ok || len(sel.Unions) != 1 {
		t.Errorf("[oracle] MINUS应解析出1段集合操作: %T %+v", stmt, stmt)
	}
}

// TestCommentAndStringInterference 注释与字符串内的干扰内容不得破坏解析
func TestCommentAndStringInterference(t *testing.T) {
	cases := []struct {
		name  string
		sql   string
		items int
		where bool
	}{
		{
			name:  "行注释含引号与逗号",
			sql:   "SELECT a, -- 注释: don't, split, here\n b FROM t WHERE x = 1",
			items: 2, where: true,
		},
		{
			name:  "块注释含关键字",
			sql:   "SELECT /* SELECT , FROM ( WHERE */ a, b FROM t",
			items: 2, where: false,
		},
		{
			name:  "字符串内逗号与括号",
			sql:   "SELECT 'a, b (c)', status FROM t WHERE msg = 'where (x = 1)'",
			items: 2, where: true,
		},
		{
			name:  "转义引号字符串",
			sql:   "SELECT 'it''s ok', name FROM t WHERE title = 'don''t stop'",
			items: 2, where: true,
		},
		{
			name:  "注释夹在FROM与表之间",
			sql:   "SELECT id FROM /* log */ users WHERE id > 0",
			items: 1, where: true,
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
				t.Errorf("[%s] %s: 期望SelectStmt，得到%T", name, c.name, stmt)
				continue
			}
			if len(sel.Items) != c.items {
				t.Errorf("[%s] %s: items=%d, 期望%d (%+v)", name, c.name, len(sel.Items), c.items, sel.Items)
			}
			if (sel.Where != nil) != c.where {
				t.Errorf("[%s] %s: where存在性=%v, 期望%v", name, c.name, sel.Where != nil, c.where)
			}
		}
	}
}

// TestDeepNestedSubqueryParse 多层嵌套子查询：括号深度不应干扰结构
func TestDeepNestedSubqueryParse(t *testing.T) {
	sql := "SELECT name FROM users WHERE id IN (SELECT uid FROM orders WHERE pid IN (SELECT id FROM products WHERE cid IN (SELECT id FROM categories WHERE level > (SELECT MIN(level) FROM categories))))"
	for name, p := range allParsers() {
		stmt, err := p.Parse(sql)
		if err != nil {
			t.Errorf("[%s] 多层嵌套解析失败: %v", name, err)
			continue
		}
		sel, ok := stmt.(*sqlstmt.SelectStmt)
		if !ok {
			t.Errorf("[%s] 期望SelectStmt，得到%T", name, stmt)
			continue
		}
		if len(sel.Items) != 1 || len(sel.From) != 1 || sel.Where == nil {
			t.Errorf("[%s] 结构错误: items=%d from=%d where=%v", name, len(sel.Items), len(sel.From), sel.Where != nil)
		}
	}
}

// TestFunctionExprParse 函数表达式易错形态：嵌套函数逗号/BETWEEN AND/无参函数
func TestFunctionExprParse(t *testing.T) {
	cases := []struct {
		name  string
		sql   string
		items int
	}{
		{
			name:  "嵌套函数多参数",
			sql:   "SELECT CONCAT(a, IFNULL(b, 'x'), UPPER(c)) AS full, id FROM t",
			items: 2,
		},
		{
			name:  "无参函数",
			sql:   "SELECT NOW(), UUID(), id FROM t",
			items: 3,
		},
		{
			name:  "BETWEEN AND不算逗号分隔",
			sql:   "SELECT id, amount FROM orders WHERE created BETWEEN '2024-01-01 00:00:00' AND '2024-12-31 23:59:59'",
			items: 2,
		},
		{
			name:  "函数内IN子查询",
			sql:   "SELECT COUNT(*) FROM t WHERE id IN (SELECT id FROM u)",
			items: 1,
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
				t.Errorf("[%s] %s: 期望SelectStmt，得到%T", name, c.name, stmt)
				continue
			}
			if len(sel.Items) != c.items {
				t.Errorf("[%s] %s: items=%d, 期望%d (%+v)", name, c.name, len(sel.Items), c.items, sel.Items)
			}
		}
	}
}

// TestOrderSpecialClauses ORDER BY特殊形态与NULLS FIRST/LAST
func TestOrderSpecialClauses(t *testing.T) {
	cases := []string{
		"SELECT id FROM t ORDER BY a NULLS FIRST, b DESC NULLS LAST",
		"SELECT id FROM t ORDER BY 1, 2",
		"SELECT id FROM t ORDER BY LENGTH(name), id",
	}
	for _, sql := range cases {
		for name, p := range allParsers() {
			stmt, err := p.Parse(sql)
			if err != nil {
				t.Errorf("[%s] %q 解析失败: %v", name, sql, err)
				continue
			}
			sel, ok := stmt.(*sqlstmt.SelectStmt)
			if !ok {
				t.Errorf("[%s] %q 期望SelectStmt，得到%T", name, sql, stmt)
				continue
			}
			if len(sel.OrderBy) == 0 {
				t.Errorf("[%s] %q ORDER BY应被识别", name, sql)
			}
		}
	}
}

// TestGroupByHavingComplex 复杂GROUP BY/HAVING（同步校验依赖其识别准确性）
func TestGroupByHavingComplex(t *testing.T) {
	sql := "SELECT dept, COUNT(*) AS cnt FROM emp GROUP BY dept HAVING COUNT(*) > 5 AND MAX(salary) < 100000 ORDER BY cnt DESC LIMIT 10"
	for name, p := range allParsers() {
		stmt, err := p.Parse(sql)
		if err != nil {
			t.Errorf("[%s] 解析失败: %v", name, err)
			continue
		}
		sel, ok := stmt.(*sqlstmt.SelectStmt)
		if !ok {
			t.Errorf("[%s] 期望SelectStmt，得到%T", name, stmt)
			continue
		}
		if len(sel.GroupBy) != 1 {
			t.Errorf("[%s] groupBy=%v, 期望1项", name, sel.GroupBy)
		}
		if sel.Having == nil {
			t.Errorf("[%s] HAVING应被识别", name)
		}
		if sel.Limit == nil && name != "oracle" {
			t.Errorf("[%s] LIMIT应被识别", name)
		}
	}
}
