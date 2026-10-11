package sqlparser

// JOIN 连接类型与 USING / NATURAL 的跨方言解析矩阵。
//
// 钉两类曾被静默误解析的形态（均为 parseJoinClause 在 4 份拷贝里的漂移）：
//   - USING 曾未登记为关键字，各方言 parseJoinClause 的 `else if IsKeyword("USING")` 分支恒不触发，
//     mysql 会把 `INNER JOIN t2 b USING (id)` 里的 USING、(id) 当成两张 FROM 表（from 虚增到 3）；
//   - dm 的 parseJoinClause 缺 NATURAL 分支，而 IsJoinStart 已含 NATURAL，导致 dm 停在 NATURAL
//     返回 nil、整个 JOIN 连同其后的 WHERE/ORDER 被丢弃（回滚/脱敏按「无 WHERE」误判形态）。
//
// 连接类型用枚举而非计数断言，确保「把 LEFT 认成 INNER」这类静默降级也会被抓到。
import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

func TestJoinKindsCrossDialect(t *testing.T) {
	cases := []struct {
		name      string
		sql       string
		from      int
		kinds     []sqlstmt.JoinKind
		onBuilt   []bool // 对应 JOIN 是否解析出 ON 条件（USING/CROSS/NATURAL 无 ON）
		wantWhere bool
	}{
		{
			name: "inner-on", sql: "SELECT a.id FROM t1 a JOIN t2 b ON a.id = b.id",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindInner}, onBuilt: []bool{true},
		},
		{
			name: "left", sql: "SELECT a.id FROM t1 a LEFT JOIN t2 b ON a.id = b.id",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindLeft}, onBuilt: []bool{true},
		},
		{
			name: "left-outer", sql: "SELECT a.id FROM t1 a LEFT OUTER JOIN t2 b ON a.id = b.id",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindLeft}, onBuilt: []bool{true},
		},
		{
			name: "right", sql: "SELECT a.id FROM t1 a RIGHT JOIN t2 b ON a.id = b.id",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindRight}, onBuilt: []bool{true},
		},
		{
			name: "full", sql: "SELECT a.id FROM t1 a FULL JOIN t2 b ON a.id = b.id",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindFull}, onBuilt: []bool{true},
		},
		{
			name: "full-outer", sql: "SELECT a.id FROM t1 a FULL OUTER JOIN t2 b ON a.id = b.id",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindFull}, onBuilt: []bool{true},
		},
		{
			name: "cross", sql: "SELECT a.id FROM t1 a CROSS JOIN t2 b",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindCross}, onBuilt: []bool{false},
		},
		{
			name: "natural", sql: "SELECT a.id FROM t1 a NATURAL JOIN t2 b",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindNatural}, onBuilt: []bool{false},
		},
		// NATURAL JOIN 后仍须保留 WHERE（dm 曾因缺分支把整个 JOIN 及其后子句一起丢弃）
		{
			name: "natural-with-where", sql: "SELECT a.id FROM t1 a NATURAL JOIN t2 b WHERE a.id > 5",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindNatural}, onBuilt: []bool{false}, wantWhere: true,
		},
		// USING 不得泄漏为 FROM 表（mysql 曾 from=3）
		{
			name: "inner-using", sql: "SELECT a.id FROM t1 a INNER JOIN t2 b USING (id)",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindInner}, onBuilt: []bool{false},
		},
		{
			name: "self-join", sql: "SELECT e.name, m.name FROM emp e LEFT JOIN emp m ON e.mgr = m.id",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindLeft}, onBuilt: []bool{true},
		},
		{
			name: "multi-chain", sql: "SELECT a.id FROM t1 a JOIN t2 b ON a.id = b.id JOIN t3 c ON b.id = c.id LEFT JOIN t4 d ON c.id = d.id",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindInner, sqlstmt.JoinKindInner, sqlstmt.JoinKindLeft}, onBuilt: []bool{true, true, true},
		},
		// 派生表 JOIN 派生表 + USING：FROM 只有一张（子查询派生），不得把 USING 计入
		{
			name: "derived-join", sql: "SELECT x.id FROM (SELECT id FROM t1 WHERE id > 0) x JOIN t2 y USING (id)",
			from: 1, kinds: []sqlstmt.JoinKind{sqlstmt.JoinKindInner}, onBuilt: []bool{false},
		},
	}
	for _, c := range cases {
		for dialect, p := range allParsers() {
			stmt, err := p.Parse(c.sql)
			if err != nil {
				t.Errorf("[%s/%s] 解析失败: %v", dialect, c.name, err)
				continue
			}
			sel, ok := stmt.(*sqlstmt.SelectStmt)
			if !ok {
				t.Errorf("[%s/%s] 期望 SelectStmt，得到 %T", dialect, c.name, stmt)
				continue
			}
			if len(sel.From) != c.from {
				t.Errorf("[%s/%s] FROM 表数=%d，期望%d（USING/子查询泄漏成 FROM 表？）: %+v", dialect, c.name, len(sel.From), c.from, sel.From)
			}
			if len(sel.Joins) != len(c.kinds) {
				t.Errorf("[%s/%s] JOIN 数=%d，期望%d", dialect, c.name, len(sel.Joins), len(c.kinds))
				continue
			}
			for i, wantKind := range c.kinds {
				if got := sel.Joins[i].Kind; got != wantKind {
					t.Errorf("[%s/%s] JOIN#%d 类型=%d，期望%d（连接类型被静默降级）", dialect, c.name, i, got, wantKind)
				}
				if gotOn := sel.Joins[i].On != nil; gotOn != c.onBuilt[i] {
					t.Errorf("[%s/%s] JOIN#%d ON 存在性=%v，期望%v", dialect, c.name, i, gotOn, c.onBuilt[i])
				}
			}
			if (sel.Where != nil) != c.wantWhere {
				t.Errorf("[%s/%s] WHERE 存在性=%v，期望%v（JOIN 解析中断致其后子句被丢弃？）", dialect, c.name, sel.Where != nil, c.wantWhere)
			}
		}
	}
}
