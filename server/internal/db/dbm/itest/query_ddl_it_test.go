package itest

// 查询类/非查询类SQL与元信息集成测试——补齐业务流程覆盖缺口：
//   - 复杂查询：JOIN/聚合GROUP BY/ORDER BY/LIMIT/子查询/表达式列（前端SQL控制台高频场景）
//   - 非查询DDL变体：TRUNCATE/ALTER TABLE/CREATE INDEX（此前仅测过基础CRUD与CREATE TABLE）
//   - 索引元信息：GetTableIndex回读（此前无任何索引metadata断言）
//   - sqlite连接与库信息、列metadata回读（此前sqlite缺独立覆盖）
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/dbm/

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

var complexQueryRows = [][]any{
	{1, "alice", "dev", 100},
	{2, "bob", "dev", 200},
	{3, "carol", "ops", 300},
	{4, "dave", "ops", 400},
}

// TestITMysqlComplexQuery mysql复杂查询：JOIN/聚合/子查询/ORDER BY/LIMIT 真实执行与结果断言
func TestITMysqlComplexQuery(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	mustExec(t, conn, "DROP TABLE IF EXISTS `it_q_emp`,`it_q_dept`")
	mustExec(t, conn, "CREATE TABLE `it_q_emp` (id INT PRIMARY KEY, name VARCHAR(50), dept VARCHAR(20), salary INT)")
	mustExec(t, conn, "CREATE TABLE `it_q_dept` (dept VARCHAR(20) PRIMARY KEY, dept_name VARCHAR(50))")
	writeByGenInsert(t, conn, "it_q_emp", []dbi.Column{{ColumnName: "id"}, {ColumnName: "name"}, {ColumnName: "dept"}, {ColumnName: "salary"}}, complexQueryRows)
	mustExec(t, conn, "INSERT INTO `it_q_dept` VALUES ('dev','开发部'),('ops','运维部')")

	// 多表JOIN + 聚合 + ORDER BY + LIMIT：解析为查询并真实执行
	joinSql := "SELECT e.dept, d.dept_name, COUNT(*) AS cnt, SUM(e.salary) AS total " +
		"FROM `it_q_emp` e JOIN `it_q_dept` d ON e.dept = d.dept " +
		"GROUP BY e.dept, d.dept_name ORDER BY total DESC LIMIT 1"
	assert.Equal(t, "*sqlstmt.SelectStmt", parseStmtType(t, conn, joinSql))
	_, rows, err := conn.Query(joinSql)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "ops", normalizeDbValue(rows[0]["dept"]))
	assert.Equal(t, int64(700), toInt64(rows[0]["total"]))

	// 子查询
	subSql := "SELECT name FROM `it_q_emp` WHERE salary > (SELECT AVG(salary) FROM `it_q_emp`) ORDER BY id"
	_, subRows, err := conn.Query(subSql)
	require.NoError(t, err)
	require.Len(t, subRows, 2)
	assert.Equal(t, "carol", normalizeDbValue(subRows[0]["name"]))

	// LEFT JOIN 计数（含无匹配行）
	leftSql := "SELECT d.dept_name, COUNT(e.id) AS cnt FROM `it_q_dept` d LEFT JOIN `it_q_emp` e ON e.dept = d.dept GROUP BY d.dept_name ORDER BY d.dept_name"
	_, leftRows, err := conn.Query(leftSql)
	require.NoError(t, err)
	require.Len(t, leftRows, 2)
}

