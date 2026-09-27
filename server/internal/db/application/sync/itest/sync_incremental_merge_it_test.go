//go:build it

package itest

// 增量合并模式（SyncMode=2）端到端集成测试（黑盒，经导出入口 SyncTask 驱动完整链路）：
// 源端「已有行被修改 + 新行插入」在下一轮增量里落到目标库的语义 —— 命中冲突键则 UPDATE，未命中则 INSERT，
// 水位之下的未变更行必须原样保留。增量字段的数据类型（时间型 / 自增整型的只追加语义）决定哪些变更能被源查询捞出。

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

const (
	mergeItSrcTable = "it_sync_merge_src"
	mergeItTgtTable = "it_sync_merge_tgt"
	mergeItTaskId   = uint64(9903)
)

// setupMerge 建 mysql 源 / pg 目标（含 update_time 列），返回黑盒应用实例与两端连接。
func setupMerge(t *testing.T) (*sync.DataSyncAppImpl, *dbi.DbConn, *dbi.DbConn) {
	t.Helper()
	src := mysqlConn(t)
	dst := pgConn(t)
	t.Cleanup(func() { src.Close(); dst.Close() })

	mustExec(t, src, "DROP TABLE IF EXISTS "+quote(src, mergeItSrcTable))
	mustExec(t, src, fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50), update_time datetime(3))", quote(src, mergeItSrcTable)))
	mustExec(t, dst, "DROP TABLE IF EXISTS "+quote(dst, mergeItTgtTable))
	mustExec(t, dst, fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50), update_time timestamp(3))", quote(dst, mergeItTgtTable)))
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (1,'alice','2026-01-01 10:00:00.123'),(2,'bob','2026-01-02 10:00:00.000'),(3,'carl','2026-01-03 10:00:00.000')", mergeItSrcTable))

	app, _ := newApp()
	return app, src, dst
}

func mergeTask(updField, watermark string) *entity.DataSyncTask {
	task := &entity.DataSyncTask{
		Id:              mergeItTaskId,
		TaskName:        "it-sync-merge",
		SrcDbId:         1,
		SrcDbName:       "mayfly_dbm_it",
		TargetDbId:      2,
		TargetDbName:    "mayfly_pg_it",
		TargetTableName: mergeItTgtTable,
		FieldMap:        `[{"src":"id","target":"id"},{"src":"name","target":"name"},{"src":"update_time","target":"update_time"}]`,
		PageSize:        100,
		SyncMode:        entity.DataSyncModeIncrementalMerge,
		UpdField:        updField,
		UpdFieldVal:     watermark,
		DataSQL:         fmt.Sprintf("SELECT id, name, update_time FROM %s", mergeItSrcTable),
	}
	// 本用例断言的是“水位推进后边界行不重发”的旧语义；P0 后 Merge Auto 默认 Inclusive（at-least-once），
	// 保留历史断言需显式 Exclusive；inclusive 行为另由 sync_inclusive_cursor_it_test.go 直接回归。
	task.SetCursorInclusivity(entity.CursorInclusivityExclusive)
	return task
}

// queryMergeNames 查询目标表 id→name 映射，供逐行断言。
func queryMergeNames(t *testing.T, c *dbi.DbConn) map[int64]string {
	t.Helper()
	_, rows, err := c.Query("SELECT id, name FROM " + quote(c, mergeItTgtTable) + " ORDER BY id")
	require.NoError(t, err)
	names := make(map[int64]string, len(rows))
	for _, row := range rows {
		names[toInt64(row["id"])] = fmt.Sprint(row["name"])
	}
	return names
}

