//go:build it

package itest

// 增量字段扩展形态的端到端集成测试（黑盒，经导出入口 SyncTask 驱动完整链路）：
//   - 辅助增量字段（UpdFieldSecondary）与主字段以 AND 连接、共用同一水位值；
//   - 增量值来源列（UpdFieldSrc）为结果集别名时，水位仍能取到值并推进；
//   - 忽略策略下写入指标只统计真正落库的行，被唯一约束丢弃的行计入跳过。

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

// setupIncrField 建 mysql 源 / pg 目标表，两端列结构由调用方给定。
func setupIncrField(t *testing.T, srcTable, srcDDL, tgtTable, tgtDDL string) (*sync.DataSyncAppImpl, *dbi.DbConn, *dbi.DbConn) {
	t.Helper()
	src := mysqlConn(t)
	dst := pgConn(t)
	t.Cleanup(func() { src.Close(); dst.Close() })

	mustExec(t, src, "DROP TABLE IF EXISTS "+quote(src, srcTable))
	mustExec(t, src, fmt.Sprintf("CREATE TABLE %s %s", quote(src, srcTable), srcDDL))
	mustExec(t, dst, "DROP TABLE IF EXISTS "+quote(dst, tgtTable))
	mustExec(t, dst, fmt.Sprintf("CREATE TABLE %s %s", quote(dst, tgtTable), tgtDDL))

	app, _ := newApp()
	return app, src, dst
}

// TestITSecondaryIncrementalField 辅助增量字段与主字段共用同一水位值并以 AND 连接：
// 只推进主增量字段、辅助字段仍低于水位的行不会被同步（该配置不是独立水位/复合游标）。
func TestITSecondaryIncrementalField(t *testing.T) {
	srcTable, tgtTable := "it_sync_secf_src", "it_sync_secf_tgt"
	app, src, dst := setupIncrField(t, srcTable,
		"(id int primary key, name varchar(50), ctime int, utime int)",
		tgtTable, "(id int primary key, name varchar(50), ctime int, utime int)")
	ctx := context.Background()

	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (1,'a',10,10),(2,'b',20,20)", srcTable))
	task := &entity.DataSyncTask{
		Id: 9904, TaskName: "it-sync-secondary-field", SyncMode: entity.DataSyncModeIncrementalMerge,
		SrcDbId: 1, TargetDbId: 2, TargetTableName: tgtTable, PageSize: 100,
		FieldMap:          `[{"src":"id","target":"id"},{"src":"name","target":"name"},{"src":"ctime","target":"ctime"},{"src":"utime","target":"utime"}]`,
		UpdField:          "ctime",
		UpdFieldSecondary: "utime",
		UpdFieldVal:       "0",
		DataSQL:           fmt.Sprintf("SELECT id, name, ctime, utime FROM %s", srcTable),
	}
	// 断言的是 AND 条件下“边界行不重发”旧语义；显式 Exclusive，保留历史预期（inclusive 由专项用例回归）
	task.SetCursorInclusivity(entity.CursorInclusivityExclusive)
	require.NoError(t, app.SyncTask(ctx, task, src, dst, &entity.DataSyncLog{}, nil))
	require.Equal(t, "20", task.UpdFieldVal, "水位应取主增量字段末行值")

	// 3：ctime 已越过水位但 utime 仍在水位之下 → AND 条件将其排除；4：两字段都越过水位 → 命中
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (3,'c',30,15),(4,'d',40,45)", srcTable))
	log2 := &entity.DataSyncLog{}
	metrics2 := sync.NewSyncMetrics()
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log2, metrics2), log2.ErrText)
	assert.Contains(t, log2.DataSQLFull, "and ctime > ", "主字段增量条件缺失: %s", log2.DataSQLFull)
	assert.Contains(t, log2.DataSQLFull, "and utime > ", "辅助字段增量条件缺失: %s", log2.DataSQLFull)
	assert.Equal(t, 1, log2.ResNum, "仅两个增量字段都越过水位的行才同步")
	assert.Equal(t, []int64{1, 2, 4}, queryIds(t, dst, tgtTable), "辅助字段未越过水位的行不得同步")
	assert.Equal(t, "40", task.UpdFieldVal)
	assert.Equal(t, 1, metrics2.InsertCount, "本批只有1行真实新增")
	assert.Equal(t, 0, metrics2.UpdateCount)

	// 只补上辅助字段（ctime 不变）后，该行仍不会被同步：AND 语义下水位已被主字段推进越过
	mustExec(t, src, fmt.Sprintf("UPDATE %s SET utime = 50 WHERE id = 3", srcTable))
	log3 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log3, nil), log3.ErrText)
	assert.Equal(t, 0, log3.ResNum, "ctime 未越过水位的行不会再次同步")
	assert.Equal(t, []int64{1, 2, 4}, queryIds(t, dst, tgtTable))
}

