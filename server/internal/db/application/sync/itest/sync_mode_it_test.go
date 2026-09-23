//go:build it

package itest

// 数据同步模式端到端集成测试（黑盒，经导出入口 SyncTask 驱动完整链路）：
// 全量刷新、删除对账（源软删除 / 源缺失删除）、数据校验，以及删除对账的边界保护。

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

const (
	modeItSrcTable = "it_sync_mode_src"
	modeItTgtTable = "it_sync_mode_tgt"
	modeItTaskId   = uint64(9902)
)

// setupMode 建同构 mysql 源 / pg 目标，返回黑盒应用实例与两端连接。
func setupMode(t *testing.T) (*sync.DataSyncAppImpl, *dbi.DbConn, *dbi.DbConn) {
	t.Helper()
	src := mysqlConn(t)
	dst := pgConn(t)
	t.Cleanup(func() { src.Close(); dst.Close() })

	mustExec(t, src, "DROP TABLE IF EXISTS "+quote(src, modeItSrcTable))
	mustExec(t, src, fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50), status int)", quote(src, modeItSrcTable)))
	mustExec(t, dst, "DROP TABLE IF EXISTS "+quote(dst, modeItTgtTable))
	mustExec(t, dst, fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50), status int)", quote(dst, modeItTgtTable)))

	app, _ := newApp()
	return app, src, dst
}

func modeTask(mode entity.DataSyncMode) *entity.DataSyncTask {
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
		DataSQL:         fmt.Sprintf("SELECT id, name, status FROM %s", modeItSrcTable),
	}
}

func TestITDataSyncMode3FullRefresh(t *testing.T) {
	app, src, dst := setupMode(t)
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (1,'alice',1),(2,'bob',1),(3,'charlie',0),(4,'dave',1),(5,'eve',0)", modeItSrcTable))
	mustExec(t, dst, fmt.Sprintf("INSERT INTO %s VALUES (100,'old_data',999)", modeItTgtTable))

	syncLog := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(context.Background(), modeTask(entity.DataSyncModeFullRefresh), src, dst, syncLog, nil))
	assert.Equal(t, 5, syncLog.ResNum)
	assert.Equal(t, 5, countRows(t, dst, modeItTgtTable), "目标应仅含源5行，旧数据被清空")
	assert.Contains(t, syncLog.RunLog, "全量刷新模式")
}

func TestITDataSyncMode5HardDelete(t *testing.T) {
	app, src, dst := setupMode(t)
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (1,'alice',1),(2,'bob',1),(3,'charlie',1),(4,'dave',1),(5,'eve',1)", modeItSrcTable))

	require.NoError(t, app.SyncTask(context.Background(), modeTask(entity.DataSyncModeIncrementalAppend), src, dst, &entity.DataSyncLog{}, nil))
	assert.Equal(t, 5, countRows(t, dst, modeItTgtTable))

	mustExec(t, src, fmt.Sprintf("DELETE FROM %s WHERE id IN (3, 5)", modeItSrcTable))

	task5 := modeTask(entity.DataSyncModeIncrementalHardDel)
	metrics5 := sync.NewSyncMetrics()
	syncLog5 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(context.Background(), task5, src, dst, syncLog5, metrics5))
	assert.Equal(t, 3, countRows(t, dst, modeItTgtTable), "对账后目标应剩3行")
	assert.Equal(t, []int64{1, 2, 4}, queryIds(t, dst, modeItTgtTable))
	assert.Contains(t, syncLog5.RunLog, "全量对账")
	assert.Greater(t, metrics5.DeleteCount, 0)
}

