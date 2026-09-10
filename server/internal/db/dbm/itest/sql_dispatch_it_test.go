package itest

// 查询/非查询语句分发正确性实证测试：
// 镜像 ExecuteSql 的分发规则（切割→解析→按Stmt类型分发）：
//   - SelectStmt/WithStmt/OtherStmt → Query（读）
//   - InsertStmt/UpdateStmt/DeleteStmt/DdlStmt → Exec（写）
// 重点验证易混淆语句在真实驱动下按该规则执行的兼容性：
//   - OtherStmt（SET/EXPLAIN/PRAGMA等非标准语句）走 Query —— 驱动需返回空/正常结果集而非报错
//   - 前置注释的 SELECT（切割器剥离注释后）不得误判为非查询而走 Exec 导致结果丢失
//   - pg INSERT...RETURNING 走 Exec —— 数据正确插入（RETURNING行不展示为已知行为）
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/dbm/

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestITMysqlDispatchMatrix mysql易混淆语句分发矩阵
func TestITMysqlDispatchMatrix(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	// 类型分类断言（文档化解析器行为）
	assert.Equal(t, "*sqlstmt.OtherStmt", parseStmtType(t, conn, "SET @it_x = 1"))
	assert.Equal(t, "*sqlstmt.OtherStmt", parseStmtType(t, conn, "EXPLAIN select 1"))
	assert.Equal(t, "*sqlstmt.SelectStmt", parseStmtType(t, conn, "SHOW TABLES LIKE 'it_dispatch%'"))
	// 前置注释被切割器剥离，不影响分类
	assert.Equal(t, "*sqlstmt.SelectStmt", parseStmtType(t, conn, "/* header */ select 1"))

	mustExec(t, conn, "DROP TABLE IF EXISTS `it_dispatch_t`")
	mustExec(t, conn, "CREATE TABLE `it_dispatch_t` (id INT PRIMARY KEY, v VARCHAR(50))")

	// 全链路分发执行：OtherStmt(SET/EXPLAIN)→Query、SHOW→Query、注释SELECT→Query、DML→Exec
	execDispatched(t, conn, "SET @it_x = 1")
	execDispatched(t, conn, "EXPLAIN select 1")
	execDispatched(t, conn, "SHOW TABLES LIKE 'it_dispatch%'")
	execDispatched(t, conn, "/* header */ select 1 as v")
	execDispatched(t, conn, "INSERT INTO `it_dispatch_t` VALUES (1, 'a')")
	execDispatched(t, conn, "UPDATE `it_dispatch_t` SET v = 'b' WHERE id = 1")
	execDispatched(t, conn, "DELETE FROM `it_dispatch_t` WHERE id = 1")
	execDispatched(t, conn, "TRUNCATE TABLE `it_dispatch_t`")

	_, rows, err := conn.Query("SELECT COUNT(*) AS cnt FROM `it_dispatch_t`")
	require.NoError(t, err)
	assertCell(t, "cnt", 0, rows[0]["cnt"])
	mustExec(t, conn, "DROP TABLE IF EXISTS `it_dispatch_t`")
}

