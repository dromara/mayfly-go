package sqlparser

// 表达式AST矩阵测试：WHERE/HAVING/ON条件经Pratt解析器建树后的
// 结构正确性（优先级/结合性/各类谓词）与跨方言一致性。
// 建树失败必须降级为纯文本（Text非空），不允许中断解析或丢文本

import (
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// whereRootOf 解析SELECT的WHERE并返回表达式树根（要求建树成功）
func whereRootOf(t *testing.T, dialect, where string) sqlstmt.ExprNode {
	t.Helper()
	stmt, err := allParsers()[dialect].Parse("SELECT id FROM t WHERE " + where)
	if err != nil {
		t.Fatalf("[%s] 解析失败 %q: %v", dialect, where, err)
	}
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok || sel.Where == nil {
		t.Fatalf("[%s] WHERE未识别: %T", dialect, stmt)
	}
	if sel.Where.Root == nil {
		t.Fatalf("[%s] WHERE表达式未建树（Root=nil）: %q", dialect, where)
	}
	return sel.Where.Root
}

// TestExprPrecedence 优先级：AND绑定紧于OR，算术紧于比较
func TestExprPrecedence(t *testing.T) {
	// a=1 OR (b=2 AND c=3)：AND子树必须是OR的右操作数
	root := whereRootOf(t, "mysql", "a = 1 OR b = 2 AND c = 3")
	or, ok := root.(*sqlstmt.BinaryExpr)
	if !ok || or.Op != "OR" {
		t.Fatalf("顶层应为OR，得到%T %+v", root, root)
	}
	and, ok := or.Right.(*sqlstmt.BinaryExpr)
	if !ok || and.Op != "AND" {
		t.Fatalf("OR右侧应为AND子树（AND优先级高于OR），得到%T %+v", or.Right, or.Right)
	}

	// (a = 1 OR b = 2) AND c = 3：括号覆盖优先级
	root = whereRootOf(t, "mysql", "(a = 1 OR b = 2) AND c = 3")
	and, ok = root.(*sqlstmt.BinaryExpr)
	if !ok || and.Op != "AND" {
		t.Fatalf("顶层应为AND，得到%T", root)
	}
	if _, ok := and.Left.(*sqlstmt.ParenExpr); !ok {
		t.Fatalf("AND左侧应为括号表达式，得到%T", and.Left)
	}

	// a + b * c > 10：* 优先于 +
	root = whereRootOf(t, "mysql", "a + b * c > 10")
	gt, ok := root.(*sqlstmt.BinaryExpr)
	if !ok || gt.Op != ">" {
		t.Fatalf("顶层应为>，得到%T", root)
	}
	plus, ok := gt.Left.(*sqlstmt.BinaryExpr)
	if !ok || plus.Op != "+" {
		t.Fatalf("比较左侧应为+，得到%T", gt.Left)
	}
	if mul, ok := plus.Right.(*sqlstmt.BinaryExpr); !ok || mul.Op != "*" {
		t.Fatalf("+右侧应为*子树（*优先于+），得到%T", plus.Right)
	}
}

// TestExprAssociativity 结合性：同优先级左结合
func TestExprAssociativity(t *testing.T) {
	root := whereRootOf(t, "mysql", "a - b - c = 0")
	eq, ok := root.(*sqlstmt.BinaryExpr)
	if !ok || eq.Op != "=" {
		t.Fatalf("顶层应为=，得到%T", root)
	}
	sub1, ok := eq.Left.(*sqlstmt.BinaryExpr)
	if !ok || sub1.Op != "-" {
		t.Fatalf("左侧应为第一层-，得到%T", eq.Left)
	}
	sub2, ok := sub1.Left.(*sqlstmt.BinaryExpr)
	if !ok || sub2.Op != "-" {
		t.Fatalf("第二层-（左结合：(a-b)-c 而非 a-(b-c)），得到%T", sub1.Left)
	}
	if _, ok := sub2.Left.(*sqlstmt.ColumnRef); !ok {
		t.Fatalf("最左操作数应为列a，得到%T", sub2.Left)
	}
}

// TestExprPredicates 各类谓词的树形结构
func TestExprPredicates(t *testing.T) {
	cases := []struct {
		name  string
		where string
		check func(*testing.T, sqlstmt.ExprNode)
	}{
		{"IN列表", "a IN (1, 2, 3)", func(t *testing.T, n sqlstmt.ExprNode) {
			in, ok := n.(*sqlstmt.InExpr)
			if !ok || len(in.List) != 3 {
				t.Fatalf("期望InExpr(3项)，得到%T %+v", n, n)
			}
		}},
		{"NOT IN", "a NOT IN (1, 2)", func(t *testing.T, n sqlstmt.ExprNode) {
			in, ok := n.(*sqlstmt.InExpr)
			if !ok || !in.Not {
				t.Fatalf("期望NOT InExpr，得到%T", n)
			}
		}},
		{"BETWEEN", "a BETWEEN 1 AND 10", func(t *testing.T, n sqlstmt.ExprNode) {
			be, ok := n.(*sqlstmt.BetweenExpr)
			if !ok || be.Low == nil || be.High == nil {
				t.Fatalf("期望BetweenExpr，得到%T %+v", n, n)
			}
		}},
		{"LIKE", "name LIKE 'a%'", func(t *testing.T, n sqlstmt.ExprNode) {
			if _, ok := n.(*sqlstmt.LikeExpr); !ok {
				t.Fatalf("期望LikeExpr，得到%T", n)
			}
		}},
		{"IS NULL", "a IS NULL OR b IS NOT NULL", func(t *testing.T, n sqlstmt.ExprNode) {
			or, ok := n.(*sqlstmt.BinaryExpr)
			if !ok || or.Op != "OR" {
				t.Fatalf("期望OR，得到%T", n)
			}
			if _, ok := or.Left.(*sqlstmt.IsNullExpr); !ok {
				t.Fatalf("OR左侧应为IsNullExpr，得到%T", or.Left)
			}
			isn, ok := or.Right.(*sqlstmt.IsNullExpr)
			if !ok || !isn.Not {
				t.Fatalf("OR右侧应为IS NOT NULL，得到%T", or.Right)
			}
		}},
		{"NOT一元", "NOT (a = 1 AND b = 2)", func(t *testing.T, n sqlstmt.ExprNode) {
			un, ok := n.(*sqlstmt.UnaryExpr)
			if !ok || un.Op != "NOT" {
				t.Fatalf("期望UnaryExpr(NOT)，得到%T", n)
			}
		}},
		{"标量子查询", "a = (SELECT MAX(x) FROM tx)", func(t *testing.T, n sqlstmt.ExprNode) {
			eq, ok := n.(*sqlstmt.BinaryExpr)
			if !ok {
				t.Fatalf("期望BinaryExpr，得到%T", n)
			}
			sq, ok := eq.Right.(*sqlstmt.SubQueryExpr)
			if !ok || sq.Select == nil {
				t.Fatalf("右侧应为带SelectStmt树的子查询，得到%T", eq.Right)
			}
		}},
		{"CASE WHEN", "CASE WHEN a = 1 THEN 'x' ELSE 'y' END = 'x'", func(t *testing.T, n sqlstmt.ExprNode) {
			eq, ok := n.(*sqlstmt.BinaryExpr)
			if !ok {
				t.Fatalf("期望BinaryExpr，得到%T", n)
			}
			ce, ok := eq.Left.(*sqlstmt.CaseExpr)
			if !ok || len(ce.Whens) != 1 || ce.ElseExpr == nil {
				t.Fatalf("左侧应为CaseExpr(1 WHEN+ELSE)，得到%T", eq.Left)
			}
		}},
		{"EXISTS", "EXISTS (SELECT 1 FROM tx WHERE tx.a = t.a)", func(t *testing.T, n sqlstmt.ExprNode) {
			ex, ok := n.(*sqlstmt.ExistsExpr)
			if !ok || ex.Subquery == nil {
				t.Fatalf("期望ExistsExpr(带Subquery)，得到%T", n)
			}
		}},
		{"函数调用", "COALESCE(a, b) = 1 AND IFNULL(c, 0) > 0", func(t *testing.T, n sqlstmt.ExprNode) {
			and, ok := n.(*sqlstmt.BinaryExpr)
			if !ok || and.Op != "AND" {
				t.Fatalf("期望AND，得到%T", n)
			}
			eq, ok := and.Left.(*sqlstmt.BinaryExpr)
			if !ok || eq.Op != "=" {
				t.Fatalf("左侧应为COALESCE(...) = 1比较，得到%T", and.Left)
			}
			fc, ok := eq.Left.(*sqlstmt.FuncCall)
			if !ok || fc.Name != "COALESCE" || len(fc.Args) != 2 {
				t.Fatalf("比较左侧应为COALESCE(2参数)，得到%T", eq.Left)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := whereRootOf(t, "mysql", c.where)
			c.check(t, root)
		})
	}
}

// TestExprCrossDialectConsistency 同一条件SQL各方言建树后序列化必须一致
// （解析器方言差异不得影响条件树的结构语义）
func TestExprCrossDialectConsistency(t *testing.T) {
	wheres := []string{
		"a = 1 OR b = 2 AND c = 3",
		"a IN (1, 2, 3) AND b LIKE 'x%'",
		"a BETWEEN 1 AND 10 OR b IS NULL",
		"NOT (a > 5 OR b <= 3)",
		"a + b * 2 = 10",
		"COALESCE(a, b) = 1",
		"a = (SELECT MAX(x) FROM tx)",
	}
	parsers := allParsers()
	for _, where := range wheres {
		want := ""
		for name, p := range parsers {
			stmt, err := p.Parse("SELECT id FROM t WHERE " + where)
			if err != nil {
				t.Errorf("[%s] %q 解析失败: %v", name, where, err)
				continue
			}
			sel, ok := stmt.(*sqlstmt.SelectStmt)
			if !ok || sel.Where == nil || sel.Where.Root == nil {
				t.Errorf("[%s] %q WHERE未建树", name, where)
				continue
			}
			got := sel.Where.Root.String()
			if want == "" {
				want = got
			} else if got != want {
				t.Errorf("%q 树形不一致: [%s]=%s != %s", where, name, got, want)
			}
		}
	}
}

// TestExprFallbackDegradation 建树失败必须降级：Text完整保留、不panic不中断
func TestExprFallbackDegradation(t *testing.T) {
	for name, p := range allParsers() {
		// 方言特有函数（各方言不认识的形态彼此不同，只要求任何方言下Text完整）
		for _, sql := range []string{
			"SELECT id FROM t WHERE a &&& b",
			"SELECT id FROM t WHERE a = ??",
		} {
			stmt, err := p.Parse(sql)
			if err != nil {
				t.Errorf("[%s] %q 不应解析失败: %v", name, sql, err)
				continue
			}
			sel, ok := stmt.(*sqlstmt.SelectStmt)
			if !ok || sel.Where == nil {
				t.Errorf("[%s] %q WHERE未识别", name, sql)
				continue
			}
			if strings.TrimSpace(sel.Where.Text) == "" {
				t.Errorf("[%s] %q 降级时Text不得为空", name, sql)
			}
		}
	}
}

// TestExprOnAndHaving ON与HAVING场景同样建树
func TestExprOnAndHaving(t *testing.T) {
	for name, p := range allParsers() {
		// ON：ON后接JOIN子句（SkipCondExpr边界）
		stmt, err := p.Parse("SELECT a.id FROM ta a JOIN tb b ON a.id = b.id AND a.x > 1 LEFT JOIN tc c ON b.y = c.y")
		if err != nil {
			t.Errorf("[%s] ON建树解析失败: %v", name, err)
			continue
		}
		sel := stmt.(*sqlstmt.SelectStmt)
		if len(sel.Joins) != 2 {
			t.Errorf("[%s] joins=%d，期望2", name, len(sel.Joins))
			continue
		}
		if sel.Joins[0].On == nil || sel.Joins[0].On.Root == nil {
			t.Errorf("[%s] 第一个JOIN的ON应建树", name)
		}

		// HAVING
		stmt, err = p.Parse("SELECT dept, COUNT(*) FROM t GROUP BY dept HAVING COUNT(*) > 1 AND MAX(s) < 100")
		if err != nil {
			t.Errorf("[%s] HAVING解析失败: %v", name, err)
			continue
		}
		sel = stmt.(*sqlstmt.SelectStmt)
		if sel.Having == nil || sel.Having.Root == nil {
			t.Errorf("[%s] HAVING应建树", name)
		}
	}
}

// TestExprNotPrecedence NOT一元的绑定力介于AND与比较之间：
// NOT a = 1 AND b = 2 必须解析为 (NOT (a=1)) AND (b=2)，而非吞掉AND（回归保护）
func TestExprNotPrecedence(t *testing.T) {
	for name, p := range allParsers() {
		stmt, err := p.Parse("SELECT id FROM t WHERE NOT a = 1 AND b = 2")
		if err != nil {
			t.Errorf("[%s] 解析失败: %v", name, err)
			continue
		}
		sel := stmt.(*sqlstmt.SelectStmt)
		if sel.Where == nil || sel.Where.Root == nil {
			t.Errorf("[%s] WHERE未建树", name)
			continue
		}
		and, ok := sel.Where.Root.(*sqlstmt.BinaryExpr)
		if !ok || and.Op != "AND" {
			t.Errorf("[%s] 顶层应为AND，得到%T %+v", name, sel.Where.Root, sel.Where.Root)
			continue
		}
		not, ok := and.Left.(*sqlstmt.UnaryExpr)
		if !ok || not.Op != "NOT" {
			t.Errorf("[%s] AND左侧应为NOT一元，得到%T", name, and.Left)
			continue
		}
		eq, ok := not.Operand.(*sqlstmt.BinaryExpr)
		if !ok || eq.Op != "=" {
			t.Errorf("[%s] NOT操作数应为a=1，得到%T", name, not.Operand)
		}
	}
}