func TestITDataSyncMode4SoftDelete(t *testing.T) {
	app, src, dst := setupMode(t)
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (1,'alice',0),(2,'bob',0),(3,'charlie',1),(4,'dave',0),(5,'eve',1)", modeItSrcTable))
	require.NoError(t, app.SyncTask(context.Background(), modeTask(entity.DataSyncModeIncrementalAppend), src, dst, &entity.DataSyncLog{}, nil))
	assert.Equal(t, 5, countRows(t, dst, modeItTgtTable))

	task4 := modeTask(entity.DataSyncModeIncrementalSoftDel)
	task4.SoftDeleteField = "status"
	task4.SoftDeleteValue = "1"
	metrics4 := sync.NewSyncMetrics()
	syncLog4 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(context.Background(), task4, src, dst, syncLog4, metrics4))
	assert.Equal(t, 3, countRows(t, dst, modeItTgtTable), "源软删除后目标应剩3行")
	assert.Equal(t, []int64{1, 2, 4}, queryIds(t, dst, modeItTgtTable))
	assert.Greater(t, metrics4.DeleteCount, 0)
}

func TestITDataSyncMode6Validation(t *testing.T) {
	app, src, dst := setupMode(t)
	{
		var vals []string
		for i := 1; i <= 10; i++ {
			vals = append(vals, fmt.Sprintf("(%d,'n%d',1)", i, i))
		}
		mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES %s", modeItSrcTable, strings.Join(vals, ",")))
	}
	require.NoError(t, app.SyncTask(context.Background(), modeTask(entity.DataSyncModeIncrementalAppend), src, dst, &entity.DataSyncLog{}, nil))

	task6 := modeTask(entity.DataSyncModeValidation)
	syncLog6 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(context.Background(), task6, src, dst, syncLog6, nil))
	assert.Equal(t, int8(entity.DataSyncTaskStateSuccess), syncLog6.Status)
	assert.Contains(t, syncLog6.ErrText, "validation passed")

	mustExec(t, dst, fmt.Sprintf("DELETE FROM %s WHERE id IN (1, 2)", modeItTgtTable))
	syncLog7 := &entity.DataSyncLog{}
	require.Error(t, app.SyncTask(context.Background(), task6, src, dst, syncLog7, nil))
	assert.Equal(t, int8(entity.DataSyncTaskStateFail), syncLog7.Status)
	assert.Contains(t, syncLog7.RunLog, "校验失败")
}

func TestITValidationQuotedNamesAndBoundaries(t *testing.T) {
	app, src, dst := setupMode(t)
	mustExec(t, src, "CREATE TABLE `Validation Source` (`First` VARCHAR(50), `Second` VARCHAR(50))")
	mustExec(t, dst, `CREATE TABLE "Validation Target" ("First" VARCHAR(50), "Second" VARCHAR(50))`)
	mustExec(t, src, "INSERT INTO `Validation Source` VALUES ('a|b','c')")
	mustExec(t, dst, `INSERT INTO "Validation Target" VALUES ('a|b','c')`)

	task := modeTask(entity.DataSyncModeValidation)
	task.DataSQL = "SELECT * FROM `Validation Source`"
	task.TargetTableName = "Validation Target"
	task.FieldMap = `[{"src":"First","target":"First"},{"src":"Second","target":"Second"}]`
	check := func(want int8) {
		log := &entity.DataSyncLog{}
		_ = app.SyncTask(context.Background(), task, src, dst, log, nil)
		assert.Equal(t, want, log.Status, log.ErrText)
	}
	check(entity.DataSyncTaskStateSuccess)
	// 分隔符语义：'a|b','c' 与 'a','b|c' 不得被判为相同行
	mustExec(t, dst, `UPDATE "Validation Target" SET "First"='a', "Second"='b|c'`)
	check(entity.DataSyncTaskStateFail)
	task.FieldMap = "invalid json"
	check(entity.DataSyncTaskStateFail)
}

