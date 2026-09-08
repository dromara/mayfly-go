//go:build it

package sync

// 数据同步水位持久化集成测试：批级水位落库、两轮增量同步、运行中中断后水位不回退。
//
// 运行方式：cd server && go test -tags it -count=1 -v -run TestITDataSyncWatermark ./internal/db/application/
//
// 注意：dbm.Conn对相同连接信息复用同一连接池，源与目标必须使用两个不同数据库（mysql源 + pg目标）。

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
)

const (
	syncItSrcTable = "it_sync_src"
	syncItTgtTable = "it_sync_tgt"
	syncItTaskId   = uint64(9901)
)

// fakeSyncLogRepo 同步日志仓储fake（仅覆写Save，吞掉日志写入）
type fakeSyncLogRepo struct {
	repository.DataSyncLog

	logs []*entity.DataSyncLog
}

func (f *fakeSyncLogRepo) Save(ctx context.Context, log *entity.DataSyncLog) error {
	f.logs = append(f.logs, log)
	return nil
}

// fakeSyncDbApp 按dbId返回预置连接（仅覆写GetDbConn，其余方法调用即panic暴露问题）
type fakeSyncDbApp struct {
	srcId, tgtId     uint64
	srcConn, tgtConn *dbi.DbConn
}

func (f *fakeSyncDbApp) GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error) {
	if dbId == f.srcId {
		return f.srcConn, nil
	}
	return f.tgtConn, nil
}

func setupSyncWatermarkIT(t *testing.T) (*DataSyncAppImpl, *fakeDataSyncRepo, *dbi.DbConn, *dbi.DbConn) {
	t.Helper()
	srcConn := verifyMysqlConn(t)
	tgtConn := verifyPgConn(t)
	t.Cleanup(func() { srcConn.Close(); tgtConn.Close() })

	quoteSrc := srcConn.GetDialect().Quoter().Quote
	quoteTgt := tgtConn.GetDialect().Quoter().Quote

	// 源表（mysql）与目标表（pg）同构
	_, err := srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteSrc(syncItSrcTable)))
	require.NoError(t, err)
	_, err = srcConn.Exec(fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50))", quoteSrc(syncItSrcTable)))
	require.NoError(t, err)

	_, err = tgtConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteTgt(syncItTgtTable)))
	require.NoError(t, err)
	_, err = tgtConn.Exec(fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50))", quoteTgt(syncItTgtTable)))
	require.NoError(t, err)

	insertRows(t, srcConn, syncItSrcTable, 1, 10)

	fake := &fakeDataSyncRepo{}
	app := &DataSyncAppImpl{
		AppImpl: base.AppImpl[*entity.DataSyncTask, repository.DataSyncTask]{Repo: fake},
	}
	app.dbApp = &fakeSyncDbApp{srcId: 1, tgtId: 2, srcConn: srcConn, tgtConn: tgtConn}
	app.dbDataSyncLogRepo = &fakeSyncLogRepo{}
	return app, fake, srcConn, tgtConn
}

// insertRows 向mysql源表插入 [from, to] 闭区间行
func insertRows(t *testing.T, conn *dbi.DbConn, table string, from, to int) {
	t.Helper()
	values := make([]string, 0, to-from+1)
	for i := from; i <= to; i++ {
		values = append(values, fmt.Sprintf("(%d, 'name-%d')", i, i))
	}
	_, err := conn.Exec(fmt.Sprintf("INSERT INTO %s VALUES %s", conn.GetDialect().Quoter().Quote(table), strings.Join(values, ",")))
	require.NoError(t, err)
}