// TestITUpdFieldSrcAliasedWatermark 源查询把增量列改名为结果集别名时，UpdFieldSrc 指明水位取值列：
// 水位 WHERE 仍用物理列名，而推进水位须读结果集里的别名列，否则水位永远停在原值。
func TestITUpdFieldSrcAliasedWatermark(t *testing.T) {
	srcTable, tgtTable := "it_sync_ali_src", "it_sync_ali_tgt"
	app, src, dst := setupIncrField(t, srcTable,
		"(id int primary key, name varchar(50), upd_time datetime)",
		tgtTable, "(id int primary key, name varchar(50), upd_time timestamp)")
	ctx := context.Background()

	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (1,'alice','2026-01-01 10:00:00'),(2,'bob','2026-01-02 10:00:00')", srcTable))
	task := &entity.DataSyncTask{
		Id: 9905, TaskName: "it-sync-upd-field-src", SyncMode: entity.DataSyncModeIncrementalMerge,
		SrcDbId: 1, TargetDbId: 2, TargetTableName: tgtTable, PageSize: 100,
		FieldMap:    `[{"src":"id","target":"id"},{"src":"name","target":"name"},{"src":"changed_at","target":"upd_time"}]`,
		UpdField:    "upd_time",
		UpdFieldSrc: "changed_at",
		UpdFieldVal: "0",
		DataSQL:     fmt.Sprintf("SELECT id, name, upd_time AS changed_at FROM %s", srcTable),
	}
	// 断言 “1 行命中” 的精确增量行数，与 exclusive 旧语义相当；inclusive 下会多拉当前水位行，需显式回退
	task.SetCursorInclusivity(entity.CursorInclusivityExclusive)
	log1 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log1, nil), log1.ErrText)
	require.Equal(t, 2, countRows(t, dst, tgtTable))
	assert.Contains(t, task.UpdFieldVal, "2026-01-02", "水位须取自结果集别名列 changed_at，实际为 %q", task.UpdFieldVal)

	// 源端就地更新并推进时间戳：命中增量的行应在目标端被覆盖
	watermark := task.UpdFieldVal
	mustExec(t, src, fmt.Sprintf("UPDATE %s SET name='alice-upd', upd_time='2026-01-05 10:00:00' WHERE id=1", srcTable))
	log2 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log2, nil), log2.ErrText)
	assert.Contains(t, log2.DataSQLFull, "upd_time >", "水位 WHERE 仍应使用物理列名，实际SQL: %s", log2.DataSQLFull)
	assert.Equal(t, 1, log2.ResNum)
	assert.NotEqual(t, watermark, task.UpdFieldVal, "水位应继续推进")
	assert.Contains(t, task.UpdFieldVal, "2026-01-05")

	_, rows, err := dst.Query("SELECT id, name FROM " + quote(dst, tgtTable) + " ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows, 2, "更新不得产生新行")
	assert.Equal(t, "alice-upd", fmt.Sprint(rows[0]["name"]))
	assert.Equal(t, "bob", fmt.Sprint(rows[1]["name"]))
}

// TestITIgnoreStrategyWriteMetrics 忽略策略：冲突行由唯一约束丢弃、未落库，
// 故只有真正插入的行计新增，被丢弃的行计跳过，更新数恒为 0。
func TestITIgnoreStrategyWriteMetrics(t *testing.T) {
	srcTable, tgtTable := "it_sync_ign_src", "it_sync_ign_tgt"
	app, src, dst := setupIncrField(t, srcTable,
		"(id int primary key, name varchar(50))",
		tgtTable, "(id int primary key, name varchar(50))")
	ctx := context.Background()

	mustExec(t, src, "INSERT INTO "+srcTable+" VALUES (1,'alice'),(2,'bob'),(3,'carl')")
	task := &entity.DataSyncTask{
		Id: 9906, TaskName: "it-sync-ignore", SyncMode: entity.DataSyncModeIncrementalAppend,
		SrcDbId: 1, TargetDbId: 2, TargetTableName: tgtTable, PageSize: 100,
		FieldMap:          `[{"src":"id","target":"id"},{"src":"name","target":"name"}]`,
		DuplicateStrategy: dbi.DuplicateStrategyIgnore,
		UpdField:          "id",
		UpdFieldVal:       "0",
		DataSQL:           fmt.Sprintf("SELECT id, name FROM %s", srcTable),
	}
	log1 := &entity.DataSyncLog{}
	metrics1 := sync.NewSyncMetrics()
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log1, metrics1), log1.ErrText)
	assert.Equal(t, 3, metrics1.InsertCount, "空目标表下3行都是新增")
	assert.Equal(t, 0, metrics1.UpdateCount)
	assert.Equal(t, 0, metrics1.SkipCount)

	// 水位归零重跑同一批数据（模拟全量扫描）：已存在行被忽略，仅新行插入，且目标旧值不被覆盖
	mustExec(t, src, "UPDATE "+srcTable+" SET name='alice-changed' WHERE id=1")
	mustExec(t, src, "INSERT INTO "+srcTable+" VALUES (4,'dave')")
	task.UpdFieldVal = "0"
	log2 := &entity.DataSyncLog{}
	metrics2 := sync.NewSyncMetrics()
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log2, metrics2), log2.ErrText)
	assert.Equal(t, 4, log2.ResNum, "源端读到4行")
	assert.Equal(t, 1, metrics2.InsertCount, "只有新行真正落库")
	assert.Equal(t, 0, metrics2.UpdateCount, "忽略策略不产生更新")
	assert.Equal(t, 3, metrics2.SkipCount, "3行冲突被丢弃应计入跳过")

	names := map[int64]string{}
	_, rows, err := dst.Query("SELECT id, name FROM " + quote(dst, tgtTable) + " ORDER BY id")
	require.NoError(t, err)
	for _, row := range rows {
		names[toInt64(row["id"])] = fmt.Sprint(row["name"])
	}
	assert.Len(t, names, 4)
	assert.Equal(t, "alice", names[1], "忽略策略下目标已存在行保持原值")
	assert.Equal(t, "dave", names[4])
}