func TestITReconcileBoundaries(t *testing.T) {
	for _, scenario := range []string{"empty", "alias", "transformedKey", "skipRow", "invalidFilter", "missingFilterField", "deleteFailure"} {
		t.Run(scenario, func(t *testing.T) {
			app, src, dst := setupMode(t)
			task := modeTask(entity.DataSyncModeIncrementalHardDel)
			mustExec(t, dst, "INSERT INTO "+modeItTgtTable+" VALUES (99,'old',1)")
			if scenario != "empty" {
				mustExec(t, src, "INSERT INTO "+modeItSrcTable+" VALUES (1,'one',1),(2,'two',1)")
			}
			switch scenario {
			case "alias":
				task.DataSQL = "SELECT id AS source_id, name, status FROM " + modeItSrcTable
				task.FieldMap = `[{"src":"source_id","target":"id"},{"src":"name","target":"name"},{"src":"status","target":"status"}]`
			case "transformedKey":
				task.TransformRules = `[{"targetColumn":"id","type":"expr","config":{"expression":"CONCAT('1', id)"}}]`
			case "skipRow":
				mustExec(t, src, "UPDATE "+modeItSrcTable+" SET name = NULL WHERE id=1")
				task.NullStrategy = entity.NullStrategySkipRow
			case "invalidFilter":
				task.FilterCondition = "(status == 1 OR name == 'one'"
			case "missingFilterField":
				task.FilterCondition = "missing_field == 1"
			case "deleteFailure":
				mustExec(t, dst, "CREATE FUNCTION reject_sync_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'delete denied'; END $$")
				mustExec(t, dst, "CREATE TRIGGER reject_sync_delete BEFORE DELETE ON "+modeItTgtTable+" FOR EACH ROW EXECUTE FUNCTION reject_sync_delete()")
			}
			metrics := sync.NewSyncMetrics()
			err := app.SyncTask(context.Background(), task, src, dst, &entity.DataSyncLog{}, metrics)
			if scenario == "skipRow" || scenario == "deleteFailure" || scenario == "invalidFilter" || scenario == "missingFilterField" {
				require.Error(t, err)
				assert.Contains(t, queryIds(t, dst, modeItTgtTable), int64(99), "失败时不得清理目标旧行")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 1, metrics.DeleteCount)
			switch scenario {
			case "empty":
				assert.Empty(t, queryIds(t, dst, modeItTgtTable))
			case "transformedKey":
				assert.Equal(t, []int64{11, 12}, queryIds(t, dst, modeItTgtTable))
			default:
				assert.Equal(t, []int64{1, 2}, queryIds(t, dst, modeItTgtTable))
			}
		})
	}
}

// TestITDataSyncHardDeleteWatermarkNoOverDelete 已推进水位的第二轮删除对账不得叠加水位增量条件、不得误删。
func TestITDataSyncHardDeleteWatermarkNoOverDelete(t *testing.T) {
	app, src, dst := setupMode(t)
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (1,'alice',1),(2,'bob',1),(3,'charlie',1),(4,'dave',1),(5,'eve',1)", modeItSrcTable))

	task := modeTask(entity.DataSyncModeIncrementalHardDel)
	task.UpdField = "id"

	require.NoError(t, app.SyncTask(context.Background(), task, src, dst, &entity.DataSyncLog{}, nil))
	require.Equal(t, 5, countRows(t, dst, modeItTgtTable))

	task.UpdFieldVal = "5"
	mustExec(t, src, fmt.Sprintf("DELETE FROM %s WHERE id = 2", quote(src, modeItSrcTable)))
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (6,'frank',1),(7,'grace',1)", modeItSrcTable))

	syncLog := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(context.Background(), task, src, dst, syncLog, nil))
	assert.NotContains(t, strings.ToLower(syncLog.DataSQLFull), "id >", "删除对账不得叠加水位增量条件，实际SQL: %s", syncLog.DataSQLFull)
	assert.Equal(t, []int64{1, 3, 4, 5, 6, 7}, queryIds(t, dst, modeItTgtTable), "目标应等于源现存集合，未变更行不得被误删")
}
