//go:build it

package application

// application 层真实库集成测试的连接助手：与 transfer/sync 包同构（各IT包各自持有连接工厂），
// 直连本地 mysql / postgres / sqlite，用于验证SQL文件执行链路的真实数据库语义

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
)

func appTestConn(t *testing.T, di *dbi.DbInfo) *dbi.DbConn {
	t.Helper()
	conn, err := dbm.Conn(context.Background(), di)
	require.NoError(t, err, "连接本地集成测试数据库失败: %s:%d/%s", di.Type, di.Port, di.Database)
	return conn
}

func appMysqlConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	admin := appTestConn(t, &dbi.DbInfo{
		Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "information_schema",
	})
	defer admin.Close()
	_, err := admin.Exec("CREATE DATABASE IF NOT EXISTS mayfly_dbm_it DEFAULT CHARSET utf8mb4")
	require.NoError(t, err)
	return appTestConn(t, &dbi.DbInfo{
		Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it",
	})
}

func appPgConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	conn := appTestConn(t, &dbi.DbInfo{
		Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it",
	})
	require.NoError(t, conn.Ping())
	return conn
}

func appSqliteConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app_sql_exec_it.sqlite")
	// sqlite方言要求库文件已存在
	require.NoError(t, os.WriteFile(path, nil, 0o644))
	return appTestConn(t, &dbi.DbInfo{Type: "sqlite", Host: path})
}

// appDialectNode 参与互测的方言节点
type appDialectNode struct {
	name string
	conn func(t *testing.T) *dbi.DbConn
	// ddl 建表列定义（id主键 + 文本列）
	ddl string
	// survivedOnTxnBegin 文件内含 BEGIN 且后续语句失败时，本机数据库实测残留行数
	//（提交点由数据库决定，平台不猜测也不改写失败原因，仅固化差异供运维参考）
	survivedOnTxnBegin int64
	// errOnTxnBegin 上述场景下期望出现在错误信息里的底层数据库错误片段
	errOnTxnBegin string
}

var appDialectNodes = []appDialectNode{
	// mysql：BEGIN隐式提交当前事务 → 在途写入落库（残留1行），执行继续到坏语句才失败
	{name: "mysql", conn: appMysqlConn, ddl: "(id int PRIMARY KEY, val varchar(200))", survivedOnTxnBegin: 1, errOnTxnBegin: "it_no_such_table"},
	// postgres：事务内BEGIN仅告警不提交 → 失败可整体回滚（残留0行）
	{name: "pg", conn: appPgConn, ddl: "(id int PRIMARY KEY, val varchar(200))", survivedOnTxnBegin: 0, errOnTxnBegin: "it_no_such_table"},
	// sqlite：事务内BEGIN直接报错 → 在该语句即中断，整体回滚（残留0行）
	{name: "sqlite", conn: appSqliteConn, ddl: "(id INTEGER PRIMARY KEY, val TEXT)", survivedOnTxnBegin: 0, errOnTxnBegin: "cannot start a transaction within a transaction"},
}

const appExecTable = "it_exec_reader"

// appPrepareTable 建一张已提交的空表（DDL不放进待执行文件：mysql的DDL会隐式提交，会干扰回滚断言）
func appPrepareTable(t *testing.T, node appDialectNode) *dbi.DbConn {
	t.Helper()
	conn := node.conn(t)
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec("DROP TABLE IF EXISTS " + quote(appExecTable))
	require.NoError(t, err)
	_, err = conn.Exec("CREATE TABLE " + quote(appExecTable) + " " + node.ddl)
	require.NoError(t, err, "[%s] 建表失败", node.name)
	return conn
}

// appRowCount 读回用例表行数
func appRowCount(t *testing.T, conn *dbi.DbConn) int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query("SELECT COUNT(*) AS cnt FROM " + quote(appExecTable))
	require.NoError(t, err, "[%s] 用例表不可读", conn.Info.Type)
	require.Len(t, rows, 1)
	cnt, ok := dbi.ValToInt64(rows[0]["cnt"])
	require.True(t, ok, "行数取值形态异常: %T", rows[0]["cnt"])
	return cnt
}
