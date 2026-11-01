package taskx

import (
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/pkg/rediscli"
)

// TestRequestStopWorksWithoutRedis 无 Redis（单实例部署）时停止也必须生效。
//
// 这是 P0 修复（RequestStop 不再 Release）引出的回归：旧实现靠 Release 清本地 held
// 让执行侧"看起来被抢占"，去掉 Release 后若停止只写 Redis，单实例场景就彻底停不下来。
// 现在停止意图同时记在进程内 map 里。
func TestRequestStopWorksWithoutRedis(t *testing.T) {
	prev := rediscli.GetCli()
	rediscli.SetCli(nil)
	t.Cleanup(func() { rediscli.SetCli(prev) })

	const taskId uint64 = 97999
	guard := &RunGuard[uint64]{}

	guard.RequestStop(taskId)
	require.True(t, guard.IsStopRequested(taskId), "无 Redis 时停止必须仍然生效（本地标记）")

	guard.ClearStopRequest(taskId)
	require.False(t, guard.IsStopRequested(taskId), "清理后本地标记也必须消失")
}

// TestRequestStopKeepsLockWithoutRedis 停止不释放互斥（与 P0 修复一致）。
func TestRequestStopKeepsLockWithoutRedis(t *testing.T) {
	prev := rediscli.GetCli()
	rediscli.SetCli(nil)
	t.Cleanup(func() { rediscli.SetCli(prev) })

	const taskId uint64 = 97998
	guard := &RunGuard[uint64]{}
	runId, ok := guard.AcquireWithRunId(taskId)
	require.True(t, ok)

	guard.RequestStop(taskId)
	require.True(t, guard.IsCurrentRun(taskId, runId), "停止不得顺手放锁（否则新旧执行并发写同一表）")
	require.True(t, guard.IsStopRequested(taskId))

	guard.ReleaseWithRunId(taskId, runId)
}