// TestITPgComplexQuery pg复杂查询（含 ::类型转换、ILIKE 等 pg 特色语法）
func TestITPgComplexQuery(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()

	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote("it_q_emp")+", "+quote("it_q_dept"))
	mustExec(t, conn, "CREATE TABLE "+quote("it_q_emp")+" (id INT PRIMARY KEY, name VARCHAR(50), dept VARCHAR(20), salary INT)")
	mustExec(t, conn, "CREATE TABLE "+quote("it_q_dept")+" (dept VARCHAR(20) PRIMARY KEY, dept_name VARCHAR(50))")
	mustExec(t, conn, "INSERT INTO "+quote("it_q_emp")+" VALUES (1,'alice','dev',100),(2,'bob','dev',200),(3,'carol','ops',300),(4,'dave','ops',400)")
	mustExec(t, conn, "INSERT INTO "+quote("it_q_dept")+" VALUES ('dev','开发部'),('ops','运维部')")

	joinSql := "SELECT e.dept, d.dept_name, COUNT(*) AS cnt, SUM(e.salary) AS total " +
		"FROM it_q_emp e JOIN it_q_dept d ON e.dept = d.dept " +
		"GROUP BY e.dept, d.dept_name ORDER BY total DESC LIMIT 1"
	assert.Equal(t, "*sqlstmt.SelectStmt", parseStmtType(t, conn, joinSql))
	_, rows, err := conn.Query(joinSql)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "ops", normalizeDbValue(rows[0]["dept"]))
	assert.Equal(t, int64(700), toInt64(rows[0]["total"]))

	// pg特色：::cast + ILIKE
	castSql := "SELECT name FROM it_q_emp WHERE dept::text ILIKE 'D%' ORDER BY id"
	_, castRows, err := conn.Query(castSql)
	require.NoError(t, err)
	require.Len(t, castRows, 2)

	// CTE（WithStmt分发）
	withSql := "WITH rich AS (SELECT * FROM it_q_emp WHERE salary > 250) SELECT name FROM rich ORDER BY id"
	assert.Equal(t, "*sqlstmt.WithStmt", parseStmtType(t, conn, withSql))
	_, withRows, err := conn.Query(withSql)
	require.NoError(t, err)
	require.Len(t, withRows, 2)
}

// TestITSqliteComplexQuery sqlite复杂查询
func TestITSqliteComplexQuery(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()

	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote("it_q_emp"))
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote("it_q_dept"))
	mustExec(t, conn, "CREATE TABLE "+quote("it_q_emp")+" (id INTEGER PRIMARY KEY, name TEXT, dept TEXT, salary INTEGER)")
	mustExec(t, conn, "CREATE TABLE "+quote("it_q_dept")+" (dept TEXT PRIMARY KEY, dept_name TEXT)")
	mustExec(t, conn, "INSERT INTO "+quote("it_q_emp")+" VALUES (1,'alice','dev',100),(2,'bob','dev',200),(3,'carol','ops',300),(4,'dave','ops',400)")
	mustExec(t, conn, "INSERT INTO "+quote("it_q_dept")+" VALUES ('dev','开发部'),('ops','运维部')")

	joinSql := "SELECT e.dept, d.dept_name, COUNT(*) AS cnt, SUM(e.salary) AS total " +
		"FROM it_q_emp e JOIN it_q_dept d ON e.dept = d.dept " +
		"GROUP BY e.dept, d.dept_name ORDER BY total DESC LIMIT 1"
	assert.Equal(t, "*sqlstmt.SelectStmt", parseStmtType(t, conn, joinSql))
	_, rows, err := conn.Query(joinSql)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "ops", normalizeDbValue(rows[0]["dept"]))
	assert.Equal(t, int64(700), toInt64(rows[0]["total"]))

	// 子查询 + LIMIT
	subSql := "SELECT name FROM it_q_emp WHERE salary > (SELECT AVG(salary) FROM it_q_emp) ORDER BY id LIMIT 2"
	_, subRows, err := conn.Query(subSql)
	require.NoError(t, err)
	require.Len(t, subRows, 2)
}

// TestITMysqlNonQueryDdlVariants mysql非查询DDL变体 + 索引元信息回读
func TestITMysqlNonQueryDdlVariants(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	mustExec(t, conn, "DROP TABLE IF EXISTS `it_ddl_v`")
	mustExec(t, conn, "CREATE TABLE `it_ddl_v` (id INT PRIMARY KEY, val VARCHAR(100))")
	mustExec(t, conn, "INSERT INTO `it_ddl_v` VALUES (1,'a'),(2,'b')")

	// TRUNCATE（真实清空；解析可能走OtherStmt/笡底路径，不做严格类型断言）
	_, err := conn.Exec("TRUNCATE TABLE `it_ddl_v`")
	require.NoError(t, err)
	_, rows, err := conn.Query("SELECT COUNT(*) AS cnt FROM `it_ddl_v`")
	require.NoError(t, err)
	assert.Equal(t, int64(0), toInt64(rows[0]["cnt"]))

	// ALTER TABLE ADD COLUMN
	mustExec(t, conn, "ALTER TABLE `it_ddl_v` ADD COLUMN remark VARCHAR(50) DEFAULT 'n/a'")
	cols, err := conn.GetMetadata().GetColumns("it_ddl_v")
	require.NoError(t, err)
	found := false
	for _, c := range cols {
		if c.ColumnName == "remark" {
			found = true
		}
	}
	assert.True(t, found, "ALTER ADD COLUMN后metadata应含remark列")

	// CREATE INDEX + 索引元信息回读
	mustExec(t, conn, "CREATE INDEX idx_it_ddl_v_val ON `it_ddl_v` (val)")
	indexes, err := conn.GetMetadata().GetTableIndex("it_ddl_v")
	require.NoError(t, err)
	require.NotEmpty(t, indexes, "应回读到索引")
	// 注：mysql实现中meta SQL显式排除了PRIMARY主键索引（index_name != 'PRIMARY'），
	// 故IsPrimaryKey恒为false，此处不断言主键标记
	var hasVal bool
	for _, idx := range indexes {
		if idx.IndexName == "idx_it_ddl_v_val" && idx.ColumnName == "val" {
			hasVal = true
		}
	}
	assert.True(t, hasVal, "应回读到新建的普通索引")
}