// TestITIncrementalMergeWithTimeWatermark 时间型增量字段：源端改动会推进 update_time，
// 故被改行与新行都落入本轮增量集合，目标端分别得到 UPDATE 与 INSERT，未变更行保持不动。
func TestITIncrementalMergeWithTimeWatermark(t *testing.T) {
	app, src, dst := setupMerge(t)
	ctx := context.Background()

	task := mergeTask("update_time", "0")
	log1 := &entity.DataSyncLog{}
	metrics1 := sync.NewSyncMetrics()
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log1, metrics1), log1.ErrText)
	assert.Equal(t, 3, countRows(t, dst, mergeItTgtTable), "首轮无水位应同步源全部3行")
	assert.Equal(t, "2026-01-03 10:00:00", task.UpdFieldVal, "水位应为末行增量字段值")
	assert.Equal(t, 3, metrics1.InsertCount, "目标表为空时3行全部是新增")
	assert.Equal(t, 0, metrics1.UpdateCount)

	// 源端：已有行改值并推进其更新时间；再插入一行新数据
	mustExec(t, src, fmt.Sprintf("UPDATE %s SET name='alice-upd', update_time='2026-01-05 10:00:00.000' WHERE id=1", mergeItSrcTable))
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (4,'dave','2026-01-06 10:00:00.000')", mergeItSrcTable))

	log2 := &entity.DataSyncLog{}
	metrics2 := sync.NewSyncMetrics()
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log2, metrics2), log2.ErrText)
	assert.Contains(t, log2.DataSQLFull, "update_time >", "增量条件必须叠加到源查询", "实际SQL: %s", log2.DataSQLFull)
	assert.Equal(t, 2, log2.ResNum, "本轮增量只应命中被改行与新增行")
	assert.Equal(t, "2026-01-06 10:00:00", task.UpdFieldVal, "水位应推进到本轮末行")
	assert.Equal(t, 1, metrics2.InsertCount, "新增行计新增")
	assert.Equal(t, 1, metrics2.UpdateCount, "命中已有冲突键的行计更新")

	names := queryMergeNames(t, dst)
	assert.Len(t, names, 4, "目标行数应等于源现存行数，水位之下的行不得丢失")
	assert.Equal(t, "alice-upd", names[1], "源端已更新的行，目标端命中冲突键后须同步更新")
	assert.Equal(t, "bob", names[2], "水位之下的未变更行保持原值")
	assert.Equal(t, "carl", names[3], "水位之下的未变更行保持原值")
	assert.Equal(t, "dave", names[4], "源端新增的行，目标端未命中冲突键须插入")

	// 源端无变更再跑一轮：增量为空，目标端既不多写也不清空
	log3 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log3, nil), log3.ErrText)
	assert.Equal(t, 0, log3.ResNum, "无新数据时本轮不应写入")
	assert.Equal(t, names, queryMergeNames(t, dst), "空增量轮不得改变目标数据")
}

// TestITIncrementalMergeIdWatermarkAppendOnly 自增 id 作增量字段时只具备「追加」语义：
// 源端就地修改已有行不会推进 id 水位，该行不会被本轮源查询捞出，目标端保持旧值。
// 断言此边界是为了明确「需要传播就地更新时必须选时间型（或版本型）增量字段」，避免误用。
func TestITIncrementalMergeIdWatermarkAppendOnly(t *testing.T) {
	app, src, dst := setupMerge(t)
	ctx := context.Background()

	task := mergeTask("id", "0")
	log1 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log1, nil), log1.ErrText)
	require.Equal(t, 3, countRows(t, dst, mergeItTgtTable))
	require.Equal(t, "3", task.UpdFieldVal)

	mustExec(t, src, fmt.Sprintf("UPDATE %s SET name='alice-upd', update_time='2026-01-05 10:00:00.000' WHERE id=1", mergeItSrcTable))
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (4,'dave','2026-01-06 10:00:00.000')", mergeItSrcTable))

	log2 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log2, nil), log2.ErrText)
	assert.Contains(t, log2.DataSQLFull, "id >", "实际SQL: %s", log2.DataSQLFull)
	assert.Equal(t, 1, log2.ResNum, "id 水位只能命中新增行")

	names := queryMergeNames(t, dst)
	assert.Equal(t, "dave", names[4], "新增行仍应插入")
	assert.Equal(t, "alice", names[1], "id 水位下源端就地更新不传播（需改用时间型增量字段或全量对账模式）")
}
