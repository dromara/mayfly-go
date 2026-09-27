package sync

import (
	"context"
	"strings"
	"testing"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetLogRepo 仅覆写 ResetStaleRunningSyncLogs 用到的两个方法：
// SelectByCond 返回预置的 Running 日志集合，UpdateById 记录被收尾的日志 id；未覆写方法被调用即 panic，暴露越界依赖。
type resetLogRepo struct {
	repository.DataSyncLog
	running []*entity.DataSyncLog
	updated map[uint64]*entity.DataSyncLog
}

func (r *resetLogRepo) SelectByCond(_ any, _ ...string) ([]*entity.DataSyncLog, error) {
	return r.running, nil
}

func (r *resetLogRepo) UpdateById(_ context.Context, e *entity.DataSyncLog, _ ...string) error {
	cp := *e
	r.updated[e.Id] = &cp
	return nil
}

// TestResetStaleRunningSyncLogsByRunId 启动收尾的归属判定：
//   - 当前锁持有 runId 与日志 RunId 一致：真活跃，跳过；
//   - 当前锁持有不同 runId：旧崩溃遗留，收尾（新接管者会写自己的新日志）；
//   - 无锁：收尾；
//   - 日志 RunId 为空（升级前数据）：无法匹配任何锁，收尾（等价于「无 runId」的历史行）。
//
// 锁未 Acquire 时 IsCurrentRun 一律 false（本实例也没持有），所以「无锁 → 收尾」这一支不需要显式 Acquire 反证。
func TestResetStaleRunningSyncLogsByRunId(t *testing.T) {
	app := &DataSyncAppImpl{}
	repo := &resetLogRepo{updated: map[uint64]*entity.DataSyncLog{}}
	app.dbDataSyncLogRepo = repo

	// 任务 1 已被本次执行持有：Acquire 拿到 runId 并让该 runId 挂在一条 Running 日志上；
	// 同任务的另一条日志带不同 runId，模拟「本实例已重新执行、旧一轮遗留」。
	// 使用高位段 taskId 避免与 sync/itest 用例的小 id 冲突（TestMain 里 sync 也初始化了 Redis）
	const activeTask, staleTask, noRunTask = uint64(98201), uint64(98202), uint64(98203)
	runId1, ok := app.runGuard.AcquireWithRunId(activeTask)
	require.True(t, ok, "acquire active task should succeed")
	t.Cleanup(func() { app.runGuard.ReleaseWithRunId(activeTask, runId1) })
	repo.running = []*entity.DataSyncLog{
		{IdModel: idModel(10), TaskId: activeTask, RunId: runId1, Status: entity.DataSyncTaskStateRunning}, // 活跃：跳过
		{IdModel: idModel(11), TaskId: activeTask, RunId: "stale-other-run", Status: entity.DataSyncTaskStateRunning},
		{IdModel: idModel(12), TaskId: staleTask, RunId: "", Status: entity.DataSyncTaskStateRunning}, // 旧行无 runId
		{IdModel: idModel(13), TaskId: noRunTask, RunId: "no-lock-at-all", Status: entity.DataSyncTaskStateRunning},
	}

	require.NoError(t, app.ResetStaleRunningSyncLogs(context.Background()))

	assert.NotContains(t, repo.updated, uint64(10), "活跃日志（runId 与当前锁匹配）不应被收尾")
	for _, id := range []uint64{11, 12, 13} {
		upd, seen := repo.updated[id]
		require.True(t, seen, "失活日志 %d 应被收尾", id)
		assert.Equal(t, entity.DataSyncTaskStateFail, upd.Status)
		assert.NotEmpty(t, upd.ErrText)
	}
}

// idModel 便捷构造内嵌 model.IdModel（entity 内嵌，字段名 Id）。
func idModel(id uint64) model.IdModel {
	return model.IdModel{Id: id}
}

// TestEndRunningSkipsTaskUpdateWhenPreempted 收尾的任务级状态写入必须以 runId 归属为门：
// 若本 run 已被新实例接管，旧收尾不得覆盖 RunningState/UpdFieldVal。
func TestEndRunningSkipsTaskUpdateWhenPreempted(t *testing.T) {
	// 不 Acquire，runGuard 无本 run 的持有权；endRunning 传入的 log.RunId 与任何锁都不匹配。
	app := &DataSyncAppImpl{}
	app.Repo = &fakeDataSyncRepo{}
	app.dbDataSyncLogRepo = &lcNoopLogRepo{}
	log := &entity.DataSyncLog{Status: entity.DataSyncTaskStateRunning, RunId: "not-current"}
	app.endRunning(&entity.DataSyncTask{Id: 42}, log)
	// 断言收尾仍把日志落终态（saveLog 被调用；lcNoopLogRepo 会记录），但任务状态未被更新（fakeDataSyncRepo.updated 为空）
	assert.Equal(t, entity.DataSyncTaskStateFail, log.Status, "日志行本身仍要落终态")
	assert.Empty(t, app.Repo.(*fakeDataSyncRepo).updated, "非当前 run 不得覆盖任务级状态")
}

// lcNoopLogRepo 吞掉 Save 落库，让 endRunning 走完路径而不需要真库。
type lcNoopLogRepo struct {
	repository.DataSyncLog
}

func (lcNoopLogRepo) Save(context.Context, *entity.DataSyncLog) error { return nil }
func (lcNoopLogRepo) UpdateById(context.Context, *entity.DataSyncLog, ...string) error {
	return nil
}

// TestEndRunningAppendsFailureReasonToRunLog 失败原因必须落到 RunLog 弹窗可见字段：
// 线上真实场景——目标表 0 列时 doDataSync 在 [5/6]-[6/6] 之间 return error，
// ErrText 列有内容但 RunLog 只到 [5/6]，用户点"运行日志"完全看不到失败原因。
// endRunning 现在把 ErrText 追加到 RunLog 末尾，本用例锁死该行为。
func TestEndRunningAppendsFailureReasonToRunLog(t *testing.T) {
	app := &DataSyncAppImpl{}
	app.Repo = &fakeDataSyncRepo{}
	saved := &entity.DataSyncLog{}
	app.dbDataSyncLogRepo = &captureSaveRepo{into: saved}

	log := &entity.DataSyncLog{
		Status:  entity.DataSyncTaskStateFail,
		ErrText: "field map target column [id] not found in target table [test_sync_target]",
		RunLog:  "[5/6] 检测Schema变更...\n- Schema检测已禁用\n",
	}
	app.endRunning(&entity.DataSyncTask{Id: 98301}, log)

	assert.Contains(t, saved.RunLog, log.ErrText, "RunLog 末尾必须包含完整错误文本")
	assert.Contains(t, saved.RunLog, "========================================", "RunLog 摘要行需以分隔线包起")
	// 双重前缀回归守卫：ErrText 已含"执行失败:"，endRunning 不再叠加"同步执行失败:"
	assert.NotContains(t, saved.RunLog, "同步执行失败: 执行失败")
}

// TestEndRunningDoesNotDuplicateExistingErrorText 若某条错误路径已经把 ErrText 写进 RunLog，
// endRunning 不重复追加，避免弹窗里同一错误出现两次。
func TestEndRunningDoesNotDuplicateExistingErrorText(t *testing.T) {
	app := &DataSyncAppImpl{}
	app.Repo = &fakeDataSyncRepo{}
	saved := &entity.DataSyncLog{}
	app.dbDataSyncLogRepo = &captureSaveRepo{into: saved}

	errText := "boom"
	log := &entity.DataSyncLog{
		Status:  entity.DataSyncTaskStateFail,
		ErrText: errText,
		RunLog:  "已经打过: " + errText + "\n",
	}
	app.endRunning(&entity.DataSyncTask{Id: 98302}, log)

	// 已含错误文本时不再追加，也就不会出现第二次分隔线
	assert.Equal(t, 1, strings.Count(saved.RunLog, "已经打过"), "RunLog 未被 endRunning 二次追加")
}

// captureSaveRepo 记录最后一次 Save 的实体副本，供断言 RunLog 内容。
type captureSaveRepo struct {
	repository.DataSyncLog
	into *entity.DataSyncLog
}

func (c *captureSaveRepo) Save(_ context.Context, l *entity.DataSyncLog) error {
	*c.into = *l
	return nil
}
