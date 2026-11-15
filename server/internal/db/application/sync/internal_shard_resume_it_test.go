//go:build it

package sync

// 片内续扫（查询存活预算 + 键集续扫）与「分片键可由唯一索引提供」的真实库用例。
//
// 为什么必须在包里测：续扫的触发条件是「单条源查询的存活时长超过预算」，预算是一个包内变量
// （shardQueryBudget），只有同包测试才能把它压到纳秒级、让每条查询在第一个检查点就断，
// 从而在不等待真实超时的前提下把整条续扫链路跑穿。外部黑盒包只能等 30s，等于不测。
//
// 断言的重点不是「有没有还有续扫」，而是两条不可让渡的性质：
//  1. 续扫前后**一行不丢、一行不重**（游标停在已写行、当前行重读一次恰好写一次）；
//  2. 分片键不依赖「必须声明主键」：单列整型唯一索引同样可分片（数仓 Unique Key 表形态），
//     而普通索引（不保证唯一）必须拒分片——否则 `key > 游标` 会静默漏行。

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

const (
	resumeSrcTable = "it_sync_resume_src"
	resumeTgtTable = "it_sync_resume_tgt"
	resumeRows     = 1200
)

// resumeFill 批量灌入 id 1..rows 的夹具（单条 INSERT 会被 max_allowed_packet 卡住）
func resumeFill(t *testing.T, c *dbi.DbConn, table string, rows int) {
	t.Helper()
	for start := 1; start <= rows; start += 500 {
		end := min(start+499, rows)
		values := make([]string, 0, end-start+1)
		for id := start; id <= end; id++ {
			values = append(values, fmt.Sprintf("(%d,'n%d')", id, id))
		}
		lcExec(t, c, fmt.Sprintf("INSERT INTO %s (id, name) VALUES %s", table, strings.Join(values, ",")))
	}
}

// TestITShardResumeOnQueryBudget 预算耗尽后按分片键续扫：分片区间被切成多段查询，
// 目标端仍必须恰好是全量、无重。
func TestITShardResumeOnQueryBudget(t *testing.T) {
	src := lcConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	dst := lcConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	t.Cleanup(func() { src.Close(); dst.Close() })
	lcExec(t, src, "DROP TABLE IF EXISTS "+resumeSrcTable)
	lcExec(t, dst, "DROP TABLE IF EXISTS "+resumeTgtTable)
	defer func() {
		lcExec(t, src, "DROP TABLE IF EXISTS "+resumeSrcTable)
		lcExec(t, dst, "DROP TABLE IF EXISTS "+resumeTgtTable)
	}()
	lcExec(t, src, "CREATE TABLE "+resumeSrcTable+" (id int primary key, name varchar(50))")
	lcExec(t, dst, "CREATE TABLE "+resumeTgtTable+" (id int primary key, name varchar(50))")
	resumeFill(t, src, resumeSrcTable, resumeRows)

	// 预算压到 1ns：每条查询在第一个检查点（= 一个批次的行数）就超预算并续扫
	oldBudget := shardQueryBudget
	shardQueryBudget = time.Nanosecond
	t.Cleanup(func() { shardQueryBudget = oldBudget })

	app := &DataSyncAppImpl{}
	app.Repo = &fakeDataSyncRepo{}
	const taskId = uint64(9931)
	runId, ok := app.runGuard.AcquireWithRunId(taskId)
	require.True(t, ok)
	defer app.runGuard.ReleaseWithRunId(taskId, runId)

	task := &entity.DataSyncTask{
		Id: taskId, TaskName: "it-shard-resume", SyncMode: entity.DataSyncModeIncrementalAppend,
		TargetTableName: resumeTgtTable, PageSize: 100, UpdField: "id", UpdFieldVal: "0",
		FieldMap: `[{"src":"id","target":"id"},{"src":"name","target":"name"}]`,
		DataSQL:  "select id, name from " + resumeSrcTable,
	}
	task.SetShardTargetRows(500) // 1200 行 → 多片，每片 ≥ 一个检查点

	syncLog := &entity.DataSyncLog{RunId: runId}
	plan := newSyncQueryPlan(task.DataSQL, "where 1=1", "", "order by id asc")
	require.NoError(t, app.doDataSync(context.Background(), plan, task, src, dst, syncLog, &SyncMetrics{}), syncLog.ErrText)

	assert.Contains(t, syncLog.RunLog, "按分片键 [id]", "应走分片扫描")
	assert.Contains(t, syncLog.RunLog, "↻ 片内查询超预算", "预算压到 1ns 后必须发生续扫")
	assert.Contains(t, syncLog.RunLog, "`id` >", "续扫谓词必须是分片键的严格大于")
	assert.Equal(t, 1200, lcCount(t, dst, resumeTgtTable), "续扫不得丢行也不得少写")
	_, rows, err := dst.Query(fmt.Sprintf("SELECT COUNT(*) AS c, COUNT(DISTINCT id) AS d, MIN(id) AS mn, MAX(id) AS mx FROM %s", resumeTgtTable))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "1200", fmt.Sprint(rows[0]["c"]))
	assert.Equal(t, "1200", fmt.Sprint(rows[0]["d"]), "主键去重后仍应是 1200：一行都不能重复")
	assert.Equal(t, "1", fmt.Sprint(rows[0]["mn"]))
	assert.Equal(t, "1200", fmt.Sprint(rows[0]["mx"]))
	assert.Equal(t, "1200", task.UpdFieldVal, "水位应推进到全表最大 id")
}

