//go:build it

package itest

// 数据同步真实链路集成测试（黑盒，经导出入口 SyncBatch）：
// mysql 源读数据 → 字段映射 → 目标方言 GenInsert / 参数化直插 → 事务写入，
// 验证全量同步与冲突覆盖策略（upsert / replace / insert or replace）真实生效。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

func syncBatch(app *sync.DataSyncAppImpl, rows []map[string]any, fieldMap []map[string]string, task *entity.DataSyncTask, dst *dbi.DbConn, cols []dbi.Column) error {
	meta := sync.BuildTargetTableMeta(dst, task.TargetTableName, cols)
	return app.SyncBatch(context.Background(), rows, fieldMap, "", task, dst, cols, meta)
}

func idNameSalaryCols() []dbi.Column {
	return []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true},
		{ColumnName: "name", DataType: "varchar"},
		{ColumnName: "salary", DataType: "int"},
	}
}

func idNameSalaryMap() []map[string]string {
	return []map[string]string{
		{"src": "id", "target": "id"},
		{"src": "name", "target": "name"},
		{"src": "salary", "target": "salary"},
	}
}

func TestITDataSyncSrc2TargetDb(t *testing.T) {
	srcConn := mysqlConn(t)
	defer srcConn.Close()
	mustExec(t, srcConn, "DROP TABLE IF EXISTS `it_sync_src`")
	mustExec(t, srcConn, "CREATE TABLE `it_sync_src` (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")
	mustExec(t, srcConn, "INSERT INTO `it_sync_src` VALUES (1,'alice',100),(2,'bob',200)")

	dstConn := pgConn(t)
	defer dstConn.Close()
	mustExec(t, dstConn, "DROP TABLE IF EXISTS it_sync_target")
	mustExec(t, dstConn, "CREATE TABLE it_sync_target (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")

	app, _ := newApp()
	task := &entity.DataSyncTask{TargetTableName: "it_sync_target", DuplicateStrategy: 2}

	_, srcRes, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_src` ORDER BY id")
	require.NoError(t, err)
	require.NoError(t, syncBatch(app, srcRes, idNameSalaryMap(), task, dstConn, idNameSalaryCols()))

	mustExec(t, srcConn, "UPDATE `it_sync_src` SET salary = 999 WHERE id = 1")
	_, srcRes2, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_src` ORDER BY id")
	require.NoError(t, err)
	require.NoError(t, syncBatch(app, srcRes2, idNameSalaryMap(), task, dstConn, idNameSalaryCols()))

	_, rows, err := dstConn.Query("SELECT id, name, salary FROM it_sync_target ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows, 2, "冲突覆盖后不应产生重复行")
	assert.Equal(t, int64(999), toInt64(rows[0]["salary"]), "冲突覆盖后值应更新")
}

func TestITDataSyncToMysqlTarget(t *testing.T) {
	srcConn := mysqlConn(t)
	defer srcConn.Close()
	mustExec(t, srcConn, "DROP TABLE IF EXISTS `it_sync_m_src`")
	mustExec(t, srcConn, "CREATE TABLE `it_sync_m_src` (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")
	mustExec(t, srcConn, "INSERT INTO `it_sync_m_src` VALUES (1,'alice',100),(2,'bob',200)")

	dstConn := mysqlConn(t)
	defer dstConn.Close()
	mustExec(t, dstConn, "DROP TABLE IF EXISTS `it_sync_m_target`")
	mustExec(t, dstConn, "CREATE TABLE `it_sync_m_target` (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")

	app, _ := newApp()
	task := &entity.DataSyncTask{TargetTableName: "it_sync_m_target", DuplicateStrategy: 2}
	_, srcRes, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_m_src` ORDER BY id")
	require.NoError(t, err)
	require.NoError(t, syncBatch(app, srcRes, idNameSalaryMap(), task, dstConn, idNameSalaryCols()))

	mustExec(t, srcConn, "UPDATE `it_sync_m_src` SET salary = 777 WHERE id = 2")
	_, srcRes2, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_m_src` ORDER BY id")
	require.NoError(t, err)
	require.NoError(t, syncBatch(app, srcRes2, idNameSalaryMap(), task, dstConn, idNameSalaryCols()))

	_, rows, err := dstConn.Query("SELECT id, name, salary FROM `it_sync_m_target` ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows, 2, "replace into 覆盖后不应产生重复行")
	assert.Equal(t, int64(777), toInt64(rows[1]["salary"]))
}

// TestITDataSyncParamInsertToMysql 验证「参数化直插」路径：目标 mysql、直接插入策略时走占位符绑定，
// 含单引号值交驱动绑定应精确落地。
func TestITDataSyncParamInsertToMysql(t *testing.T) {
	srcConn := mysqlConn(t)
	defer srcConn.Close()
	mustExec(t, srcConn, "DROP TABLE IF EXISTS `it_sync_pi_src`")
	mustExec(t, srcConn, "CREATE TABLE `it_sync_pi_src` (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")
	mustExec(t, srcConn, "INSERT INTO `it_sync_pi_src` VALUES (1,'o''brien',100),(2,'bob',200)")

	dstConn := mysqlConn(t)
	defer dstConn.Close()
	mustExec(t, dstConn, "DROP TABLE IF EXISTS `it_sync_pi_target`")
	mustExec(t, dstConn, "CREATE TABLE `it_sync_pi_target` (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")

	app, _ := newApp()
	task := &entity.DataSyncTask{TargetTableName: "it_sync_pi_target", DuplicateStrategy: dbi.DuplicateStrategyNone}
	_, srcRes, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_pi_src` ORDER BY id")
	require.NoError(t, err)
	require.NoError(t, syncBatch(app, srcRes, idNameSalaryMap(), task, dstConn, idNameSalaryCols()))

	_, rows, err := dstConn.Query("SELECT id, name, salary FROM `it_sync_pi_target` ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "o'brien", fmt.Sprint(rows[0]["name"]), "单引号值经驱动参数绑定应精确落地")
}

func TestITDataSyncToSQLiteTarget(t *testing.T) {
	srcConn := mysqlConn(t)
	defer srcConn.Close()
	mustExec(t, srcConn, "DROP TABLE IF EXISTS `it_sync_sq_src`")
	mustExec(t, srcConn, "CREATE TABLE `it_sync_sq_src` (id INT PRIMARY KEY, name VARCHAR(50), salary INT)")
	mustExec(t, srcConn, "INSERT INTO `it_sync_sq_src` VALUES (1,'alice',100),(2,'bob',200)")

	sqlitePath := filepath.Join(t.TempDir(), "sync_target.sqlite")
	require.NoError(t, os.WriteFile(sqlitePath, nil, 0o644))
	dstConn := conn(t, &dbi.DbInfo{Type: "sqlite", Host: sqlitePath})
	defer dstConn.Close()
	mustExec(t, dstConn, "CREATE TABLE `it_sync_sq_target` (id INTEGER PRIMARY KEY, name TEXT, salary INTEGER)")

	app, _ := newApp()
	task := &entity.DataSyncTask{TargetTableName: "it_sync_sq_target", DuplicateStrategy: 2}
	_, srcRes, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_sq_src` ORDER BY id")
	require.NoError(t, err)
	require.NoError(t, syncBatch(app, srcRes, idNameSalaryMap(), task, dstConn, idNameSalaryCols()))

	mustExec(t, srcConn, "UPDATE `it_sync_sq_src` SET salary = 888 WHERE id = 1")
	_, srcRes2, err := srcConn.Query("SELECT id, name, salary FROM `it_sync_sq_src` ORDER BY id")
	require.NoError(t, err)
	require.NoError(t, syncBatch(app, srcRes2, idNameSalaryMap(), task, dstConn, idNameSalaryCols()))

	_, rows, err := dstConn.Query("SELECT id, name, salary FROM `it_sync_sq_target` ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows, 2, "insert or replace 覆盖后不应产生重复行")
	assert.Equal(t, int64(888), toInt64(rows[0]["salary"]))
}

// TestITSyncParameterizedBinary 参数化直插的二进制保真：字节、空字节串、NULL 与普通文本都应原样落地。
func TestITSyncParameterizedBinary(t *testing.T) {
	c := mysqlConn(t)
	defer c.Close()
	mustExec(t, c, "CREATE TABLE it_bind_src (id INT PRIMARY KEY, payload BLOB, note VARCHAR(30))")
	mustExec(t, c, "CREATE TABLE it_bind_dst LIKE it_bind_src")
	mustExec(t, c, "INSERT INTO it_bind_src VALUES (1,X'00ff005c27','00ff'),(2,X'',''),(3,NULL,NULL)")
	_, rows, err := c.Query("SELECT * FROM it_bind_src ORDER BY id")
	require.NoError(t, err)
	columns, err := c.Metadata().GetColumns("it_bind_dst")
	require.NoError(t, err)
	mapping := []map[string]string{{"src": "id", "target": "id"}, {"src": "payload", "target": "payload"}, {"src": "note", "target": "note"}}
	task := &entity.DataSyncTask{TargetTableName: "it_bind_dst", DuplicateStrategy: dbi.DuplicateStrategyNone}
	require.NoError(t, syncBatch(newApp1(), rows, mapping, task, c, columns))

	_, actual, err := c.Query("SELECT * FROM it_bind_dst ORDER BY id")
	require.NoError(t, err)
	assert.Equal(t, rows, actual, "二进制字节、空字节串、NULL 与普通文本都应保持原值")
}

func newApp1() *sync.DataSyncAppImpl {
	app, _ := newApp()
	return app
}
