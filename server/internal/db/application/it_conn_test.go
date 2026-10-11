//go:build it

package application

// 父包真实库集成测试的连接助手：与 sqlexec/it_conn_test.go 同构（各 IT 包各自持有连接工厂）。
// 父包留守的 IT 只有列值采样（mysql/pg）与数据导入（sqlite），故这里只提供它们所需的连接工厂；
// exec/review/rollback/export 的方言节点与建表/行数助手随 App 一起留在 sqlexec，不在两包间共享。

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/sqlexec"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/ititest/scratchclean"
	"mayfly-go/pkg/ioc"
)

// TestMain 装配数据导入 IT 所需的最小 IOC。
//
// 数据导入经 GetDbSQLExecApp() 取策略闸门（见 db_data_import.go），IT 进程不跑完整 IOC 初始化，
// 所以这里只注册一个满足 sqlexec.DbSQLExec 的本地替身：嵌入接口 + 只覆写 CheckSqlsWithoutTicket，
// 越界用到别的方法会直接 panic 而不是静默放行。
func TestMain(m *testing.M) {
	ioc.Register(&itSqlExecBean{})
	os.Exit(scratchclean.Run(m.Run))
}

// itSqlExecBean 父包 IT 用的最小执行应用替身：只实现数据导入链路真正用到的判定方法，
// 其余方法由嵌入的接口占位（被误用即 panic）。
type itSqlExecBean struct {
	sqlexec.DbSQLExec
}

// CheckSqlsWithoutTicket 复刻「未绑定流程定义」时的真实结论：每条语句一个空提醒、不阻断
// （见 sqlexec 的 checkSqlsWithoutTicket 在 procdef==nil 分支返回 make([]string, len(sqls))）。
// IT 的 sqlite 库没有绑定流程定义，故用这一确定结论替代真实实现——父包无法构造 sqlexec 实现的未导出依赖。
func (b *itSqlExecBean) CheckSqlsWithoutTicket(_ context.Context, _ *dbi.DbConn, sqls []string) ([]string, error) {
	return make([]string, len(sqls)), nil
}

func appTestConn(t *testing.T, di *dbi.DbInfo) *dbi.DbConn {
	t.Helper()
	conn, err := scratchclean.Conn(context.Background(), di)
	require.NoError(t, err, "连接本地集成测试数据库失败: %s:%d/%s", di.Type, di.Port, di.Database)
	return conn
}

func appMysqlConn(t *testing.T) *dbi.DbConn {
	t.Helper()
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

func appSQLiteConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app_data_import_it.sqlite")
	// sqlite方言要求库文件已存在
	require.NoError(t, os.WriteFile(path, nil, 0o644))
	return appTestConn(t, &dbi.DbInfo{Type: "sqlite", Host: path})
}
