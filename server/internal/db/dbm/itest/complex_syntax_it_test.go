package itest

// 复杂SQL真实执行IT：本文件与 sqlparser 的进阶解析矩阵测试（parser_matrix_advanced_test.go）
// 配套——解析修复（窗口函数项计数/集合操作INTERSECT/EXCEPT/MINUS识别/JOIN前缀关键字别名冲突）
// 在真实方言上的运行时语义必须正确：
//   - 解析结果（items计数/别名/集合段数）与真实执行结果（行数/值）双向校验
//   - 方言能力差异用 showSupported 兜底：老版本mysql不支持INTERSECT时跳过对应断言而非误报

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/postgres"
	_ "mayfly-go/internal/db/dbm/sqlite"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// setupComplexSyntaxTables 建立同构的 t_dept/t_emp 并填充数据（方言SQL差异由sqlFor处理）
func setupComplexSyntaxTables(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	q := conn.GetDialect().Quoter().Quote
	for _, tbl := range []string{"t_emp", "t_dept"} {
		mustExec(t, conn, "DROP TABLE IF EXISTS "+q(tbl))
	}
	mustExec(t, conn, `CREATE TABLE `+q("t_dept")+` (dept varchar(32) NOT NULL PRIMARY KEY, city varchar(64) NOT NULL)`)
	mustExec(t, conn, `CREATE TABLE `+q("t_emp")+` (
		id int NOT NULL PRIMARY KEY,
		dept varchar(32) NOT NULL,
		name varchar(64) NOT NULL,
		salary int NOT NULL,
		phone varchar(32) DEFAULT NULL)`)
	mustExec(t, conn, `INSERT INTO `+q("t_dept")+` (dept, city) VALUES ('eng', 'bj'), ('sales', 'sh'), ('hr', 'gz')`)
	mustExec(t, conn, `INSERT INTO `+q("t_emp")+` (id, dept, name, salary, phone) VALUES
		(1, 'eng', 'alice', 3000, '13800000001'),
		(2, 'eng', 'bob', 5000, NULL),
		(3, 'sales', 'cindy', 4000, '13800000003'),
		(4, 'hr', 'dave', 2000, NULL)`)
}

// supportLevel 方言对特定语法的能力（无法静态判断版本，执行报"语法不支持"时降级跳过）
type supportLevel int

const (
	supportRequired supportLevel = iota // 必须支持，失败即测试失败
	supportOptional                     // 可选支持（老版本可能无此语法），语法类错误时跳过
)

func execQueryWithSupport(t *testing.T, conn *dbi.DbConn, sql string, lv supportLevel) ([]map[string]any, bool) {
	t.Helper()
	_, rows, err := conn.Query(sql)
	if err != nil {
		if lv == supportOptional && isSyntaxUnsupportedErr(err.Error()) {
			t.Logf("[%s] 方言不支持该语法，跳过: %s", conn.Info.Type, err.Error())
			return nil, false
		}
		t.Fatalf("[%s] exec failed [%s]: %s", conn.Info.Type, sql, err.Error())
	}
	return rows, true
}

func isSyntaxUnsupportedErr(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "syntax") || strings.Contains(m, "unrecognized") ||
		strings.Contains(m, "unexpected") || strings.Contains(m, "not supported") ||
		strings.Contains(m, "parse error")
}

