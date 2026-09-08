//go:build it

package sync

// 数据同步（srcData2TargetDb）真实链路集成测试：
// 从mysql源表读取数据，经字段映射→目标方言GenInsert（ON CONFLICT upsert）→事务写入pg目标表，
// 验证全量同步与DuplicateStrategy覆盖策略（冲突时更新）真实生效。
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/application/

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

func syncTestConn(t *testing.T, di *dbi.DbInfo) *dbi.DbConn {
	t.Helper()
	conn, err := dbm.Conn(context.Background(), di)
	require.NoError(t, err)
	return conn
}

// TestITDataSyncSrc2TargetDb 数据同步 mysql源 → pg目标（含冲突覆盖策略）
func TestITDataSyncSrc2TargetDb(t *testing.T) {
	// 源：mysql
	srcConn := syncTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	defer srcConn.Close()
	mustSyncExec(t, srcConn, "DROP TABLE IF EXISTS `it_sync_src`")
	mustSyncExec(t, srcConn, "CREATE TABLE `it_sync_src` (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")
	mustSyncExec(t, srcConn, "INSERT INTO `it_sync_src` VALUES (1,'alice',100),(2,'bob',200)")

	// 目标：pg
	dstConn := syncTestConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	defer dstConn.Close()
	mustSyncExec(t, dstConn, "DROP TABLE IF EXISTS it_sync_target")
	mustSyncExec(t, dstConn, "CREATE TABLE it_sync_target (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")

	targetColumns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true},
		{ColumnName: "name", DataType: "varchar"},
		{ColumnName: "salary", DataType: "int"},
	}
	targetMeta := dbi.BuildTargetTableMeta(dstConn, "it_sync_target", targetColumns)

	_, srcRes, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_src` ORDER BY id")
	require.NoError(t, err)
	require.Len(t, srcRes, 2)

	fieldMap := []map[string]string{
		{"src": "id", "target": "id"},
		{"src": "name", "target": "name"},
		{"src": "salary", "target": "salary"},
	}
	task := &entity.DataSyncTask{TargetTableName: "it_sync_target", DuplicateStrategy: 2} // 2=覆盖(upsert)

	app := &DataSyncAppImpl{}
	// 首次全量同步
	require.NoError(t, app.srcData2TargetDb(context.Background(), srcRes, fieldMap, "", task, dstConn, targetColumns, targetMeta))

	_, rows, err := dstConn.Query("SELECT id, name, salary FROM it_sync_target ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "alice", fmt.Sprint(rows[0]["name"]))
	assert.Equal(t, int64(100), syncToInt64(rows[0]["salary"]))

	// 修改源数据后再次同步：主键冲突应走覆盖策略（ON CONFLICT DO UPDATE）
	mustSyncExec(t, srcConn, "UPDATE `it_sync_src` SET salary = 999 WHERE id = 1")
	_, srcRes2, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_src` ORDER BY id")
	require.NoError(t, err)
	require.NoError(t, app.srcData2TargetDb(context.Background(), srcRes2, fieldMap, "", task, dstConn, targetColumns, targetMeta))

	_, rows2, err := dstConn.Query("SELECT id, name, salary FROM it_sync_target ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows2, 2, "冲突覆盖后不应产生重复行")
	assert.Equal(t, int64(999), syncToInt64(rows2[0]["salary"]), "冲突覆盖后值应更新")

	// 清理
	_, _ = dstConn.Exec("DROP TABLE IF EXISTS it_sync_target")
	_, _ = srcConn.Exec("DROP TABLE IF EXISTS `it_sync_src`")
}