// TestITShardKeyFromUniqueIndex 无主键、但有单列整型唯一索引的表也能分片；
// 只有普通索引（不保证唯一）时必须拒分片并给出原因。
func TestITShardKeyFromUniqueIndex(t *testing.T) {
	src := lcConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	dst := lcConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	t.Cleanup(func() { src.Close(); dst.Close() })
	cases := []struct {
		name      string
		srcTable  string
		ddl       string
		wantShard bool
	}{
		{
			name:      "unique index without primary key",
			srcTable:  "it_sync_uk_src",
			ddl:       "id int not null, name varchar(50), UNIQUE KEY uk_id (id)",
			wantShard: true,
		},
		{
			name:      "plain index is not unique",
			srcTable:  "it_sync_plain_src",
			ddl:       "id int not null, name varchar(50), KEY idx_id (id)",
			wantShard: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tgt := "it_sync_uk_tgt"
			lcExec(t, src, "DROP TABLE IF EXISTS "+tc.srcTable)
			lcExec(t, dst, "DROP TABLE IF EXISTS "+tgt)
			defer func() {
				lcExec(t, src, "DROP TABLE IF EXISTS "+tc.srcTable)
				lcExec(t, dst, "DROP TABLE IF EXISTS "+tgt)
			}()
			lcExec(t, src, "CREATE TABLE "+tc.srcTable+" ("+tc.ddl+")")
			lcExec(t, dst, "CREATE TABLE "+tgt+" (id int primary key, name varchar(50))")
			resumeFill(t, src, tc.srcTable, resumeRows)

			app := &DataSyncAppImpl{}
			app.Repo = &fakeDataSyncRepo{}
			taskId := uint64(9941)
			runId, ok := app.runGuard.AcquireWithRunId(taskId)
			require.True(t, ok)
			defer app.runGuard.ReleaseWithRunId(taskId, runId)

			task := &entity.DataSyncTask{
				Id: taskId, TaskName: "it-shard-key", SyncMode: entity.DataSyncModeIncrementalAppend,
				TargetTableName: tgt, PageSize: 100, UpdField: "id", UpdFieldVal: "0",
				FieldMap: `[{"src":"id","target":"id"},{"src":"name","target":"name"}]`,
				DataSQL:  "select id, name from " + tc.srcTable,
			}
			task.SetShardTargetRows(500)
			syncLog := &entity.DataSyncLog{RunId: runId}
			plan := newSyncQueryPlan(task.DataSQL, "where 1=1", "", "order by id asc")
			require.NoError(t, app.doDataSync(context.Background(), plan, task, src, dst, syncLog, &SyncMetrics{}), syncLog.ErrText)

			if tc.wantShard {
				assert.Contains(t, syncLog.RunLog, "按分片键 [id]", "单列整型唯一索引应可作为分片键")
				assert.Contains(t, syncLog.RunLog, "分 2 片", "1200 行 / 单片 500 行应切出多片")
			} else {
				assert.Contains(t, syncLog.RunLog, "整表单次查询", "普通索引不保证唯一，不得分片")
				assert.Contains(t, syncLog.RunLog, "唯一索引", "日志必须给出「为什么没分片」")
			}
			assert.Equal(t, resumeRows, lcCount(t, dst, tgt), "无论分片与否，数据必须完整落库")
		})
	}
}

