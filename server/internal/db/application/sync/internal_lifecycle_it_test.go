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
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/taskx"
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

// TestITIncrementalFieldIndexValidator 私有 P1 校验器的端到端语义：
// 走 validateIncrementalFieldIndex 直接函数，绕开 Save 的 dbApp/cron 依赖注入。
// 三条分支：无索引拒 / 有索引放行 / 逃生阀跳过 / JOIN 场景跳过。
func TestITIncrementalFieldIndexValidator(t *testing.T) {
	src := lcConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	t.Cleanup(func() { src.Close() })
	mustDrop := func(name string) { lcExec(t, src, "DROP TABLE IF EXISTS "+name) }
	mustDrop("it_lc_idx_no")
	mustDrop("it_lc_idx_yes")
	mustDrop("it_lc_idx_comp")
	defer func() { mustDrop("it_lc_idx_no"); mustDrop("it_lc_idx_yes"); mustDrop("it_lc_idx_comp") }()
	lcExec(t, src, "CREATE TABLE it_lc_idx_no (id int primary key, upd_time datetime)")
	lcExec(t, src, "CREATE TABLE it_lc_idx_yes (id int primary key, upd_time datetime, KEY idx_upd (upd_time))")
	lcExec(t, src, "CREATE TABLE it_lc_idx_comp (id int primary key, seq int, upd_time datetime, KEY idx_upd_id (upd_time, seq))")

	newTask := func(table string, skip bool) *entity.DataSyncTask {
		task := &entity.DataSyncTask{
			UpdField: "upd_time",
			DataSQL:  "SELECT id, upd_time FROM " + table,
		}
		task.SetSkipIndexValidation(skip)
		return task
	}
	ctx := context.Background()

	// 无索引 → 拒
	assert.Error(t, ValidateIncrementalFieldIndex(ctx, src, newTask("it_lc_idx_no", false)))
	// 有索引 → 通过
	assert.NoError(t, ValidateIncrementalFieldIndex(ctx, src, newTask("it_lc_idx_yes", false)))
	// 无索引但开逃生阀 → 通过
	assert.NoError(t, ValidateIncrementalFieldIndex(ctx, src, newTask("it_lc_idx_no", true)))
	// JOIN 场景 validator 无法定位单表 → 通过（不误拒）
	joinTask := &entity.DataSyncTask{
		UpdField: "upd_time",
		DataSQL:  "SELECT a.upd_time FROM it_lc_idx_no a JOIN it_lc_idx_yes b ON a.id=b.id",
	}
	assert.NoError(t, ValidateIncrementalFieldIndex(ctx, src, joinTask))

	// 复合索引 (upd_time, seq)：UpdField 命中首列 → pass；UpdField=seq 落在第二列 → reject
	// （回归守卫：validator 早期按整列串等值比较会误拒本用例的 upd_time）
	compFirst := &entity.DataSyncTask{
		UpdField: "upd_time",
		DataSQL:  "SELECT id, upd_time, seq FROM it_lc_idx_comp",
	}
	assert.NoError(t, ValidateIncrementalFieldIndex(ctx, src, compFirst), "复合索引首列命中必须 pass")
	compSecond := &entity.DataSyncTask{
		UpdField: "seq",
		DataSQL:  "SELECT id, upd_time, seq FROM it_lc_idx_comp",
	}
	assert.Error(t, ValidateIncrementalFieldIndex(ctx, src, compSecond), "seq 只是第二列，不满足最左前缀")
}

// TestITCrossInstanceStopTaskAbortsBatch 跨实例停止的端到端语义：
// 本实例持有 runGuard（IsCurrentRun true），他实例调 RequestStop 写 Redis 停止标记；
// 本实例的批次边界靠 `IsCurrentRun || IsStopRequested` 合并判定必须能观察到停止意图并中止。
// 若把 batch check 只留 IsCurrentRun（回归到旧逻辑），本用例首批次边界看到本地 fast path 仍 true，
// 会默默多跑一批 → 断言失败指向该回归。
func TestITCrossInstanceStopTaskAbortsBatch(t *testing.T) {
	src := lcConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	dst := lcConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	t.Cleanup(func() { src.Close(); dst.Close() })
	lcExec(t, src, "DROP TABLE IF EXISTS it_lc_xins_src")
	lcExec(t, dst, "DROP TABLE IF EXISTS it_lc_xins_tgt")
	defer func() {
		lcExec(t, src, "DROP TABLE IF EXISTS it_lc_xins_src")
		lcExec(t, dst, "DROP TABLE IF EXISTS it_lc_xins_tgt")
	}()
	lcExec(t, src, "CREATE TABLE it_lc_xins_src (id int primary key, name varchar(50))")
	lcExec(t, dst, "CREATE TABLE it_lc_xins_tgt (id int primary key, name varchar(50))")
	var vals []string
	for i := 1; i <= 10; i++ {
		vals = append(vals, fmt.Sprintf("(%d,'n%d')", i, i))
	}
	lcExec(t, src, fmt.Sprintf("INSERT INTO it_lc_xins_src VALUES %s", strings.Join(vals, ",")))

	if rediscli.GetCli() == nil {
		t.Skip("Redis not available at 127.0.0.1:6332; cross-instance stop requires it")
	}
	app := &DataSyncAppImpl{}
	app.Repo = &fakeDataSyncRepo{}
	const taskId = uint64(9912)
	// 本实例持有 runGuard 并记录 runId
	runId, ok := app.runGuard.AcquireWithRunId(taskId)
	require.True(t, ok)
	defer app.runGuard.ReleaseWithRunId(taskId, runId)

	// 模拟另一实例发起停止：RequestStop 内部 Release 是 no-op（它没持有），只写 Redis 标记
	remoteStopper := &taskx.RunGuard[uint64]{}
	remoteStopper.RequestStop(taskId)
	defer remoteStopper.ClearStopRequest(taskId)

	task := &entity.DataSyncTask{
		Id: taskId, SyncMode: entity.DataSyncModeIncrementalAppend,
		TargetTableName: "it_lc_xins_tgt", PageSize: 2, UpdField: "id", UpdFieldVal: "0",
		FieldMap: `[{"src":"id","target":"id"},{"src":"name","target":"name"}]`,
	}
	sql := "select id, name from it_lc_xins_src order by id asc"
	syncLog := &entity.DataSyncLog{RunId: runId}
	err := app.doDataSync(context.Background(), sql, task, src, dst, syncLog, &SyncMetrics{})
	require.Error(t, err, "他实例发起停止后，本实例首批边界必须中止")
	assert.Contains(t, err.Error(), "terminated")
	// 首批 PageSize=2 行已提交（批次边界在写完之后检查），后续批次因停止信号中止
	assert.Equal(t, 2, lcCount(t, dst, "it_lc_xins_tgt"), "仅首批 2 行提交，标记读到即中止")
}
