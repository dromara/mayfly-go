//go:build it

package sync

// 依赖包内私有实现的集成/组件测试（黑盒入口无法覆盖的细粒度生命周期）：
//   - paginateTopN 跨方言分页改写（私有辅助函数）；
//   - 批级同步运行中被手动终止（runGuard 私有守卫）后，已提交批次水位不回退；
//   - endRunning 对遗留「执行中」日志的收尾兜底（Running→Fail）。
// 端到端数据正确性用例已迁至 sync/itest（外部黑盒包）。

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/ititest/scratchclean"
)

func lcConn(t *testing.T, di *dbi.DbInfo) *dbi.DbConn {
	t.Helper()
	c, err := scratchclean.Conn(context.Background(), di)
	require.NoError(t, err)
	return c
}

func lcExec(t *testing.T, c *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := c.Exec(sql); err != nil {
		t.Fatalf("exec failed [%s]: %s", sql, err.Error())
	}
}

func lcCount(t *testing.T, c *dbi.DbConn, table string) int {
	t.Helper()
	_, rows, err := c.Query(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", c.GetDialect().Quoter().Quote(table)))
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	v, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(rows[0]["cnt"])), 10, 64)
	return int(v)
}

// TestITPaginateTopN 验证 paginateTopN 在 MySQL/PG 上的分页改写产物可真实执行且精确返回限定行数。
func TestITPaginateTopN(t *testing.T) {
	tests := []struct {
		name   string
		dbInfo *dbi.DbInfo
	}{
		{"mysql", &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"}},
		{"postgres", &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := lcConn(t, tt.dbInfo)
			defer c.Close()
			baseSQL := "SELECT 1 AS n UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5"
			rewritten, err := paginateTopN(c, baseSQL, 3)
			require.NoError(t, err)
			require.NotEmpty(t, rewritten)
			assert.Contains(t, strings.ToUpper(rewritten), "LIMIT")
			rowCount := 0
			_, err = c.WalkQueryRows(context.Background(), rewritten, func(map[string]any, []*dbi.QueryColumn) error {
				rowCount++
				return nil
			})
			require.NoError(t, err)
			assert.Equal(t, 3, rowCount, "paginateTopN 应精确取前3行")
		})
	}
}

// TestITDataSyncStopMidRunKeepsCommittedWatermark 未获取守卫（模拟 StopTask 已释放）时，
// 首批提交后检测到终止并中断，已提交批次的水位落库且不回退。
func TestITDataSyncStopMidRunKeepsCommittedWatermark(t *testing.T) {
	src := lcConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	dst := lcConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	t.Cleanup(func() { src.Close(); dst.Close() })
	lcExec(t, src, "DROP TABLE IF EXISTS it_lc_stop_src")
	lcExec(t, src, "CREATE TABLE it_lc_stop_src (id int primary key, name varchar(50))")
	lcExec(t, dst, "DROP TABLE IF EXISTS it_lc_stop_tgt")
	lcExec(t, dst, "CREATE TABLE it_lc_stop_tgt (id int primary key, name varchar(50))")
	var vals []string
	for i := 1; i <= 7; i++ {
		vals = append(vals, fmt.Sprintf("(%d,'n%d')", i, i))
	}
	lcExec(t, src, fmt.Sprintf("INSERT INTO it_lc_stop_src VALUES %s", strings.Join(vals, ",")))

	app := &DataSyncAppImpl{}
	fakeRepo := &fakeDataSyncRepo{}
	app.Repo = fakeRepo
	task := &entity.DataSyncTask{
		Id: 9911, TaskName: "it-lc-stop", SyncMode: entity.DataSyncModeIncrementalAppend,
		TargetTableName: "it_lc_stop_tgt", PageSize: 3, UpdField: "id", UpdFieldVal: "0",
		FieldMap: `[{"src":"id","target":"id"},{"src":"name","target":"name"}]`,
	}
	// 不 Acquire：首批（3行）提交后 IsCurrentRun=false → 终止
	sql := "select id, name from it_lc_stop_src where 1=1 order by id asc"
	err := app.doDataSync(context.Background(), sql, task, src, dst, &entity.DataSyncLog{}, &SyncMetrics{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "terminated")
	assert.Equal(t, 3, lcCount(t, dst, "it_lc_stop_tgt"), "仅首批3行应提交")
	assert.Equal(t, "3", task.UpdFieldVal, "内存水位应为已提交批次末行")
	// 关键断言：preempted（本实例无锁）时 persistUpdFieldVal 必须全链路 skip，
	// 否则新接管者推进的水位会被旧任务默默拉回。内存推进但 DB 不落，fakeRepo 不得收到任何 Update。
	assert.Empty(t, fakeRepo.updated, "preempted 任务不得向 DB 写水位")
}

// TestITSyncEarlyFailureFinalizesLog endRunning 收尾：遗留「执行中」日志应兜底置为失败，保留原错误文本。
func TestITSyncEarlyFailureFinalizesLog(t *testing.T) {
	app := &DataSyncAppImpl{}
	app.Repo = &fakeDataSyncRepo{}
	app.dbDataSyncLogRepo = &lcLogRepo{}
	log := &entity.DataSyncLog{Status: entity.DataSyncTaskStateRunning, ErrText: "connection failed"}
	app.endRunning(&entity.DataSyncTask{Id: 1}, log)
	assert.Equal(t, entity.DataSyncTaskStateFail, log.Status)
	assert.Equal(t, "connection failed", log.ErrText)
}

// lcLogRepo 仅覆写 Save（吞掉日志落库），其余方法未覆写；调用即 panic 暴露越界依赖。
type lcLogRepo struct {
	repository.DataSyncLog
}

func (lcLogRepo) Save(context.Context, *entity.DataSyncLog) error { return nil }
