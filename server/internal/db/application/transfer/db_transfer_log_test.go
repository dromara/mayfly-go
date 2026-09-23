package transfer

// 迁移执行日志收尾的回归测试。
// 缺陷现场：校验失败的迁移日志一直显示「执行中」——日志「失败」状态取值为0（int8零值），
// 而收尾是以结构体更新落库，GORM 会跳过零值字段，导致状态永远写不进库。
//
// 运行：cd server && go test -count=1 -run TestTransferLog ./internal/db/application/transfer/

import (
	"context"
	"errors"
	"testing"
	"time"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// stub 仓储：只借用 RepoImpl 的通用增删改查，业务仓储方法在本测试中无用例
type stubTransferLogRepo struct {
	*base.RepoImpl[*entity.DbTransferLog]
}

func (s *stubTransferLogRepo) GetPageList(*entity.DbTransferLogQuery, ...string) (*model.PageResult[*entity.DbTransferLog], error) {
	return nil, nil
}

type stubTransferTaskRepo struct {
	*base.RepoImpl[*entity.DbTransferTask]
}

func (s *stubTransferTaskRepo) GetPageList(*entity.DbTransferTaskQuery, ...string) (*model.PageResult[*entity.DbTransferTask], error) {
	return nil, nil
}

// setupTransferLogDb 内存 sqlite 建表并接管全局 db（gormx 默认使用 global.Db）
func setupTransferLogDb(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entity.DbTransferLog{}, &entity.DbTransferTask{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })
	return db
}

func newLogTestApp() *DbTransferAppImpl {
	app := &DbTransferAppImpl{}
	app.Repo = &stubTransferTaskRepo{RepoImpl: &base.RepoImpl[*entity.DbTransferTask]{}}
	app.transferLogRepo = &stubTransferLogRepo{RepoImpl: &base.RepoImpl[*entity.DbTransferLog]{}}
	return app
}

// newTestTask 插入一条运行中的迁移任务
func newTestTask(t *testing.T, db *gorm.DB) *entity.DbTransferTask {
	t.Helper()
	task := &entity.DbTransferTask{TaskName: "t1", TaskKey: "k1", RunningState: entity.DbTransferTaskRunStateRunning}
	task.FillBaseInfo(model.IdGenTypeNone, nil)
	require.NoError(t, db.Create(task).Error)
	return task
}

// insertRunningLog 插入一条处于「执行中」的日志
func insertRunningLog(t *testing.T, db *gorm.DB, taskId uint64) uint64 {
	t.Helper()
	now := time.Now()
	log := &entity.DbTransferLog{CreateTime: &now, TaskId: taskId, Mode: entity.DbTransferTaskModeDb, Status: entity.DbTransferLogStatusRunning}
	require.NoError(t, db.Create(log).Error)
	// 日志缓存为进程级，需清掉上一次用例留下的同id副本
	cache.Del(getTransferLogKey(log.Id))
	return log.Id
}

func getLog(t *testing.T, db *gorm.DB, logId uint64) *entity.DbTransferLog {
	t.Helper()
	log := new(entity.DbTransferLog)
	require.NoError(t, db.First(log, logId).Error)
	return log
}

// TestEndTransferFailPersistsFailStatus 执行失败收尾：日志状态必须落库为「失败」，而非停留在「执行中」
func TestEndTransferFailPersistsFailStatus(t *testing.T) {
	db := setupTransferLogDb(t)
	app := newLogTestApp()
	ctx := context.TODO()

	task := newTestTask(t, db)
	logId := insertRunningLog(t, db, task.Id)

	app.EndTransfer(ctx, logId, task.Id, "transfer table failed", errors.New("not all tables match"), nil)

	assert.Equal(t, entity.DbTransferLogStatusFail, getLog(t, db, logId).Status, "失败状态未落库，日志会一直显示「执行中」")
	assert.Equal(t, "not all tables match", getLog(t, db, logId).ErrText)
	assert.Contains(t, getLog(t, db, logId).RunLog, "迁移执行失败")

	require.NoError(t, db.First(task, task.Id).Error)
	assert.Equal(t, entity.DbTransferTaskRunStateFail, task.RunningState)
}

// TestEndTransferSuccessPersistsSuccessStatus 执行成功收尾同样落库
func TestEndTransferSuccessPersistsSuccessStatus(t *testing.T) {
	db := setupTransferLogDb(t)
	app := newLogTestApp()

	task := newTestTask(t, db)
	logId := insertRunningLog(t, db, task.Id)

	app.EndTransfer(context.TODO(), logId, task.Id, "transfer complete", nil, nil)

	assert.Equal(t, entity.DbTransferLogStatusSuccess, getLog(t, db, logId).Status)
	assert.Contains(t, getLog(t, db, logId).RunLog, "迁移执行完成")
}

