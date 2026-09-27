//go:build it

package itest

// 交付语义（CursorInclusivity）端到端回归：
// 水位推进到 cursor=X 之后，源侧新插入一行 cursor 值仍是 X 的边界行，
// Inclusive（>=）能拉到，Exclusive（>）会漏拉。这是 P0 修正的"同秒时间戳/同版本号"漏数据场景。

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
	inclSrcTable = "it_sync_incl_cursor_src"
	inclTgtTable = "it_sync_incl_cursor_tgt"
)

// setupInclusiveCursorTable mysql 源 / pg 目标，version 作 cursor 列（非唯一，允许多行同值）
func setupInclusiveCursorTable(t *testing.T) (*sync.DataSyncAppImpl, *dbi.DbConn, *dbi.DbConn) {
	t.Helper()
	src := mysqlConn(t)
	dst := pgConn(t)
	t.Cleanup(func() { src.Close(); dst.Close() })

	mustExec(t, src, "DROP TABLE IF EXISTS "+quote(src, inclSrcTable))
	mustExec(t, src, fmt.Sprintf("CREATE TABLE %s (id int primary key, version int, name varchar(50))", quote(src, inclSrcTable)))
	mustExec(t, dst, "DROP TABLE IF EXISTS "+quote(dst, inclTgtTable))
	mustExec(t, dst, fmt.Sprintf("CREATE TABLE %s (id int primary key, version int, name varchar(50))", quote(dst, inclTgtTable)))

	app, _ := newApp()
	return app, src, dst
}

func inclTask(id uint64, inclusivity entity.CursorInclusivity) *entity.DataSyncTask {
	task := &entity.DataSyncTask{
		Id:              id,
		TaskName:        "it-incl-cursor",
		SrcDbId:         1,
		SrcDbName:       "mayfly_dbm_it",
		TargetDbId:      2,
		TargetDbName:    "mayfly_pg_it",
		TargetTableName: inclTgtTable,
		FieldMap:        `[{"src":"id","target":"id"},{"src":"version","target":"version"},{"src":"name","target":"name"}]`,
		PageSize:        100,
		SyncMode:        entity.DataSyncModeIncrementalMerge,
		UpdField:        "version",
		UpdFieldVal:     "2",
		DataSQL:         fmt.Sprintf("SELECT id, version, name FROM %s", inclSrcTable),
	}
	// CursorInclusivity 已存于 Extra（非查询维度不建列），需 setter写入
	task.SetCursorInclusivity(inclusivity)
	return task
}

func runInclSync(t *testing.T, app *sync.DataSyncAppImpl, task *entity.DataSyncTask, src, dst *dbi.DbConn) {
	t.Helper()
	_, err := src.Exec(task.DataSQL)
	require.NoError(t, err)
	log := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(context.Background(), task, src, dst, log, nil))
}

// TestITInclusiveCursorPullsSameVersionBoundaryRow：水位=2 时源侧新插入 version=2 的行，
// Inclusive 能拉到该边界行（at-least-once 语义 + UPSERT 幂等），目标出现 id=3 的边界行。
func TestITInclusiveCursorPullsSameVersionBoundaryRow(t *testing.T) {
	app, src, dst := setupInclusiveCursorTable(t)
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (1,1,'alice'),(2,2,'bob')", inclSrcTable))
	runInclSync(t, app, inclTask(9921, entity.CursorInclusivityInclusive), src, dst)

	// 首轮水位推进到 2；插入边界行 version=2（与水位同值）
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (3,2,'carl')", inclSrcTable))
	task := inclTask(9921, entity.CursorInclusivityInclusive)
	task.UpdFieldVal = "2"
	runInclSync(t, app, task, src, dst)

	_, rows, err := dst.Query("SELECT id, name FROM " + quote(dst, inclTgtTable) + " WHERE id = 3")
	require.NoError(t, err)
	require.Len(t, rows, 1, "Inclusive 应把 version=2 的边界新行拉到目标库")
	assert.Equal(t, "carl", fmt.Sprint(rows[0]["name"]))
}

// TestITExclusiveCursorSkipsSameVersionBoundaryRow：同数据、水位=2、模式 Merge 但显式 Exclusive，
// 边界行 version=2 因 `> 2` 被跳过。锁死两种语义的可区分性——若未来 inclusive 逻辑被误改，
// 上/下两测试会同步失败指向该回归。
func TestITExclusiveCursorSkipsSameVersionBoundaryRow(t *testing.T) {
	app, src, dst := setupInclusiveCursorTable(t)
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (1,1,'alice'),(2,2,'bob')", inclSrcTable))
	runInclSync(t, app, inclTask(9922, entity.CursorInclusivityExclusive), src, dst)

	mustExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (3,2,'carl')", inclSrcTable))
	task := inclTask(9922, entity.CursorInclusivityExclusive)
	task.UpdFieldVal = "2"
	runInclSync(t, app, task, src, dst)

	_, rows, err := dst.Query("SELECT id FROM " + quote(dst, inclTgtTable) + " WHERE id = 3")
	require.NoError(t, err)
	assert.Empty(t, rows, "Exclusive 会漏 version=2 的边界新行（这正是 P0 修的场景）")
}
