//go:build it

package itest

// 数据同步水位集成测试（黑盒，经 SyncTask）：批级水位落库次数、两轮增量、水位单调不回退。
// 运行中被手动终止（依赖 runGuard 私有守卫）的生命周期用例见 sync/internal_lifecycle_it_test.go。

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dbi/value"
	"mayfly-go/internal/db/domain/entity"
)

const (
	wmSrcTable = "it_sync_wm_src"
	wmTgtTable = "it_sync_wm_tgt"
	wmTaskId   = uint64(9901)
)

func wmInsertRows(t *testing.T, c *dbi.DbConn, from, to int) {
	t.Helper()
	values := make([]string, 0, to-from+1)
	for i := from; i <= to; i++ {
		values = append(values, fmt.Sprintf("(%d, 'name-%d')", i, i))
	}
	mustExec(t, c, fmt.Sprintf("INSERT INTO %s VALUES %s", quote(c, wmSrcTable), strings.Join(values, ",")))
}

func setupWatermark(t *testing.T) (*sync.DataSyncAppImpl, *fakeRepo, *dbi.DbConn, *dbi.DbConn) {
	t.Helper()
	src := mysqlConn(t)
	dst := pgConn(t)
	t.Cleanup(func() { src.Close(); dst.Close() })

	mustExec(t, src, "DROP TABLE IF EXISTS "+quote(src, wmSrcTable))
	mustExec(t, src, fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50))", quote(src, wmSrcTable)))
	mustExec(t, dst, "DROP TABLE IF EXISTS "+quote(dst, wmTgtTable))
	mustExec(t, dst, fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50))", quote(dst, wmTgtTable)))
	wmInsertRows(t, src, 1, 10)

	app, fake := newApp()
	return app, fake, src, dst
}

func wmTask(watermark string) *entity.DataSyncTask {
	return &entity.DataSyncTask{
		Id:              wmTaskId,
		TaskName:        "it-sync-watermark",
		SrcDbId:         1,
		SrcDbName:       "mayfly_dbm_it",
		TargetDbId:      2,
		TargetDbName:    "mayfly_pg_it",
		TargetTableName: wmTgtTable,
		FieldMap:        `[{"src":"id","target":"id"},{"src":"name","target":"name"}]`,
		PageSize:        3,
		UpdField:        "id",
		UpdFieldVal:     watermark,
		DataSQL:         fmt.Sprintf("select id, name from %s", wmSrcTable),
	}
}

func TestITDataSyncWatermark(t *testing.T) {
	app, fake, src, dst := setupWatermark(t)
	ctx := context.Background()

	// 第一轮全量（10行，PageSize=3 → 3满批+1尾批）
	task := wmTask("0")
	log1 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log1, nil))
	assert.Equal(t, 10, log1.ResNum)
	assert.Equal(t, "10", task.UpdFieldVal)
	assert.Equal(t, 10, countRows(t, dst, wmTgtTable))
	require.Len(t, fake.updated, 4, "批级水位应持久化4次")
	last := int64(0)
	for _, ut := range fake.updated {
		v, _ := value.ValToInt64(ut.UpdFieldVal)
		assert.GreaterOrEqual(t, v, last, "水位不得回退")
		last = v
	}

	// 第二轮增量（新增11~13）
	wmInsertRows(t, src, 11, 13)
	task2 := wmTask("10")
	log2 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(ctx, task2, src, dst, log2, nil))
	assert.Equal(t, 3, log2.ResNum)
	assert.Equal(t, "13", task2.UpdFieldVal)
	assert.Equal(t, 13, countRows(t, dst, wmTgtTable))
}
