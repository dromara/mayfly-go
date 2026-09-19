//go:build it

package sync

// 数据同步模式 3/5/6 端到端集成测试：
//
//	Mode 3 全量刷新：TRUNCATE + 全量 INSERT
//	Mode 5 硬删除：收集源PK → 写入 → 批量删除目标多余行
//	Mode 6 数据校验：对比源/目标行数 + 抽样 checksum
//
// 运行方式：cd server && go test -tags it -count=1 -v -run TestITDataSyncMode ./internal/db/application/sync/

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
)

const (
	modeItSrcTable = "it_sync_mode_src"
	modeItTgtTable = "it_sync_mode_tgt"
	modeItTaskId   = uint64(9902)
)

// setupSyncModeIT 初始化 Mode 3/5/6 测试环境
func setupSyncModeIT(t *testing.T) (*DataSyncAppImpl, *fakeDataSyncRepo, *dbi.DbConn, *dbi.DbConn) {
	t.Helper()
	srcConn := verifyMysqlConn(t)
	tgtConn := verifyPgConn(t)
	t.Cleanup(func() { srcConn.Close(); tgtConn.Close() })

	quoteSrc := srcConn.GetDialect().Quoter().Quote
	quoteTgt := tgtConn.GetDialect().Quoter().Quote

	// 源表（mysql）与目标表（pg）同构
	_, err := srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteSrc(modeItSrcTable)))
	require.NoError(t, err)
	_, err = srcConn.Exec(fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50), status int)", quoteSrc(modeItSrcTable)))
	require.NoError(t, err)

	_, err = tgtConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteTgt(modeItTgtTable)))
	require.NoError(t, err)
	_, err = tgtConn.Exec(fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50), status int)", quoteTgt(modeItTgtTable)))
	require.NoError(t, err)

	fake := &fakeDataSyncRepo{}
	app := &DataSyncAppImpl{
		AppImpl: base.AppImpl[*entity.DataSyncTask, repository.DataSyncTask]{Repo: fake},
	}
	app.dbApp = &fakeSyncDbApp{srcId: 1, tgtId: 2, srcConn: srcConn, tgtConn: tgtConn}
	app.dbDataSyncLogRepo = &fakeSyncLogRepo{}
	return app, fake, srcConn, tgtConn
}

// newModeSyncTask 构造指定模式的同步任务
func newModeSyncTask(mode entity.DataSyncMode) *entity.DataSyncTask {
	return &entity.DataSyncTask{
		Id:              modeItTaskId,
		TaskName:        fmt.Sprintf("it-sync-mode-%d", mode),
		SrcDbId:         1,
		SrcDbName:       "mayfly_dbm_it",
		TargetDbId:      2,
		TargetDbName:    "mayfly_pg_it",
		TargetTableName: modeItTgtTable,
		FieldMap:        `[{"src":"id","target":"id"},{"src":"name","target":"name"},{"src":"status","target":"status"}]`,
		PageSize:        100,
		SyncMode:        mode,
		DataSql:         fmt.Sprintf("SELECT id, name, status FROM %s", modeItSrcTable),
	}
}

// TestITDataSyncMode3FullRefresh 全量刷新模式：TRUNCATE + 全量 INSERT
func TestITDataSyncMode3FullRefresh(t *testing.T) {
	app, _, srcConn, tgtConn := setupSyncModeIT(t)
	ctx := context.Background()

	// 准备源数据：5行
	_, err := srcConn.Exec(fmt.Sprintf("INSERT INTO %s VALUES (1,'alice',1),(2,'bob',1),(3,'charlie',0),(4,'dave',1),(5,'eve',0)", modeItSrcTable))
	require.NoError(t, err)

	// 目标表预置旧数据（应被清空）
	_, err = tgtConn.Exec(fmt.Sprintf("INSERT INTO %s VALUES (100,'old_data',999)", modeItTgtTable))
	require.NoError(t, err)

	// 执行全量刷新
	app.runGuard.Acquire(modeItTaskId)
	task := newModeSyncTask(entity.DataSyncModeFullRefresh)
	sql := fmt.Sprintf("SELECT id, name, status FROM %s", modeItSrcTable)
	syncLog := &entity.DataSyncLog{}
	metrics := NewSyncMetrics()

	require.NoError(t, app.doDataSync(ctx, sql, task, syncLog, metrics))

	// 验证：目标表应仅包含源表的5行，旧数据被清空
	assert.Equal(t, 5, syncLog.ResNum, "应同步5行")
	assert.Equal(t, 5, countSyncRows(t, tgtConn, modeItTgtTable), "目标表应有5行")

	// 验证旧数据被清空
	_, rows, err := tgtConn.Query(fmt.Sprintf("SELECT id FROM %s WHERE id = 100", modeItTgtTable))
	require.NoError(t, err)
	assert.Empty(t, rows, "旧数据应被清空")

	// 验证新数据存在
	_, rows, err = tgtConn.Query(fmt.Sprintf("SELECT id, name FROM %s WHERE id = 1", modeItTgtTable))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "alice", fmt.Sprint(rows[0]["name"]))

	// 验证运行日志包含关键信息
	assert.Contains(t, syncLog.RunLog, "全量刷新模式", "运行日志应包含全量刷新模式信息")
	assert.Contains(t, syncLog.RunLog, "清空目标表", "运行日志应包含清空目标表信息")
}