// TestITPgDispatchMatrix pg易混淆语句分发矩阵
func TestITPgDispatchMatrix(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()

	mustExec(t, conn, "DROP TABLE IF EXISTS it_dispatch_t")
	mustExec(t, conn, "CREATE TABLE it_dispatch_t (id int PRIMARY KEY, v text)")

	// OtherStmt(SET/EXPLAIN)→Query、ANALYZE→DdlStmt→Exec、注释SELECT→Query、DML→Exec
	execDispatched(t, conn, "SET timezone = 'UTC'")
	execDispatched(t, conn, "EXPLAIN select 1")
	execDispatched(t, conn, "/* header */ select 1 as v")
	execDispatched(t, conn, "INSERT INTO it_dispatch_t VALUES (1, 'a')")
	execDispatched(t, conn, "UPDATE it_dispatch_t SET v = 'b' WHERE id = 1")
	execDispatched(t, conn, "ANALYZE it_dispatch_t")

	// INSERT...RETURNING：解析为InsertStmt走Exec，数据正确插入（RETURNING行不展示为已知行为）
	assert.Equal(t, "*sqlstmt.InsertStmt", parseStmtType(t, conn, "INSERT INTO it_dispatch_t VALUES (2, 'r') RETURNING id"))
	execDispatched(t, conn, "INSERT INTO it_dispatch_t VALUES (2, 'r') RETURNING id")
	_, rows, err := conn.Query("SELECT id, v FROM it_dispatch_t ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "b", normalizeDbValue(rows[0]["v"]))
	assert.Equal(t, "r", normalizeDbValue(rows[1]["v"]))

	mustExec(t, conn, "DROP TABLE IF EXISTS it_dispatch_t")
}

// TestITSqliteDispatchMatrix sqlite易混淆语句分发矩阵（沿用pg解析器+标准语义切割器）
func TestITSqliteDispatchMatrix(t *testing.T) {
	conn := sqliteConn(t)

	mustExec(t, conn, "DROP TABLE IF EXISTS `it_dispatch_t`")
	mustExec(t, conn, "CREATE TABLE `it_dispatch_t` (id INTEGER PRIMARY KEY, v TEXT)")

	// PRAGMA/EXPLAIN归OtherStmt→Query（PRAGMA本身返回结果集，Query是正确选择）
	assert.Equal(t, "*sqlstmt.OtherStmt", parseStmtType(t, conn, "PRAGMA table_info('it_dispatch_t')"))
	assert.Equal(t, "*sqlstmt.OtherStmt", parseStmtType(t, conn, "EXPLAIN select 1"))

	execDispatched(t, conn, "PRAGMA table_info('it_dispatch_t')")
	execDispatched(t, conn, "EXPLAIN select 1")
	execDispatched(t, conn, "/* header */ select 1 as v")
	execDispatched(t, conn, "INSERT INTO `it_dispatch_t` VALUES (1, 'a')")
	execDispatched(t, conn, "UPDATE `it_dispatch_t` SET v = 'b' WHERE id = 1")
	execDispatched(t, conn, "DELETE FROM `it_dispatch_t` WHERE id = 1")

	_, rows, err := conn.Query("SELECT COUNT(*) AS cnt FROM `it_dispatch_t`")
	require.NoError(t, err)
	assertCell(t, "cnt", 0, rows[0]["cnt"])
}

// TestITExecSqlViaQueryEffect 实证：exec类SQL（DML/DDL）误用Query执行时数据是否生效
// database/sql语义下大多数驱动会真实执行语句（返回空结果集），但丢失rowsAffected且
// 结果表现为空查询；本用例验证本项目三种方言驱动的真实行为
func TestITExecSqlViaQueryEffect(t *testing.T) {
	// mysql：INSERT/UPDATE/DDL走Query——数据生效、无驱动错误
	mconn := mysqlConn(t)
	defer mconn.Close()
	mustExec(t, mconn, "DROP TABLE IF EXISTS `it_exec_via_query`")
	mustExec(t, mconn, "CREATE TABLE `it_exec_via_query` (id INT PRIMARY KEY, v VARCHAR(50))")
	_, _, err := mconn.Query("INSERT INTO `it_exec_via_query` VALUES (1, 'a')")
	require.NoError(t, err, "mysql驱动对Query执行INSERT应兼容（返回空结果集）")
	_, _, err = mconn.Query("UPDATE `it_exec_via_query` SET v = 'b' WHERE id = 1")
	require.NoError(t, err)
	_, rows, err := mconn.Query("SELECT v FROM `it_exec_via_query` WHERE id = 1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "b", normalizeDbValue(rows[0]["v"]), "Query执行INSERT/UPDATE后数据应真实生效")
	mustExec(t, mconn, "DROP TABLE IF EXISTS `it_exec_via_query`")

	// pg：INSERT走Query——数据生效
	pconn := pgConn(t)
	defer pconn.Close()
	mustExec(t, pconn, "DROP TABLE IF EXISTS it_exec_via_query")
	mustExec(t, pconn, "CREATE TABLE it_exec_via_query (id int PRIMARY KEY, v text)")
	_, _, err = pconn.Query("INSERT INTO it_exec_via_query VALUES (1, 'a')")
	require.NoError(t, err, "pg驱动对Query执行INSERT应兼容")
	_, rows, err = pconn.Query("SELECT v FROM it_exec_via_query WHERE id = 1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "a", normalizeDbValue(rows[0]["v"]))
	mustExec(t, pconn, "DROP TABLE IF EXISTS it_exec_via_query")

	// sqlite：INSERT走Query——数据生效
	sconn := sqliteConn(t)
	mustExec(t, sconn, "DROP TABLE IF EXISTS `it_exec_via_query`")
	mustExec(t, sconn, "CREATE TABLE `it_exec_via_query` (id INTEGER PRIMARY KEY, v TEXT)")
	_, _, err = sconn.Query("INSERT INTO `it_exec_via_query` VALUES (1, 'a')")
	require.NoError(t, err, "sqlite驱动对Query执行INSERT应兼容")
	_, rows, err = sconn.Query("SELECT v FROM `it_exec_via_query` WHERE id = 1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "a", normalizeDbValue(rows[0]["v"]))
}

// TestITFallbackSelectWithLeadingComment 前置注释下的语句类型兜底判定：
// 切割保留注释原文（Oracle hint / mysql 可执行注释不丢语义），故解析失败时
// 不能按字符前缀匹配，必须用方言感知的首关键字（跳过前导空白与注释）
func TestITFallbackSelectWithLeadingComment(t *testing.T) {
	splitter := sqliteConn(t).GetDialect().GetSQLSplitter()
	var stmts []string
	require.NoError(t, splitter.SplitSQL(strings.NewReader("/* header */ select 1; update t set a = 1;"), func(s string) error {
		stmts = append(stmts, s)
		return nil
	}))
	require.Len(t, stmts, 2)
	// 注释原文保留，且未被改写
	assert.Equal(t, "/* header */ select 1", stmts[0])
	assert.Equal(t, "select", splitter.LeadingKeyword(stmts[0]), "首关键字必须跳过前导注释")
	assert.Equal(t, "update", splitter.LeadingKeyword(stmts[1]))
}
