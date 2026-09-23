//go:build it

package itest

// sync 集成测试公共底座：真实库连接、断言辅助与最小依赖 fake。
//
// 与业务包的关系：本包为外部黑盒包，仅通过 DataSyncAppImpl 的导出入口
// （SyncTask 单轮同步 / SyncBatch 单批写入 / BuildTargetTableMeta）驱动完整链路，
// 不触及 doDataSync/srcData2TargetDb/buildSyncQuery 等私有编排方法。
// 少数依赖包内私有的生命周期用例（任务终止、endRunning 收尾）仍以同包集成测试形式留在业务目录，
// 见 sync/internal_lifecycle_it_test.go。

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/ititest/scratchclean"
)

// TestMain 复用进程独占临时库，测试结束无条件清理。
func TestMain(m *testing.M) {
	os.Exit(scratchclean.Run(m.Run))
}

func conn(t *testing.T, di *dbi.DbInfo) *dbi.DbConn {
	t.Helper()
	c, err := scratchclean.Conn(context.Background(), di)
	require.NoError(t, err)
	return c
}

func mysqlConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	return conn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
}

func pgConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	return conn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
}

func mustExec(t *testing.T, c *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := c.Exec(sql); err != nil {
		t.Fatalf("exec failed [%s]: %s", sql, err.Error())
	}
}

// fakeRepo 嵌入接口仅覆写 UpdateById：用于断言批级水位持久化次数，未覆写方法被调用即 panic 暴露问题。
type fakeRepo struct {
	repository.DataSyncTask

	updated []*entity.DataSyncTask
}

func (f *fakeRepo) UpdateById(_ context.Context, e *entity.DataSyncTask, _ ...string) error {
	f.updated = append(f.updated, e)
	return nil
}

// newApp 构造仅注入任务仓储的应用实例：SyncTask/SyncBatch 以显式连接驱动，无需 dbApp/logRepo。
func newApp() (*sync.DataSyncAppImpl, *fakeRepo) {
	fake := &fakeRepo{}
	app := &sync.DataSyncAppImpl{}
	app.Repo = fake
	return app, fake
}

func quote(c *dbi.DbConn, name string) string {
	return c.GetDialect().Quoter().QuoteIdent(name)
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case uint64:
		return int64(val)
	case float64:
		return int64(val)
	default:
		n, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(v)), 10, 64)
		return n
	}
}

func countRows(t *testing.T, c *dbi.DbConn, table string) int {
	t.Helper()
	_, rows, err := c.Query(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", quote(c, table)))
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	return int(toInt64(rows[0]["cnt"]))
}

func queryIds(t *testing.T, c *dbi.DbConn, table string) []int64 {
	t.Helper()
	_, rows, err := c.Query(fmt.Sprintf("SELECT id FROM %s ORDER BY id", quote(c, table)))
	require.NoError(t, err)
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, toInt64(r["id"]))
	}
	return ids
}
