package itest

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== GenTruncate 真实执行测试 ==========

func TestSyncGenTruncate_MySQL(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()
	mustExec(t, conn, "DROP TABLE IF EXISTS `sync_trunc_test`")
	mustExec(t, conn, "CREATE TABLE `sync_trunc_test` (`id` int PRIMARY KEY, `name` varchar(64))")
	mustExec(t, conn, "INSERT INTO `sync_trunc_test` VALUES (1, 'a'), (2, 'b'), (3, 'c')")
	rows := readAllRows(t, conn, "sync_trunc_test", "id")
	assert.Len(t, rows, 3)
	sqls := conn.GetDialect().GetSQLGenerator().GenTruncate("sync_trunc_test")
	require.Len(t, sqls, 1)
	mustExec(t, conn, sqls[0])
	rows = readAllRows(t, conn, "sync_trunc_test", "id")
	assert.Len(t, rows, 0)
	mustExec(t, conn, "DROP TABLE IF EXISTS `sync_trunc_test`")
}

func TestSyncGenTruncate_Postgres(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()
	mustExec(t, conn, "DROP TABLE IF EXISTS sync_trunc_test")
	mustExec(t, conn, "CREATE TABLE sync_trunc_test (id int PRIMARY KEY, name varchar(64))")
	mustExec(t, conn, "INSERT INTO sync_trunc_test VALUES (1, 'a'), (2, 'b'), (3, 'c')")
	rows := readAllRows(t, conn, "sync_trunc_test", "id")
	assert.Len(t, rows, 3)
	sqls := conn.GetDialect().GetSQLGenerator().GenTruncate("sync_trunc_test")
	require.Len(t, sqls, 1)
	mustExec(t, conn, sqls[0])
	rows = readAllRows(t, conn, "sync_trunc_test", "id")
	assert.Len(t, rows, 0)
	mustExec(t, conn, "DROP TABLE IF EXISTS sync_trunc_test")
}

func TestSyncGenTruncate_SQLite(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()
	mustExec(t, conn, "CREATE TABLE sync_trunc_test (id int PRIMARY KEY, name text)")
	mustExec(t, conn, "INSERT INTO sync_trunc_test VALUES (1, 'a'), (2, 'b'), (3, 'c')")
	rows := readAllRows(t, conn, "sync_trunc_test", "id")
	assert.Len(t, rows, 3)
	sqls := conn.GetDialect().GetSQLGenerator().GenTruncate("sync_trunc_test")
	require.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "DELETE FROM")
	mustExec(t, conn, sqls[0])
	rows = readAllRows(t, conn, "sync_trunc_test", "id")
	assert.Len(t, rows, 0)
}

// ========== GenBatchDelete 真实执行测试 ==========
// GenBatchDelete 现在生成 NOT IN 语句：删除 NOT IN 给定PK的记录
// 用于硬删除/软删除同步：删除目标表中源库不存在的记录

func TestSyncGenBatchDelete_MySQL(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()
	mustExec(t, conn, "DROP TABLE IF EXISTS `sync_del_test`")
	mustExec(t, conn, "CREATE TABLE `sync_del_test` (`id` int PRIMARY KEY, `name` varchar(64))")
	mustExec(t, conn, "INSERT INTO `sync_del_test` VALUES (1, 'a'), (2, 'b'), (3, 'c'), (4, 'd'), (5, 'e')")
	// GenBatchDelete 生成 NOT IN 语句：删除 NOT IN (2, 4) 的记录，即删除 1, 3, 5
	sqls := conn.GetDialect().GetSQLGenerator().GenBatchDelete("sync_del_test", []string{"id"}, [][]any{{2}, {4}}, nil)
	require.Len(t, sqls, 1)
	mustExec(t, conn, sqls[0])
	rows := readAllRows(t, conn, "sync_del_test", "id")
	assert.Len(t, rows, 2)
	assertCell(t, "id", int64(2), rows[0]["id"])
	assertCell(t, "id", int64(4), rows[1]["id"])
	mustExec(t, conn, "DROP TABLE IF EXISTS `sync_del_test`")
}