// TestITDataSyncMode5HardDelete 硬删除模式：源端删除后目标端同步清理
func TestITDataSyncMode5HardDelete(t *testing.T) {
	app, _, srcConn, tgtConn := setupSyncModeIT(t)
	ctx := context.Background()

	// 准备源数据：5行
	_, err := srcConn.Exec(fmt.Sprintf("INSERT INTO %s VALUES (1,'alice',1),(2,'bob',1),(3,'charlie',1),(4,'dave',1),(5,'eve',1)", modeItSrcTable))
	require.NoError(t, err)

	// 首次同步：全量同步5行到目标
	app.runGuard.Acquire(modeItTaskId)
	task := newModeSyncTask(entity.DataSyncModeIncrementalAppend)
	sql := fmt.Sprintf("SELECT id, name, status FROM %s", modeItSrcTable)
	syncLog := &entity.DataSyncLog{}
	metrics := NewSyncMetrics()
	require.NoError(t, app.doDataSync(ctx, sql, task, syncLog, metrics))
	assert.Equal(t, 5, countSyncRows(t, tgtConn, modeItTgtTable), "首次同步后目标应有5行")
	app.runGuard.Release(modeItTaskId)

	// 源端删除2行（模拟业务删除）
	_, err = srcConn.Exec(fmt.Sprintf("DELETE FROM %s WHERE id IN (3, 5)", modeItSrcTable))
	require.NoError(t, err)

	// 硬删除模式同步：应删除目标端的多余行
	app.runGuard.Acquire(modeItTaskId)
	task5 := newModeSyncTask(entity.DataSyncModeIncrementalHardDel)
	task5.DuplicateStrategy = int(dbi.DuplicateStrategyUpdate) // UPSERT
	sql5 := fmt.Sprintf("SELECT id, name, status FROM %s", modeItSrcTable)
	syncLog5 := &entity.DataSyncLog{}
	metrics5 := NewSyncMetrics()
	require.NoError(t, app.doDataSync(ctx, sql5, task5, syncLog5, metrics5))
	app.runGuard.Release(modeItTaskId)

	// 验证：目标端应只剩3行（id=1,2,4）
	assert.Equal(t, 3, countSyncRows(t, tgtConn, modeItTgtTable), "硬删除后目标应剩3行")

	// 验证被删除的行不存在
	_, rows, err := tgtConn.Query(fmt.Sprintf("SELECT id FROM %s WHERE id IN (3, 5)", modeItTgtTable))
	require.NoError(t, err)
	assert.Empty(t, rows, "id=3,5 应从目标删除")

	// 验证保留的行存在
	_, rows, err = tgtConn.Query(fmt.Sprintf("SELECT id FROM %s WHERE id IN (1, 2, 4) ORDER BY id", modeItTgtTable))
	require.NoError(t, err)
	assert.Len(t, rows, 3, "id=1,2,4 应保留在目标")

	// 验证运行日志包含硬删除信息
	assert.Contains(t, syncLog5.RunLog, "硬删除模式", "运行日志应包含硬删除模式信息")
	assert.Greater(t, metrics5.DeleteCount, 0, "删除计数应大于0")
}