// TestITShardResumeWithFilterAndAliasKey 续扫 × 过滤 × 别名分片键三者叠加：
// 被过滤掉的行也必须推进游标（否则续扫会把它们重读，虽无害但会掩盖游标停滞），
// 分片键带别名时游标必须从别名列取值，而续扫谓词仍用真实列名。
func TestITShardResumeWithFilterAndAliasKey(t *testing.T) {
	src := lcConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	dst := lcConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	t.Cleanup(func() { src.Close(); dst.Close() })
	const srcT, tgtT = "it_sync_resume_f_src", "it_sync_resume_f_tgt"
	lcExec(t, src, "DROP TABLE IF EXISTS "+srcT)
	lcExec(t, dst, "DROP TABLE IF EXISTS "+tgtT)
	defer func() {
		lcExec(t, src, "DROP TABLE IF EXISTS "+srcT)
		lcExec(t, dst, "DROP TABLE IF EXISTS "+tgtT)
	}()
	lcExec(t, src, "CREATE TABLE "+srcT+" (id int primary key, name varchar(50), keep int)")
	lcExec(t, dst, "CREATE TABLE "+tgtT+" (id int primary key, name varchar(50), keep int)")
	for start := 1; start <= 2000; start += 500 {
		end := min(start+499, 2000)
		values := make([]string, 0, end-start+1)
		for id := start; id <= end; id++ {
			values = append(values, fmt.Sprintf("(%d,'n%d',%d)", id, id, id%3))
		}
		lcExec(t, src, fmt.Sprintf("INSERT INTO %s (id, name, keep) VALUES %s", srcT, strings.Join(values, ",")))
	}

	oldBudget := shardQueryBudget
	shardQueryBudget = time.Nanosecond
	t.Cleanup(func() { shardQueryBudget = oldBudget })

	app := &DataSyncAppImpl{}
	app.Repo = &fakeDataSyncRepo{}
	const taskId = uint64(9932)
	runId, ok := app.runGuard.AcquireWithRunId(taskId)
	require.True(t, ok)
	defer app.runGuard.ReleaseWithRunId(taskId, runId)

	task := &entity.DataSyncTask{
		Id: taskId, TaskName: "it-shard-resume-filter", SyncMode: entity.DataSyncModeIncrementalAppend,
		TargetTableName: tgtT, PageSize: 100, UpdField: "pk_id", UpdFieldVal: "0",
		FilterCondition: "keep == 1",
		FieldMap:        `[{"src":"pk_id","target":"id"},{"src":"name","target":"name"},{"src":"keep","target":"keep"}]`,
		DataSQL:         "select id as pk_id, name, keep from " + srcT,
	}
	task.SetShardTargetRows(1000)
	syncLog := &entity.DataSyncLog{RunId: runId}
	plan := newSyncQueryPlan(task.DataSQL, "where 1=1", "", "order by pk_id asc")
	require.NoError(t, app.doDataSync(context.Background(), plan, task, src, dst, syncLog, &SyncMetrics{}), syncLog.ErrText)

	assert.Contains(t, syncLog.RunLog, "↻ 片内查询超预算", "过滤+别名场景仍必须发生续扫")
	// keep==1 的行是 id%3==1：1,4,...,1999 共 667 行
	assert.Equal(t, 667, lcCount(t, dst, tgtT), "只应落过滤后的行，且续扫不得丢行")
	_, rows, err := dst.Query(fmt.Sprintf("SELECT COUNT(DISTINCT id) AS d, MIN(id) AS mn, MAX(id) AS mx, SUM(CASE WHEN id=2 THEN 1 ELSE 0 END) AS filtered FROM %s", tgtT))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "667", fmt.Sprint(rows[0]["d"]), "不得重复")
	assert.Equal(t, "1", fmt.Sprint(rows[0]["mn"]))
	assert.Equal(t, "1999", fmt.Sprint(rows[0]["mx"]))
	assert.Equal(t, "0", fmt.Sprint(rows[0]["filtered"]), "被过滤的行不得落库")
	assert.Equal(t, "1999", task.UpdFieldVal, "水位应取过滤后行里的最大键值")
}