func TestSyncGenBatchDelete_Postgres(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()
	mustExec(t, conn, "DROP TABLE IF EXISTS sync_del_test")
	mustExec(t, conn, "CREATE TABLE sync_del_test (id int PRIMARY KEY, name varchar(64))")
	mustExec(t, conn, "INSERT INTO sync_del_test VALUES (1, 'a'), (2, 'b'), (3, 'c'), (4, 'd'), (5, 'e')")
	// GenBatchDelete 生成 NOT IN 语句：删除 NOT IN (2, 4) 的记录，即删除 1, 3, 5
	sqls := conn.GetDialect().GetSQLGenerator().GenBatchDelete("sync_del_test", []string{"id"}, [][]any{{2}, {4}}, nil)
	require.Len(t, sqls, 1)
	mustExec(t, conn, sqls[0])
	rows := readAllRows(t, conn, "sync_del_test", "id")
	assert.Len(t, rows, 2)
	mustExec(t, conn, "DROP TABLE IF EXISTS sync_del_test")
}

func TestSyncGenBatchDelete_SQLite(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()
	mustExec(t, conn, "CREATE TABLE sync_del_test (id int PRIMARY KEY, name text)")
	mustExec(t, conn, "INSERT INTO sync_del_test VALUES (1, 'a'), (2, 'b'), (3, 'c'), (4, 'd'), (5, 'e')")
	// GenBatchDelete 生成 NOT IN 语句：删除 NOT IN (2, 4) 的记录，即删除 1, 3, 5
	sqls := conn.GetDialect().GetSQLGenerator().GenBatchDelete("sync_del_test", []string{"id"}, [][]any{{2}, {4}}, nil)
	require.Len(t, sqls, 1)
	mustExec(t, conn, sqls[0])
	rows := readAllRows(t, conn, "sync_del_test", "id")
	assert.Len(t, rows, 2)
}

// ========== 全量刷新模拟：MySQL → PG ==========

func TestSyncFullRefresh_MySQLToPG(t *testing.T) {
	srcConn := mysqlConn(t)
	defer srcConn.Close()
	targetConn := pgConn(t)
	defer targetConn.Close()

	mustExec(t, srcConn, "DROP TABLE IF EXISTS `sync_src_full`")
	mustExec(t, srcConn, "CREATE TABLE `sync_src_full` (`id` int PRIMARY KEY, `name` varchar(64), `score` decimal(10,2))")
	mustExec(t, srcConn, "INSERT INTO `sync_src_full` VALUES (1, 'alice', 95.5), (2, 'bob', 88.0), (3, 'carl', 72.3)")

	mustExec(t, targetConn, "DROP TABLE IF EXISTS sync_tgt_full")
	mustExec(t, targetConn, "CREATE TABLE sync_tgt_full (id int PRIMARY KEY, name varchar(64), score numeric(10,2))")
	mustExec(t, targetConn, "INSERT INTO sync_tgt_full VALUES (99, 'old_data', 0)")

	// 全量刷新：truncate + insert
	for _, sql := range targetConn.GetDialect().GetSQLGenerator().GenTruncate("sync_tgt_full") {
		mustExec(t, targetConn, sql)
	}

	srcRows := readAllRows(t, srcConn, "sync_src_full", "id")
	require.Len(t, srcRows, 3)

	targetCols := []dbi.Column{
		{ColumnName: "id", DataType: "integer"},
		{ColumnName: "name", DataType: "varchar"},
		{ColumnName: "score", DataType: "numeric"},
	}
	values := make([][]any, 0, len(srcRows))
	for _, row := range srcRows {
		values = append(values, []any{row["id"], row["name"], row["score"]})
	}
	for _, sql := range targetConn.GetDialect().GetSQLGenerator().GenInsert("sync_tgt_full", targetCols, values, dbi.DuplicateStrategyNone, nil) {
		mustExec(t, targetConn, sql)
	}

	targetRows := readAllRows(t, targetConn, "sync_tgt_full", "id")
	assert.Len(t, targetRows, 3)
	assert.Equal(t, "alice", fmt.Sprintf("%v", targetRows[0]["name"]))

	mustExec(t, srcConn, "DROP TABLE IF EXISTS `sync_src_full`")
	mustExec(t, targetConn, "DROP TABLE IF EXISTS sync_tgt_full")
}