func countSyncRows(t *testing.T, conn *dbi.DbConn, table string) int {
	t.Helper()
	_, rows, err := conn.Query(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", conn.GetDialect().Quoter().Quote(table)))
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	cnt, _ := dbi.ValToInt64(rows[0]["cnt"])
	return int(cnt)
}

// newSyncTask 构造增量同步任务（PageSize=3制造多批，验证批级水位）
func newSyncTask(watermark string) *entity.DataSyncTask {
	return &entity.DataSyncTask{
		Id:              syncItTaskId,
		TaskName:        "it-sync-watermark",
		SrcDbId:         1,
		SrcDbName:       "mayfly_dbm_it",
		TargetDbId:      2,
		TargetDbName:    "mayfly_pg_it",
		TargetTableName: syncItTgtTable,
		FieldMap:        `[{"src":"id","target":"id"},{"src":"name","target":"name"}]`,
		PageSize:        3,
		UpdField:        "id",
		UpdFieldVal:     watermark,
	}
}

// TestITDataSyncWatermark 两轮增量同步 + 运行中中断后水位不回退
func TestITDataSyncWatermark(t *testing.T) {
	app, fake, srcConn, tgtConn := setupSyncWatermarkIT(t)
	ctx := context.Background()

	// ===== 第一轮全量（10行，PageSize=3 → 3个满批+1尾批）=====
	app.runGuard.Acquire(syncItTaskId)
	task := newSyncTask("0")
	fullSql := fmt.Sprintf("select id, name from %s where 1=1 order by id asc", syncItSrcTable)
	log1 := &entity.DataSyncLog{}
	require.NoError(t, app.doDataSync(ctx, fullSql, task, log1))
	assert.Equal(t, 10, log1.ResNum)
	assert.Equal(t, "10", task.UpdFieldVal, "水位应为末行id")
	assert.Equal(t, 10, countSyncRows(t, tgtConn, syncItTgtTable))

	// 批级水位持久化触发：4次（3满批+1尾批）
	require.NotEmpty(t, fake.updated)
	if len(fake.updated) != 4 {
		t.Fatalf("expected 4 batch watermark persists, got %d", len(fake.updated))
	}
	// 水位单调不回退
	last := int64(0)
	for _, ut := range fake.updated {
		v, _ := dbi.ValToInt64(ut.UpdFieldVal)
		if v < last {
			t.Fatalf("watermark regressed: %d -> %d", last, v)
		}
		last = v
	}

	// ===== 第二轮增量（新增11~13，仅增量行被同步）=====
	app.runGuard.Release(syncItTaskId)
	insertRows(t, srcConn, syncItSrcTable, 11, 13)

	app.runGuard.Acquire(syncItTaskId)
	task2 := newSyncTask("10")
	incrSql := fmt.Sprintf("select id, name from %s where 1=1 and id > 10 order by id asc", syncItSrcTable)
	log2 := &entity.DataSyncLog{}
	require.NoError(t, app.doDataSync(ctx, incrSql, task2, log2))
	assert.Equal(t, 3, log2.ResNum, "第二轮应仅同步增量行")
	assert.Equal(t, "13", task2.UpdFieldVal)
	assert.Equal(t, 13, countSyncRows(t, tgtConn, syncItTgtTable))

	// ===== 第三轮：运行中被终止 → 已提交批次水位已落库且不回退 =====
	app.runGuard.Release(syncItTaskId)
	insertRows(t, srcConn, syncItSrcTable, 14, 17)

	// 不Acquire（模拟StopTask已Release）：第一批（14,15,16）提交后检测到终止
	task3 := newSyncTask("13")
	incrSql2 := fmt.Sprintf("select id, name from %s where 1=1 and id > 13 order by id asc", syncItSrcTable)
	log3 := &entity.DataSyncLog{}
	err := app.doDataSync(ctx, incrSql2, task3, log3)
	require.Error(t, err, "终止后应返回错误中断同步")
	assert.Contains(t, err.Error(), "terminated")
	// 第一批3行已提交，剩余1行未同步
	assert.Equal(t, 16, countSyncRows(t, tgtConn, syncItTgtTable))
	// 水位推进到已提交批次末行16，未回退
	assert.Equal(t, "16", task3.UpdFieldVal)
	finalWatermark, _ := dbi.ValToInt64(fake.updated[len(fake.updated)-1].UpdFieldVal)
	assert.Equal(t, int64(16), finalWatermark,
		"中断后最后持久化水位应为已提交批次的水位")
}
