package itest

// oracle 扩展对象（视图/序列）内省的连库回归。
// 与 mssql 同为「非必装实例」：连接不可达则 t.Skip，不影响其余方言回归。
// 验证目标：listSequences 从 ALL_SEQUENCES 填的 attrs（增量/范围/缓存/循环）与真实定义一致，
// 且 ObjectDDL 经 DBMS_METADATA.GET_DDL 能取回 CREATE SEQUENCE。

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/dialect/oracle"
	"mayfly-go/internal/db/ititest/scratchclean"
)

const (
	itOracleUser = "mayfly"
	itOraclePwd  = "Mayfly_123456"
	itOracleSvc  = "FREEPDB1"
)

// oracleDbInfo 构造 Oracle 连接信息：oracle 方言从 Extra.serviceName 取服务名（否则 DSN 报 empty SID and service name）。
func oracleDbInfo() *dbi.DbInfo {
	di := &dbi.DbInfo{
		Type: "oracle", Host: "127.0.0.1", Port: 1521,
		Username: itOracleUser, Password: itOraclePwd, Database: itOracleSvc,
	}
	di.SetExtraValue("serviceName", itOracleSvc)
	return di
}

// oracleConn 连本机 Oracle Free 测试实例；不可达则跳过
func oracleConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	conn, err := dbm.Conn(itCtx(), oracleDbInfo())
	if err != nil {
		scratchclean.SkipOrRequire(t, "oracle", err.Error())
	}
	if err = conn.Ping(); err != nil {
		_ = conn.Close()
		scratchclean.SkipOrRequire(t, "oracle", err.Error())
	}
	return conn
}

func TestITOracleSequenceObjectIntrospection(t *testing.T) {
	conn := oracleConn(t)
	defer conn.Close()
	md := conn.Metadata()

	seq := "IT_" + strings.ToUpper(rand.Text())
	mustExec(t, conn, "CREATE SEQUENCE "+seq+" START WITH 1 INCREMENT BY 5 MINVALUE 1 MAXVALUE 999999 CACHE 7 CYCLE")
	// 仅登记创建成功的独占序列，连接关闭前清理，清理失败使测试失败。
	defer func() {
		_, err := conn.Exec("DROP SEQUENCE " + seq)
		assert.NoError(t, err, "清理测试序列 %s", seq)
	}()

	objs, err := md.ListObjects(itCtx(), "", dbi.KindSequence)
	require.NoError(t, err)
	var found *dbi.MetadataObject
	for i := range objs {
		if strings.EqualFold(objs[i].Name, seq) {
			found = &objs[i]
		}
	}
	require.NotNil(t, found, "ListObjects 应含刚建序列")
	assert.Equal(t, "5", found.Attrs["incrementBy"])
	assert.Equal(t, "999999", found.Attrs["maxValue"])
	assert.Equal(t, "1", found.Attrs["minValue"])
	assert.Equal(t, "true", found.Attrs["isCycle"])

	ddl, err := md.ObjectDDL(itCtx(), "", dbi.KindSequence, seq)
	require.NoError(t, err, "oracle 序列 DDL 经 GET_DDL 应成功")
	assert.Contains(t, strings.ToUpper(ddl), "CREATE SEQUENCE")
	assert.Contains(t, strings.ToUpper(ddl), seq)
}

// TestITOracleTableColumnViewIntrospection 实跑验证 oracle 元数据内省 SQL（ALL_TABLES / ALL_TAB_COLUMNS /
// ALL_CONSTRAINTS / ALL_VIEWS / sys_context）：建一张含主键/非空/精度的表与一个视图，
// 断言 GetTables/GetColumns/GetPrimaryKeys/ListObjects(view) 与真实定义逐项一致。
func TestITOracleTableColumnViewIntrospection(t *testing.T) {
	conn := oracleConn(t)
	defer conn.Close()
	md := conn.Metadata()

	tbl := "IT_META_" + strings.ToUpper(rand.Text()[:8])
	vw := tbl + "_V"
	mustExec(t, conn, "CREATE TABLE "+tbl+" (ID NUMBER(10) PRIMARY KEY, NAME VARCHAR2(50) NOT NULL, AMT NUMBER(12,2), CREATED DATE)")
	mustExec(t, conn, "CREATE VIEW "+vw+" AS SELECT ID, NAME FROM "+tbl)
	defer func() {
		_, _ = conn.Exec("DROP VIEW " + vw)
		_, err := conn.Exec("DROP TABLE " + tbl)
		assert.NoError(t, err, "清理测试表 %s", tbl)
	}()

	tables, err := md.GetTables(tbl)
	require.NoError(t, err)
	require.Len(t, tables, 1, "GetTables(ALL_TABLES) 应能按名取回刚建表")
	assert.Equal(t, tbl, tables[0].TableName)

	cols, err := md.GetColumns(tbl)
	require.NoError(t, err)
	require.Len(t, cols, 4, "ALL_TAB_COLUMNS 应返回 4 列")
	byName := map[string]dbi.Column{}
	for _, c := range cols {
		byName[strings.ToUpper(c.ColumnName)] = c
	}
	id, ok := byName["ID"]
	require.True(t, ok, "应含 ID 列")
	assert.True(t, id.IsPrimaryKey, "ID 应被识别为主键（ALL_CONSTRAINTS）")
	assert.False(t, id.Nullable, "主键列应非空")
	name := byName["NAME"]
	assert.False(t, name.Nullable, "NOT NULL 列应非空")
	assert.Equal(t, 50, name.CharMaxLength, "VARCHAR2(50) 长度应回填")
	amt := byName["AMT"]
	assert.Equal(t, 12, amt.NumPrecision, "NUMBER(12,2) 精度应回填")
	assert.Equal(t, 2, amt.NumScale, "NUMBER(12,2) 标度应回填")

	pks, err := md.GetPrimaryKeys(tbl)
	require.NoError(t, err)
	assert.Equal(t, []string{"ID"}, upperAll(pks), "GetPrimaryKeys 应返回 [ID]")

	views, err := md.ListObjects(itCtx(), "", dbi.KindView)
	require.NoError(t, err)
	found := false
	for i := range views {
		if strings.EqualFold(views[i].Name, vw) {
			found = true
		}
	}
	assert.True(t, found, "ALL_VIEWS 经 ListObjects 应含刚建视图 %s", vw)
}

func upperAll(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.ToUpper(s)
	}
	return out
}