// ========== UPSERT 真实执行测试 ==========

func TestSyncUpsert_MySQL(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()
	mustExec(t, conn, "DROP TABLE IF EXISTS `sync_upsert_test`")
	mustExec(t, conn, "CREATE TABLE `sync_upsert_test` (`id` int PRIMARY KEY, `name` varchar(64), `score` int)")
	mustExec(t, conn, "INSERT INTO `sync_upsert_test` VALUES (1, 'alice', 90), (2, 'bob', 80)")

	cols := []dbi.Column{{ColumnName: "id", DataType: "int"}, {ColumnName: "name", DataType: "varchar"}, {ColumnName: "score", DataType: "int"}}
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}, IdentityColumns: []string{"id"}}
	values := [][]any{{1, "alice_updated", 95}, {3, "carl", 70}}
	sqls := conn.GetDialect().GetSQLGenerator().GenInsert("sync_upsert_test", cols, values, dbi.DuplicateStrategyUpdate, meta)
	require.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "ON DUPLICATE KEY UPDATE")
	mustExec(t, conn, sqls[0])

	rows := readAllRows(t, conn, "sync_upsert_test", "id")
	assert.Len(t, rows, 3)
	assert.Equal(t, "alice_updated", fmt.Sprintf("%v", rows[0]["name"]))
	assert.Equal(t, "bob", fmt.Sprintf("%v", rows[1]["name"]))
	assert.Equal(t, "carl", fmt.Sprintf("%v", rows[2]["name"]))
	mustExec(t, conn, "DROP TABLE IF EXISTS `sync_upsert_test`")
}

func TestSyncUpsert_Postgres(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()
	mustExec(t, conn, "DROP TABLE IF EXISTS sync_upsert_test")
	mustExec(t, conn, "CREATE TABLE sync_upsert_test (id int PRIMARY KEY, name varchar(64), score int)")
	mustExec(t, conn, "INSERT INTO sync_upsert_test VALUES (1, 'alice', 90), (2, 'bob', 80)")

	cols := []dbi.Column{{ColumnName: "id", DataType: "integer"}, {ColumnName: "name", DataType: "varchar"}, {ColumnName: "score", DataType: "integer"}}
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}, IdentityColumns: []string{"id"}}
	values := [][]any{{1, "alice_updated", 95}, {3, "carl", 70}}
	sqls := conn.GetDialect().GetSQLGenerator().GenInsert("sync_upsert_test", cols, values, dbi.DuplicateStrategyUpdate, meta)
	require.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "on conflict")
	mustExec(t, conn, sqls[0])

	rows := readAllRows(t, conn, "sync_upsert_test", "id")
	assert.Len(t, rows, 3)
	mustExec(t, conn, "DROP TABLE IF EXISTS sync_upsert_test")
}

func TestSyncUpsert_SQLite(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()
	mustExec(t, conn, "CREATE TABLE sync_upsert_test (id int PRIMARY KEY, name text, score int)")
	mustExec(t, conn, "INSERT INTO sync_upsert_test VALUES (1, 'alice', 90), (2, 'bob', 80)")

	cols := []dbi.Column{{ColumnName: "id", DataType: "integer"}, {ColumnName: "name", DataType: "text"}, {ColumnName: "score", DataType: "integer"}}
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}, IdentityColumns: []string{"id"}}
	values := [][]any{{1, "alice_updated", 95}, {3, "carl", 70}}
	sqls := conn.GetDialect().GetSQLGenerator().GenInsert("sync_upsert_test", cols, values, dbi.DuplicateStrategyUpdate, meta)
	require.NotEmpty(t, sqls)
	for _, sql := range sqls {
		if len(sql) > 0 && sql[0] == 'i' {
			mustExec(t, conn, sql)
		}
	}

	rows := readAllRows(t, conn, "sync_upsert_test", "id")
	assert.Len(t, rows, 3)
}