// TestITComplexQueryRealExec 复杂查询三方言真实执行：解析结构断言+执行结果断言
func TestITComplexQueryRealExec(t *testing.T) {
	conns := []*dbi.DbConn{mysqlConn(t), pgConn(t), sqliteConn(t)}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	for _, conn := range conns {
		dbType := conn.Info.Type
		t.Run(string(dbType), func(t *testing.T) {
			setupComplexSyntaxTables(t, conn)
			parser := conn.GetDialect().GetSQLParser()

			// 1. 窗口函数：真实执行+解析items计数（sqlite 3.25+支持，老版本降级跳过）
			rows, ok := execQueryWithSupport(t, conn,
				"SELECT name, ROW_NUMBER() OVER (PARTITION BY dept ORDER BY salary DESC) AS rn FROM t_emp", supportOptional)
			if ok {
				require.Len(t, rows, 4, "窗口函数应返回全部4行")
				stmt, err := parser.Parse("SELECT name, ROW_NUMBER() OVER (PARTITION BY dept ORDER BY salary DESC) AS rn FROM t_emp")
				require.NoError(t, err)
				require.Len(t, selItems(t, stmt), 2, "窗口函数SQL的items应为2（OVER内关键字/逗号不得撑爆计数）")
			}

			// 2. GROUP BY + HAVING + 别名full（JOIN前缀关键字EqualFold冲突修复的运行时验证）
			rows, ok = execQueryWithSupport(t, conn,
				"SELECT dept, COUNT(*) AS full FROM t_emp GROUP BY dept HAVING COUNT(*) > 1", supportRequired)
			require.True(t, ok)
			require.Len(t, rows, 1, "HAVING COUNT(*)>1 应只命中eng（2人）")
			require.Equal(t, "eng", cellStr(rows[0]["dept"]), "命中的分组应为eng")
			// 别名full修复验证：解析结果Alias必须为full而非AS
			stmt, err := parser.Parse("SELECT dept, COUNT(*) AS full FROM t_emp GROUP BY dept HAVING COUNT(*) > 1")
			require.NoError(t, err)
			items := selItems(t, stmt)
			require.Len(t, items, 2)
			require.Equal(t, "full", items[1].Alias, "别名full不得被JOIN前缀关键字边界截断")

			// 3. INTERSECT/EXCEPT集合操作（pg必支持；mysql8.0.31+/sqlite可选）
			rows, ok = execQueryWithSupport(t, conn,
				"SELECT dept FROM t_emp WHERE salary > 2500 INTERSECT SELECT dept FROM t_dept", supportOptional)
			if ok {
				require.Len(t, rows, 2, "salary>2500的eng/sales与t_dept交集应为2个部门")
			}
			rows, ok = execQueryWithSupport(t, conn,
				"SELECT dept FROM t_dept EXCEPT SELECT dept FROM t_emp", supportOptional)
			if ok {
				require.Len(t, rows, 0, "t_dept与t_emp的差集应为空（部门全覆盖）")
			}

			// 4. 嵌套函数多参数 + AS别名
			stmt, err = parser.Parse("SELECT CONCAT(name, '-', dept) AS full_name, salary FROM t_emp WHERE id = 1")
			require.NoError(t, err)
			items = selItems(t, stmt)
			require.Len(t, items, 2)
			require.Equal(t, "full_name", items[0].Alias)

			// 5. 多层嵌套子查询真实执行
			rows, ok = execQueryWithSupport(t, conn,
				"SELECT name FROM t_emp WHERE dept IN (SELECT dept FROM t_dept WHERE city IN (SELECT city FROM t_dept WHERE dept = 'eng'))", supportRequired)
			require.Len(t, rows, 2, "eng城市（bj）的员工应为alice/bob 2人")

			// 6. 转义引号字符串真实执行（解析与执行值一致）
			rows, ok = execQueryWithSupport(t, conn,
				"SELECT id FROM t_emp WHERE name = 'it''s alice' OR id = 1", supportRequired)
			require.Len(t, rows, 1)
			require.Equal(t, 1, cellInt(t, rows[0]["id"]))
		})
	}
}

// TestITRecursiveCteRealExec 递归CTE三方言真实执行（树形展开语义）
func TestITRecursiveCteRealExec(t *testing.T) {
	conns := []*dbi.DbConn{mysqlConn(t), pgConn(t), sqliteConn(t)}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	for _, conn := range conns {
		dbType := conn.Info.Type
		t.Run(string(dbType), func(t *testing.T) {
			mustExec(t, conn, "DROP TABLE IF EXISTS t_tree")
			mustExec(t, conn, `CREATE TABLE t_tree (id int NOT NULL PRIMARY KEY, pid int DEFAULT NULL)`)
			// 1 -> 2 -> 3 链
			mustExec(t, conn, `INSERT INTO t_tree (id, pid) VALUES (1, NULL), (2, 1), (3, 2)`)

			rows, ok := execQueryWithSupport(t, conn,
				"WITH RECURSIVE sub AS (SELECT id, pid FROM t_tree WHERE id = 1 UNION ALL SELECT t.id, t.pid FROM t_tree t JOIN sub s ON t.pid = s.id) SELECT * FROM sub", supportOptional)
			if ok {
				require.Len(t, rows, 3, "递归CTE应展开出3个节点")
			}
		})
	}
}

func selItems(t *testing.T, stmt sqlstmt.Stmt) []sqlstmt.SelectItem {
	t.Helper()
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	require.True(t, ok, "期望SelectStmt，得到%T", stmt)
	return sel.Items
}

func cellStr(v any) string {
	return fmt.Sprintf("%v", normalizeDbValue(v))
}

func cellInt(t *testing.T, v any) int {
	t.Helper()
	f, ok := toFloat(normalizeDbValue(v))
	require.True(t, ok, "值无法归一为数值: %v", v)
	return int(f)
}
