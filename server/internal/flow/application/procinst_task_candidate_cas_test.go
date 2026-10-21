package application

import (
	"context"
	"os"
	"testing"

	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/i18n"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newCandidateCasApp 用真实仓储（SQLite）装配只含候选人仓储的应用实例
func newCandidateCasApp(t *testing.T) (*procinstTaskAppImpl, repository_CandidateRepo) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entity.ProcinstTaskCandidate{}))

	previous := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = previous })

	repo := &base.RepoImpl[*entity.ProcinstTaskCandidate]{}
	app := &procinstTaskAppImpl{}
	app.procinstTaskCandidateRepo = repo
	return app, repo
}

// repository_CandidateRepo 只是类型别名，避免测试里反复书写泛型
type repository_CandidateRepo = base.Repo[*entity.ProcinstTaskCandidate]

// TestCompleteTaskCandidateRejectsDoubleClick 候选人行的终态写入必须按 CAS 判赢。
//
// 缺陷类（真实发生过）：UpdateById 只按 id 更新、不回报行数。同一账号双击（或两个并发请求）
// 会两次都写成功，把会签票数 +2、把别人刚写的意见覆盖掉。
func TestCompleteTaskCandidateRejectsDoubleClick(t *testing.T) {
	app, repo := newCandidateCasApp(t)
	ctx := context.Background()

	row := &entity.ProcinstTaskCandidate{ProcinstId: 1, TaskId: 1, Candidate: "5", Status: entity.ProcinstTaskStatusProcess}
	require.NoError(t, repo.Insert(ctx, row))

	done := &entity.ProcinstTaskCandidate{Id: row.Id, Status: entity.ProcinstTaskStatusCompleted}
	handler := "alice"
	done.Handler = &handler
	done.Remark = "同意"
	require.NoError(t, app.completeTaskCandidate(ctx, done))

	// 第二次（双击/并发第二个请求）必须被拒：条件 status=Process 已不成立
	err := app.completeTaskCandidate(ctx, done)
	require.Error(t, err)
	require.Contains(t, err.Error(), i18n.T(imsg.ErrProcinstTaskAlreadyHandled),
		"第二次完成必须报「已被处理」，不能静默成功")

	// 票据数从库里派生：只有 1 行完成
	n, err := app.countCompletedCandidates(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, n)
}

// TestCountCompletedIsDerivedFromRows 票数必须来自库里的已完成行，而不是内存自增。
//
// 缺陷类：旧实现 `vars.GetInt(NrOfCompleted) + len(taskCandidates)` 是读-改-写——
// 4 人并发各写 1 票最终只落库 1，剩余候选人行已终态、进程行已删，节点永久卡死。
func TestCountCompletedIsDerivedFromRows(t *testing.T) {
	app, repo := newCandidateCasApp(t)
	ctx := context.Background()

	for i, candidate := range []string{"5", "6", "7"} {
		require.NoError(t, repo.Insert(ctx, &entity.ProcinstTaskCandidate{
			ProcinstId: 2, TaskId: 2, Candidate: candidate, Status: entity.ProcinstTaskStatusProcess,
		}))
		_ = i
	}
	rows, err := repo.SelectByCond(&entity.ProcinstTaskCandidate{TaskId: 2})
	require.NoError(t, err)
	require.Len(t, rows, 3)

	// 完成前两行后派生计数必须是 2（若退回内存自增，测试注入的初始值无从得知，会不成立）
	for _, r := range rows[:2] {
		done := &entity.ProcinstTaskCandidate{Id: r.Id, Status: entity.ProcinstTaskStatusCompleted}
		require.NoError(t, app.completeTaskCandidate(ctx, done))
	}
	n, err := app.countCompletedCandidates(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, 2, n)
}

// TestTaskCompletionUsesSingleWinningCAS 源码守卫：三处候选人更新必须走 CAS，计数不得自增。
func TestTaskCompletionUsesSingleWinningCAS(t *testing.T) {
	raw, err := os.ReadFile("procinst_task.go")
	require.NoError(t, err)
	text := stripLineCommentsForFlowGuard(string(raw))

	require.NotContains(t, text, "procinstTaskCandidateRepo.UpdateById(ctx, taskCandidate)",
		"候选人行退回 UpdateById：不判行数，双击会重复计票")
	require.NotContains(t, text, "vars.GetInt(NrOfCompleted) + len(taskCandidates)",
		"会签计数退回读-改-写：并发会丢票并让节点卡死")
	require.Contains(t, text, "countCompletedCandidates(ctx, taskId)",
		"票数必须从库里派生")
}