// TestITShardResumeNullKeyShardNotTruncated NULL 键片不可续扫（NULL 之间无 `>` 顺序），
// 因此预算到点也不得中断它——否则该片剩余行会被静默丢掉。
func TestITShardResumeNullKeyShardNotTruncated(t *testing.T) {
	src := lcConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	dst := lcConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	t.Cleanup(func() { src.Close(); dst.Close() })
	const srcT, tgtT = "it_sync_resume_n_src", "it_sync_resume_n_tgt"
	lcExec(t, src, "DROP TABLE IF EXISTS "+srcT)
	lcExec(t, dst, "DROP TABLE IF EXISTS "+tgtT)
	defer func() {
		lcExec(t, src, "DROP TABLE IF EXISTS "+srcT)
		lcExec(t, dst, "DROP TABLE IF EXISTS "+tgtT)
	}()
	// 可空整型唯一索引：无主键，走 DetectIntShardKey 的唯一索引分支，且会补 `id IS NULL` 片
	lcExec(t, src, "CREATE TABLE "+srcT+" (id int, name varchar(50), UNIQUE KEY uk_id (id))")
	lcExec(t, dst, "CREATE TABLE "+tgtT+" (id int, name varchar(50))")
	for start := 1; start <= 2400; start += 400 {
		end := min(start+399, 2400)
		values := make([]string, 0, end-start+1)
		for id := start; id <= end; id++ {
			values = append(values, fmt.Sprintf("(%d,'n%d')", id, id))
		}
		lcExec(t, src, fmt.Sprintf("INSERT INTO %s (id, name) VALUES %s", srcT, strings.Join(values, ",")))
	}
	nullValues := make([]string, 0, 250)
	for i := 1; i <= 250; i++ {
		nullValues = append(nullValues, fmt.Sprintf("(NULL,'null%d')", i))
	}
	lcExec(t, src, fmt.Sprintf("INSERT INTO %s (id, name) VALUES %s", srcT, strings.Join(nullValues, ",")))

	oldBudget := shardQueryBudget
	shardQueryBudget = time.Nanosecond
	t.Cleanup(func() { shardQueryBudget = oldBudget })

	app := &DataSyncAppImpl{}
	app.Repo = &fakeDataSyncRepo{}
	const taskId = uint64(9933)
	runId, ok := app.runGuard.AcquireWithRunId(taskId)
	require.True(t, ok)
	defer app.runGuard.ReleaseWithRunId(taskId, runId)

	task := &entity.DataSyncTask{
		Id: taskId, TaskName: "it-shard-resume-null", SyncMode: entity.DataSyncModeIncrementalAppend,
		TargetTableName: tgtT, PageSize: 100,
		FieldMap: `[{"src":"id","target":"id"},{"src":"name","target":"name"}]`,
		DataSQL:  "select id, name from " + srcT,
	}
	task.SetShardTargetRows(1000)
	syncLog := &entity.DataSyncLog{RunId: runId}
	plan := newSyncQueryPlan(task.DataSQL, "where 1=1", "", "")
	require.NoError(t, app.doDataSync(context.Background(), plan, task, src, dst, syncLog, &SyncMetrics{}), syncLog.ErrText)

	assert.Contains(t, syncLog.RunLog, "按分片键 [id]", "可空唯一索引应可作为分片键")
	assert.Equal(t, 2650, lcCount(t, dst, tgtT), "2400 个非 NULL 键 + 250 个 NULL 键行都必须落库")
	_, rows, err := dst.Query(fmt.Sprintf("SELECT COUNT(*) AS nulls, COUNT(DISTINCT name) AS names FROM %s WHERE id IS NULL", tgtT))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "250", fmt.Sprint(rows[0]["nulls"]), "NULL 键片不得被预算中断截断")
	assert.Equal(t, "250", fmt.Sprint(rows[0]["names"]), "NULL 键行不得重复")
}