// TestITPgNonQueryDdlVariants pg非查询DDL变体 + 索引元信息回读
func TestITPgNonQueryDdlVariants(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()

	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote("it_ddl_v"))
	mustExec(t, conn, "CREATE TABLE "+quote("it_ddl_v")+" (id INT PRIMARY KEY, val VARCHAR(100))")
	mustExec(t, conn, "INSERT INTO "+quote("it_ddl_v")+" VALUES (1,'a'),(2,'b')")

	// TRUNCATE
	_, err := conn.Exec("TRUNCATE TABLE " + quote("it_ddl_v"))
	require.NoError(t, err)
	_, rows, err := conn.Query("SELECT COUNT(*) AS cnt FROM it_ddl_v")
	require.NoError(t, err)
	assert.Equal(t, int64(0), toInt64(rows[0]["cnt"]))

	// ALTER TABLE
	mustExec(t, conn, "ALTER TABLE "+quote("it_ddl_v")+" ADD COLUMN remark VARCHAR(50) DEFAULT 'n/a'")
	cols, err := conn.GetMetadata().GetColumns("it_ddl_v")
	require.NoError(t, err)
	found := false
	for _, c := range cols {
		if c.ColumnName == "remark" {
			found = true
		}
	}
	assert.True(t, found, "ALTER ADD COLUMN后metadata应含remark列")

	// CREATE INDEX + 元信息回读
	mustExec(t, conn, "CREATE INDEX idx_it_ddl_v_val ON "+quote("it_ddl_v")+" (val)")
	indexes, err := conn.GetMetadata().GetTableIndex("it_ddl_v")
	require.NoError(t, err)
	require.NotEmpty(t, indexes, "应回读到索引")
	var hasVal bool
	for _, idx := range indexes {
		if idx.IndexName == "idx_it_ddl_v_val" && idx.ColumnName == "val" {
			hasVal = true
		}
	}
	assert.True(t, hasVal, "应回读到新建的普通索引")
}

// TestITSqliteConnectAndMetadata sqlite连接信息、库列表、表与列metadata回读
func TestITSqliteConnectAndMetadata(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()

	// 数据库服务信息（版本号非空）
	server, err := conn.GetMetadata().GetDbServer()
	require.NoError(t, err)
	require.NotNil(t, server)
	assert.NotEmpty(t, server.Version, "sqlite版本应非空")

	// 库名列表（sqlite为单文件库）
	dbNames, err := conn.GetMetadata().GetDbNames()
	require.NoError(t, err)
	require.NotEmpty(t, dbNames)

	// 建表后表列表与列metadata回读
	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote("it_meta"))
	mustExec(t, conn, "CREATE TABLE "+quote("it_meta")+" (id INTEGER PRIMARY KEY, name TEXT NOT NULL, score REAL DEFAULT 0)")
	tables, err := conn.GetMetadata().GetTables("it_meta")
	require.NoError(t, err)
	require.NotEmpty(t, tables, "应回读到it_meta表")

	cols, err := conn.GetMetadata().GetColumns("it_meta")
	require.NoError(t, err)
	require.Len(t, cols, 3)
	byName := make(map[string]dbi.Column)
	for _, c := range cols {
		byName[c.ColumnName] = c
	}
	assert.Contains(t, byName, "id")
	assert.Contains(t, byName, "name")
	assert.Contains(t, byName, "score")
}

// toInt64 数值归一（sqlite/mysql驱动可能返回int64/string等）
func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case []byte:
		n := int64(0)
		for _, b := range val {
			if b < '0' || b > '9' {
				return n
			}
			n = n*10 + int64(b-'0')
		}
		return n
	case string:
		n := int64(0)
		for _, c := range val {
			if c < '0' || c > '9' {
				return n
			}
			n = n*10 + int64(c-'0')
		}
		return n
	}
	return 0
}