// TestEndVerifyUsesVerifyWording 校验收尾的摘要文案必须是校验措辞：
// 校验全程只读不迁移数据，复用「迁移执行完成」会让人误以为本次执行写入了数据。
// 同时校验收尾摘要与执行过程日志一起落库（收尾摘要若只写进缓存副本，日志就看不到最终结果）
func TestEndVerifyUsesVerifyWording(t *testing.T) {
	db := setupTransferLogDb(t)
	app := newLogTestApp()
	ctx := context.TODO()

	task := newTestTask(t, db)
	mismatchLogId := insertRunningLog(t, db, task.Id)
	app.Log(ctx, mismatchLogId, "verify table [user_account] mismatch: src=2 target=1")
	app.EndVerify(ctx, mismatchLogId, task.Id, "data verification complete", errors.New("verification failed: not all tables match"))

	mismatchLog := getLog(t, db, mismatchLogId)
	assert.Equal(t, entity.DbTransferLogStatusFail, mismatchLog.Status)
	assert.Contains(t, mismatchLog.RunLog, "数据校验未通过")
	assert.Contains(t, mismatchLog.RunLog, "verify table [user_account] mismatch")
	assert.NotContains(t, mismatchLog.RunLog, "迁移执行")

	matchLogId := insertRunningLog(t, db, task.Id)
	app.EndVerify(ctx, matchLogId, task.Id, "data verification complete", nil)

	matchLog := getLog(t, db, matchLogId)
	assert.Equal(t, entity.DbTransferLogStatusSuccess, matchLog.Status)
	assert.Contains(t, matchLog.RunLog, "数据校验完成")
	assert.NotContains(t, matchLog.RunLog, "迁移执行")
}

// TestTransferLogFailStatusIsNonZero 失败状态必须为非零值：
// 收尾以结构体更新落库，零值字段会被 GORM 跳过（不显式指定列时的对照用例见下一用例）
func TestTransferLogFailStatusIsNonZero(t *testing.T) {
	assert.NotEqual(t, int8(0), entity.DbTransferLogStatusFail)
	assert.Contains(t, transferLogEndColumns, "status")
}

// TestTransferLogEndWithoutColumnsDropsZeroStatus 对照用例：不显式指定收尾列时零值状态被丢弃
func TestTransferLogEndWithoutColumnsDropsZeroStatus(t *testing.T) {
	db := setupTransferLogDb(t)
	repo := &base.RepoImpl[*entity.DbTransferLog]{}

	logId := insertRunningLog(t, db, 1)
	end := new(entity.DbTransferLog)
	end.Id = logId
	end.RunLog = "迁移执行失败"
	require.NoError(t, repo.UpdateById(context.TODO(), end))

	assert.Equal(t, entity.DbTransferLogStatusRunning, getLog(t, db, logId).Status, "零值字段被跳过正是缺陷成因")
}

// TestResetStaleRunningLogs 启动收尾：无人收尾的「执行中」日志置为失败，已终态日志不受影响
func TestResetStaleRunningLogs(t *testing.T) {
	db := setupTransferLogDb(t)
	app := newLogTestApp()

	staleLogId := insertRunningLog(t, db, 1)
	now := time.Now()
	doneLog := &entity.DbTransferLog{CreateTime: &now, TaskId: 2, Mode: entity.DbTransferTaskModeDb, Status: entity.DbTransferLogStatusSuccess}
	require.NoError(t, db.Create(doneLog).Error)

	require.NoError(t, app.ResetStaleRunningLogs(context.TODO()))

	assert.Equal(t, entity.DbTransferLogStatusFail, getLog(t, db, staleLogId).Status)
	assert.Equal(t, entity.DbTransferLogStatusSuccess, getLog(t, db, doneLog.Id).Status)
}

// TestCreateLogPurposeDistinguishesRunKind 日志用途必须区分迁移/导出文件/校验：
// 校验记录若沿用任务 Mode 会被标成「迁移」，且行数/表数恒为 0，列表无法辨识。
func TestCreateLogPurposeDistinguishesRunKind(t *testing.T) {
	db := setupTransferLogDb(t)
	app := newLogTestApp()
	ctx := context.TODO()

	task := newTestTask(t, db)

	// 库模式迁移 → 用途为迁移
	transferLogId, err := app.CreateLog(ctx, task.Id, transferPurposeOfMode(entity.DbTransferTaskModeDb))
	require.NoError(t, err)
	assert.Equal(t, entity.DbTransferLogPurposeTransfer, getLog(t, db, transferLogId).Purpose)

	// 文件模式迁移 → 用途为导出文件
	assert.Equal(t, entity.DbTransferLogPurposeExport, transferPurposeOfMode(entity.DbTransferTaskModeFile))

	// 校验 → 用途为校验，且回填表数/行数指标（收尾后随 transferLogEndColumns 落库）
	verifyLogId, err := app.CreateLog(ctx, task.Id, entity.DbTransferLogPurposeVerify)
	require.NoError(t, err)
	cache.Del(getTransferLogKey(verifyLogId))
	app.setVerifyMetrics(verifyLogId, &TransferVerifyReport{Results: []TableVerifyResult{{SrcCount: 5}, {SrcCount: 7}}})
	app.EndVerify(ctx, verifyLogId, task.Id, "data verification complete", nil)

	vlog := getLog(t, db, verifyLogId)
	assert.Equal(t, entity.DbTransferLogPurposeVerify, vlog.Purpose)
	assert.Equal(t, 2, vlog.TableCount, "校验表数未落库")
	assert.Equal(t, int64(12), vlog.TotalRows, "校验涉及行数未落库")
}
