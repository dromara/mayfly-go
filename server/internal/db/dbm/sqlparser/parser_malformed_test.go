package sqlparser

// 跨方言畸形SQL回归测试：验证解析器对非法/截断SQL不会死循环或panic（DoS防御）
import (
	"testing"
	"time"

	"mayfly-go/internal/db/dbm/sqlparser/dm"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/internal/db/dbm/sqlparser/oracle"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// malformedSQLs 畸形SQL清单：历史上曾触发 JOIN 解析死循环（parseJoinClause失败重置Pos而调用方循环不推进）
var malformedSQLs = []string{
	"SELECT NATURAL JOIN",
	"SELECT LEFT JOIN",
	"SELECT INNER JOIN ON",
	"SELECT * FROM JOIN",
	"SELECT * FROM t1 LEFT JOIN",
	"SELECT * FROM t1 JOIN ON",
	"SELECT * FROM t1, LEFT JOIN t2",
	"UPDATE t1 JOIN SET a = 1",
	"SELECT FROM WHERE",
	"SELECT ((",
	"SELECT '",
	"SELECT * FROM t1 WHERE",
}

// TestMalformedSQLNoDeadloop 畸形SQL必须在限定时间内返回（不允许panic逃逸或死循环）
func TestMalformedSQLNoDeadloop(t *testing.T) {
	parsers := map[string]func(string) (sqlstmt.Stmt, error){
		"mysql":  new(mysql.MysqlParser).Parse,
		"pgsql":  new(pgsql.PgsqlParser).Parse,
		"dm":     new(dm.DmParser).Parse,
		"oracle": new(oracle.OracleParser).Parse,
	}
	for name, parse := range parsers {
		for _, sql := range malformedSQLs {
			done := make(chan struct{})
			go func(name, sql string) {
				defer close(done)
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("[%s] %q panic逃逸: %v", name, sql, r)
					}
				}()
				_, _ = parse(sql) // 允许返回错误或空stmt，但必须返回
			}(name, sql)
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatalf("[%s] %q 解析死循环(>3s)", name, sql)
			}
		}
	}
}

// TestMalformedSQLBasics 畸形SQL后接合法子句仍能解析出核心结构（降级但不至于全盘丢失）
func TestMalformedSQLBasics(t *testing.T) {
	stmt, err := new(mysql.MysqlParser).Parse("SELECT * FROM t1 JOIN WHERE id = 1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok {
		t.Fatalf("期望SelectStmt，得到%T", stmt)
	}
	if len(sel.From) != 1 || sel.From[0].Name != "t1" {
		t.Fatalf("FROM解析错误: %+v", sel.From)
	}
}