// TestITDataSyncToMysqlTarget 数据同步 mysql源 → mysql目标（replace into 覆盖策略）
func TestITDataSyncToMysqlTarget(t *testing.T) {
	srcConn := syncTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	defer srcConn.Close()
	mustSyncExec(t, srcConn, "DROP TABLE IF EXISTS `it_sync_m_src`")
	mustSyncExec(t, srcConn, "CREATE TABLE `it_sync_m_src` (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")
	mustSyncExec(t, srcConn, "INSERT INTO `it_sync_m_src` VALUES (1,'alice',100),(2,'bob',200)")

	targetColumns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true},
		{ColumnName: "name", DataType: "varchar"},
		{ColumnName: "salary", DataType: "int"},
	}

	// 目标：同实例mysql（replace into覆盖策略）
	dstConn := syncTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	defer dstConn.Close()
	mustSyncExec(t, dstConn, "DROP TABLE IF EXISTS `it_sync_m_target`")
	mustSyncExec(t, dstConn, "CREATE TABLE `it_sync_m_target` (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")

	targetMeta := dbi.BuildTargetTableMeta(dstConn, "it_sync_m_target", targetColumns)
	_, srcRes, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_m_src` ORDER BY id")
	require.NoError(t, err)
	require.Len(t, srcRes, 2)

	fieldMap := []map[string]string{
		{"src": "id", "target": "id"},
		{"src": "name", "target": "name"},
		{"src": "salary", "target": "salary"},
	}
	task := &entity.DataSyncTask{TargetTableName: "it_sync_m_target", DuplicateStrategy: 2}

	app := &DataSyncAppImpl{}
	require.NoError(t, app.srcData2TargetDb(context.Background(), srcRes, fieldMap, "", task, dstConn, targetColumns, targetMeta))

	_, rows, err := dstConn.Query("SELECT id, name, salary FROM `it_sync_m_target` ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "alice", fmt.Sprint(rows[0]["name"]))

	// 源数据变更后重同步：replace into覆盖后不产生重复行且值更新
	mustSyncExec(t, srcConn, "UPDATE `it_sync_m_src` SET salary = 777 WHERE id = 2")
	_, srcRes2, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_m_src` ORDER BY id")
	require.NoError(t, err)
	require.NoError(t, app.srcData2TargetDb(context.Background(), srcRes2, fieldMap, "", task, dstConn, targetColumns, targetMeta))

	_, rows2, err := dstConn.Query("SELECT id, name, salary FROM `it_sync_m_target` ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows2, 2, "replace into覆盖后不应产生重复行")
	assert.Equal(t, int64(777), syncToInt64(rows2[1]["salary"]), "replace into覆盖后值应更新")

	_, _ = dstConn.Exec("DROP TABLE IF EXISTS `it_sync_m_target`")
	_, _ = srcConn.Exec("DROP TABLE IF EXISTS `it_sync_m_src`")
}

// TestITDataSyncToSqliteTarget 数据同步 mysql源 → sqlite目标（insert or replace覆盖策略）
func TestITDataSyncToSqliteTarget(t *testing.T) {
	srcConn := syncTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	defer srcConn.Close()
	mustSyncExec(t, srcConn, "DROP TABLE IF EXISTS `it_sync_sq_src`")
	mustSyncExec(t, srcConn, "CREATE TABLE `it_sync_sq_src` (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")
	mustSyncExec(t, srcConn, "INSERT INTO `it_sync_sq_src` VALUES (1,'alice',100),(2,'bob',200)")

	targetColumns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true},
		{ColumnName: "name", DataType: "varchar"},
		{ColumnName: "salary", DataType: "int"},
	}

	// 目标：sqlite文件库（sqlite方言要求库文件已存在，需先创建空文件）
	sqlitePath := filepath.Join(t.TempDir(), "sync_target.sqlite")
	require.NoError(t, os.WriteFile(sqlitePath, nil, 0o644))
	dstConn := syncTestConn(t, &dbi.DbInfo{Type: "sqlite", Host: sqlitePath})
	defer dstConn.Close()
	mustSyncExec(t, dstConn, "CREATE TABLE `it_sync_sq_target` (id INTEGER PRIMARY KEY, name TEXT, salary INTEGER)")

	targetMeta := dbi.BuildTargetTableMeta(dstConn, "it_sync_sq_target", targetColumns)
	_, srcRes, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_sq_src` ORDER BY id")
	require.NoError(t, err)
	require.Len(t, srcRes, 2)

	fieldMap := []map[string]string{
		{"src": "id", "target": "id"},
		{"src": "name", "target": "name"},
		{"src": "salary", "target": "salary"},
	}
	task := &entity.DataSyncTask{TargetTableName: "it_sync_sq_target", DuplicateStrategy: 2}

	app := &DataSyncAppImpl{}
	require.NoError(t, app.srcData2TargetDb(context.Background(), srcRes, fieldMap, "", task, dstConn, targetColumns, targetMeta))

	_, rows, err := dstConn.Query("SELECT id, name, salary FROM `it_sync_sq_target` ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "alice", fmt.Sprint(rows[0]["name"]))
	assert.Equal(t, int64(100), syncToInt64(rows[0]["salary"]))

	// 源数据变更后重同步：insert or replace覆盖
	mustSyncExec(t, srcConn, "UPDATE `it_sync_sq_src` SET salary = 888 WHERE id = 1")
	_, srcRes2, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_sq_src` ORDER BY id")
	require.NoError(t, err)
	require.NoError(t, app.srcData2TargetDb(context.Background(), srcRes2, fieldMap, "", task, dstConn, targetColumns, targetMeta))

	_, rows2, err := dstConn.Query("SELECT id, name, salary FROM `it_sync_sq_target` ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows2, 2, "insert or replace覆盖后不应产生重复行")
	assert.Equal(t, int64(888), syncToInt64(rows2[0]["salary"]), "insert or replace覆盖后值应更新")
}

func mustSyncExec(t *testing.T, conn *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := conn.Exec(sql); err != nil {
		t.Fatalf("exec failed [%s]: %s", sql, err.Error())
	}
}

func syncToInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int16:
		return int64(val)
	case uint64:
		return int64(val)
	case float64:
		return int64(val)
	default:
		// 驱动可能返回string/[]byte形式的数字（如"100"），统一解析
		n, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(v)), 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
}