// TestITDataSyncMode6Validation 数据校验模式：对比源/目标行数与checksum
func TestITDataSyncMode6Validation(t *testing.T) {
	app, _, srcConn, tgtConn := setupSyncModeIT(t)
	ctx := context.Background()

	// 准备源数据：10行
	_, err := srcConn.Exec(fmt.Sprintf("INSERT INTO %s VALUES (1,'a',1),(2,'b',1),(3,'c',1),(4,'d',1),(5,'e',1),(6,'f',1),(7,'g',1),(8,'h',1),(9,'i',1),(10,'j',1)", modeItSrcTable))
	require.NoError(t, err)

	// 首次同步：全量同步10行到目标
	app.runGuard.Acquire(modeItTaskId)
	task := newModeSyncTask(entity.DataSyncModeIncrementalAppend)
	sql := fmt.Sprintf("SELECT id, name, status FROM %s", modeItSrcTable)
	syncLog := &entity.DataSyncLog{}
	metrics := NewSyncMetrics()
	require.NoError(t, app.doDataSync(ctx, sql, task, syncLog, metrics))
	assert.Equal(t, 10, countSyncRows(t, tgtConn, modeItTgtTable), "首次同步后目标应有10行")
	app.runGuard.Release(modeItTaskId)

	// 执行数据校验：应通过
	app.runGuard.Acquire(modeItTaskId)
	task6 := newModeSyncTask(entity.DataSyncModeValidation)
	syncLog6 := &entity.DataSyncLog{}
	metrics6 := NewSyncMetrics()
	app.doDataValidation(ctx, task6, syncLog6, metrics6)
	app.runGuard.Release(modeItTaskId)

	// 验证：校验应通过
	assert.Equal(t, int8(entity.DataSyncTaskStateSuccess), syncLog6.Status, "校验应通过")
	assert.Contains(t, syncLog6.ErrText, "validation passed", "校验结果应包含passed")
	assert.Contains(t, syncLog6.RunLog, "数据校验完成", "运行日志应包含校验完成信息")

	// 目标端删除2行，再次校验应失败
	_, err = tgtConn.Exec(fmt.Sprintf("DELETE FROM %s WHERE id IN (1, 2)", modeItTgtTable))
	require.NoError(t, err)

	app.runGuard.Acquire(modeItTaskId)
	syncLog7 := &entity.DataSyncLog{}
	metrics7 := NewSyncMetrics()
	app.doDataValidation(ctx, task6, syncLog7, metrics7)
	app.runGuard.Release(modeItTaskId)

	// 验证：校验应失败（行数不匹配）
	assert.Equal(t, int8(entity.DataSyncTaskStateFail), syncLog7.Status, "校验应失败")
	assert.Contains(t, syncLog7.ErrText, "validation failed", "校验结果应包含failed")
	assert.Contains(t, syncLog7.RunLog, "校验失败", "运行日志应包含校验失败信息")
}

// TestITDataSyncMode4SoftDelete 软删除模式：源端软删除后目标端同步清理
func TestITDataSyncMode4SoftDelete(t *testing.T) {
	app, _, srcConn, tgtConn := setupSyncModeIT(t)
	ctx := context.Background()

	// 准备源数据：5行，status=0表示正常，status=1表示软删除
	_, err := srcConn.Exec(fmt.Sprintf("INSERT INTO %s VALUES (1,'alice',0),(2,'bob',0),(3,'charlie',1),(4,'dave',0),(5,'eve',1)", modeItSrcTable))
	require.NoError(t, err)

	// 首次同步：全量同步5行到目标
	app.runGuard.Acquire(modeItTaskId)
	task := newModeSyncTask(entity.DataSyncModeIncrementalAppend)
	sql := fmt.Sprintf("SELECT id, name, status FROM %s", modeItSrcTable)
	syncLog := &entity.DataSyncLog{}
	metrics := NewSyncMetrics()
	require.NoError(t, app.doDataSync(ctx, sql, task, syncLog, metrics))
	assert.Equal(t, 5, countSyncRows(t, tgtConn, modeItTgtTable), "首次同步后目标应有5行")
	app.runGuard.Release(modeItTaskId)

	// 软删除模式同步：应过滤掉status=1的行，并清理目标端多余行
	app.runGuard.Acquire(modeItTaskId)
	task4 := newModeSyncTask(entity.DataSyncModeIncrementalSoftDel)
	task4.DuplicateStrategy = int(dbi.DuplicateStrategyUpdate) // UPSERT
	task4.SoftDeleteField = "status"
	task4.SoftDeleteValue = "1"
	// 软删除过滤条件需要在SQL中手动添加（模拟Run()中的行为）
	sql4 := fmt.Sprintf("SELECT id, name, status FROM %s WHERE (status != '%s' OR status IS NULL)", modeItSrcTable, task4.SoftDeleteValue)
	syncLog4 := &entity.DataSyncLog{}
	metrics4 := NewSyncMetrics()
	require.NoError(t, app.doDataSync(ctx, sql4, task4, syncLog4, metrics4))
	app.runGuard.Release(modeItTaskId)

	// 验证：目标端应只剩3行（id=1,2,4，status=0的行）
	assert.Equal(t, 3, countSyncRows(t, tgtConn, modeItTgtTable), "软删除后目标应剩3行")

	// 验证被软删除的行在目标端被清理
	_, rows, err := tgtConn.Query(fmt.Sprintf("SELECT id FROM %s WHERE id IN (3, 5)", modeItTgtTable))
	require.NoError(t, err)
	assert.Empty(t, rows, "id=3,5 应从目标清理")

	// 验证保留的行存在
	_, rows, err = tgtConn.Query(fmt.Sprintf("SELECT id FROM %s WHERE id IN (1, 2, 4) ORDER BY id", modeItTgtTable))
	require.NoError(t, err)
	assert.Len(t, rows, 3, "id=1,2,4 应保留在目标")

	// 验证运行日志包含软删除信息
	assert.Contains(t, syncLog4.RunLog, "软删除", "运行日志应包含软删除模式信息")
	assert.Greater(t, metrics4.DeleteCount, 0, "删除计数应大于0")
}
