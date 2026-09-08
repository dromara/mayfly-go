//go:build it

package sync

// 复杂源查询数据同步IT：源SQL使用JOIN+GROUP BY+HAVING的聚合查询做真实同步，
// 验证解析器GroupBy/Having回填（CaptureSkip）与validateDataSyncAppendable
// 校验在同步全链路的正确性——这类SQL历史上曾被GROUP BY解析缺陷误拦或误拼

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
)

// TestITDataSyncComplexJoinGroupBy 复杂聚合源查询 → pg目标真实同步
func TestITDataSyncComplexJoinGroupBy(t *testing.T) {
	srcConn := syncTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	defer srcConn.Close()
	mustSyncExec(t, srcConn, "DROP TABLE IF EXISTS `it_cs_dept`")
	mustSyncExec(t, srcConn, "DROP TABLE IF EXISTS `it_cs_emp`")
	mustSyncExec(t, srcConn, "CREATE TABLE `it_cs_dept` (dept VARCHAR(32) PRIMARY KEY, city VARCHAR(64))")
	mustSyncExec(t, srcConn, "CREATE TABLE `it_cs_emp` (id INT PRIMARY KEY, dept VARCHAR(32), salary INT)")
	mustSyncExec(t, srcConn, "INSERT INTO `it_cs_dept` VALUES ('eng','bj'),('sales','sh')")
	mustSyncExec(t, srcConn, "INSERT INTO `it_cs_emp` VALUES (1,'eng',3000),(2,'eng',5000),(3,'sales',4000)")

	dstConn := syncTestConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	defer dstConn.Close()
	mustSyncExec(t, dstConn, "DROP TABLE IF EXISTS it_cs_target")
	mustSyncExec(t, dstConn, "CREATE TABLE it_cs_target (dept VARCHAR(32) PRIMARY KEY, cnt INT)")

	targetColumns := []dbi.Column{
		{ColumnName: "dept", DataType: "varchar", IsPrimaryKey: true},
		{ColumnName: "cnt", DataType: "int"},
	}
	targetMeta := dbi.BuildTargetTableMeta(dstConn, "it_cs_target", targetColumns)

	// 复杂源查询：JOIN + GROUP BY + HAVING（含函数聚合），解析器必须完整回填GroupBy/Having
	complexSrcSql := "SELECT e.dept, COUNT(*) AS cnt FROM `it_cs_emp` e JOIN `it_cs_dept` d ON d.dept = e.dept GROUP BY e.dept HAVING COUNT(*) >= 1"
	_, srcRes, err := srcConn.Query(complexSrcSql)
	require.NoError(t, err)
	require.Len(t, srcRes, 2, "聚合源查询应返回2个分组")

	// 解析断言：GroupBy/Having必须被识别（同步可内省校验依赖）
	stmt, err := srcConn.GetDialect().GetSQLParser().Parse(complexSrcSql)
	require.NoError(t, err)
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	require.True(t, ok, "期望SelectStmt，得到%T", stmt)
	require.Len(t, sel.GroupBy, 1, "GROUP BY应被回填")
	require.NotNil(t, sel.Having, "HAVING应被回填")

	fieldMap := []map[string]string{
		{"src": "dept", "target": "dept"},
		{"src": "cnt", "target": "cnt"},
	}
	task := &entity.DataSyncTask{TargetTableName: "it_cs_target", DuplicateStrategy: 2}
	app := &DataSyncAppImpl{}
	require.NoError(t, app.SyncBatch(context.Background(), srcRes, fieldMap, "", task, dstConn, targetColumns, targetMeta))

	_, rows, err := dstConn.Query("SELECT dept, cnt FROM it_cs_target ORDER BY dept")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "eng", fmt.Sprint(rows[0]["dept"]))
	assert.Equal(t, int64(2), syncToInt64(rows[0]["cnt"]), "eng组应有2人")
	assert.Equal(t, int64(1), syncToInt64(rows[1]["cnt"]), "sales组应有1人")

	// 清理
	_, _ = dstConn.Exec("DROP TABLE IF EXISTS it_cs_target")
	_, _ = srcConn.Exec("DROP TABLE IF EXISTS `it_cs_emp`")
	_, _ = srcConn.Exec("DROP TABLE IF EXISTS `it_cs_dept`")
}
