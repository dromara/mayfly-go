package itest

// sqlite 视图内省连库回归：验证 MetadataNavigator 在 sqlite 上真实可用
// （sqlite 本地零依赖可测；dm/mssql/clickhouse 无本地实例，仅编译+护栏校验）。

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

func TestITSQLiteViewIntrospection(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()
	md := conn.Metadata()

	mustExec(t, conn, "CREATE TABLE t_emp (id INTEGER PRIMARY KEY, name TEXT, dept TEXT)")
	mustExec(t, conn, "CREATE VIEW v_eng AS SELECT id, name FROM t_emp WHERE dept='eng'")

	objs, err := md.ListObjects(itCtx(), "", dbi.KindView)
	require.NoError(t, err)
	var found bool
	for _, o := range objs {
		if o.Name == "v_eng" {
			found = true
		}
	}
	assert.True(t, found, "ListObjects(view) 应含 v_eng")

	ddl, err := md.ObjectDDL(itCtx(), "", dbi.KindView, "v_eng")
	require.NoError(t, err, "sqlite 视图 ObjectDDL 应成功")
	assert.Contains(t, strings.ToUpper(ddl), "CREATE VIEW")
	assert.Contains(t, strings.ToLower(ddl), "t_emp")

	// sqlite 无序列：请求 sequence 应回传 ErrUnsupportedKind（能力声明与实现一致）
	_, err = md.ListObjects(itCtx(), "", dbi.KindSequence)
	assert.ErrorIs(t, err, dbi.ErrUnsupportedKind, "sqlite 不应支持序列列举")
}