// TestITShardResumePgSource pg 作源：分片键引用与续扫谓词必须走 pg 的双引号标识符
// （`"id" > N`、`order by "id" asc`），不能沿用 mysql 反引号。
func TestITShardResumePgSource(t *testing.T) {
	src := lcConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	dst := lcConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	t.Cleanup(func() { src.Close(); dst.Close() })
	const srcT, tgtT = "it_sync_resume_pg_src", "it_sync_resume_pg_tgt"
	lcExec(t, src, "DROP TABLE IF EXISTS "+srcT)
	lcExec(t, dst, "DROP TABLE IF EXISTS "+tgtT)
	defer func() {
		lcExec(t, src, "DROP TABLE IF EXISTS "+srcT)
		lcExec(t, dst, "DROP TABLE IF EXISTS "+tgtT)
	}()
	lcExec(t, src, "CREATE TABLE "+srcT+" (id int primary key, name varchar(50))")
	lcExec(t, dst, "CREATE TABLE "+tgtT+" (id int primary key, name varchar(50))")
	for start := 1; start <= 2000; start += 500 {
		end := min(start+499, 2000)
		values := make([]string, 0, end-start+1)
		for id := start; id <= end; id++ {
			values = append(values, fmt.Sprintf("(%d,'n%d')", id, id))
		}
		lcExec(t, src, fmt.Sprintf("INSERT INTO %s (id, name) VALUES %s", srcT, strings.Join(values, ",")))
	}

	oldBudget := shardQueryBudget
	shardQueryBudget = time.Nanosecond
	t.Cleanup(func() { shardQueryBudget = oldBudget })

	app := &DataSyncAppImpl{}
	app.Repo = &fakeDataSyncRepo{}
	const taskId = uint64(9934)
	runId, ok := app.runGuard.AcquireWithRunId(taskId)
	require.True(t, ok)
	defer app.runGuard.ReleaseWithRunId(taskId, runId)

	task := &entity.DataSyncTask{
		Id: taskId, TaskName: "it-shard-resume-pg", SyncMode: entity.DataSyncModeIncrementalAppend,
		TargetTableName: tgtT, PageSize: 100, UpdField: "id", UpdFieldVal: "0",
		FieldMap: `[{"src":"id","target":"id"},{"src":"name","target":"name"}]`,
		DataSQL:  "select id, name from " + srcT,
	}
	task.SetShardTargetRows(1000)
	syncLog := &entity.DataSyncLog{RunId: runId}
	plan := newSyncQueryPlan(task.DataSQL, "where 1=1", "", "order by id asc")
	require.NoError(t, app.doDataSync(context.Background(), plan, task, src, dst, syncLog, &SyncMetrics{}), syncLog.ErrText)

	assert.Contains(t, syncLog.RunLog, "↻ 片内查询超预算")
	assert.Contains(t, syncLog.RunLog, `"id" >`, "pg 源的续扫谓词必须用双引号标识符")
	assert.Equal(t, 2000, lcCount(t, dst, tgtT), "pg 源续扫不得丢行")
	assert.Equal(t, "2000", task.UpdFieldVal)
}
