package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mayfly-go/internal/ai/session"

	"github.com/stretchr/testify/require"
)

// 无人值守运行不得发起「等人填参数」的中断。
//
// 真机起因（告警 → 触发器 → 诊断运行）：模型提交 `MachineCommandExec{authCertName:""}`，
// 命令是 `free -h && ps aux | head` 这类全只读的复合命令，护栏按入参判定为 Read 并放行；
// 工具体随即发起参数补全中断，而无头运行的 publisher 是 nil——没有前端、没有应答者。
// 于是整轮停在 interrupted，运行被记成 failed，摘要就是那句「缺少机器信息，请完善参数」，
// 7139 tokens 花完、零诊断产出。而它上一步的 ListResources 已经把真实凭证名返回给模型了，
// 只差把「缺哪个参数、可选值有哪些」当成工具错误回传，让模型自己补一次。

// unattendedFakeStore 空会话存储替身：不提供任何历史补全消息，使恢复判定走「未被中断」分支。
// 嵌入接口本身，未预期的方法调用直接暴露为 panic（而不是静默返回零值掩盖接线错误）
type unattendedFakeStore struct {
	session.Store
}

func (unattendedFakeStore) GetMessage(_ context.Context, _ *session.MessageQuery) ([]*session.Message, error) {
	return nil, nil
}

func withEmptySessionStore(t *testing.T) {
	t.Helper()
	restore := session.DefaultSessionStore
	session.DefaultSessionStore = unattendedFakeStore{}
	t.Cleanup(func() { session.DefaultSessionStore = restore })
}

func TestUnattendedCtxRoundTrip(t *testing.T) {
	// 缺省必须是 false：对话路径仍按「有人应答」弹卡片，不能因为开了这个接缝就静默改掉等待语义
	require.False(t, IsUnattended(context.Background()))
	require.False(t, IsUnattended(nil), "nil ctx 不得 panic，也不得被判成无人值守")

	require.True(t, IsUnattended(WithUnattended(context.Background())))
	// 标记不可反向清除：一次注入即整条运行有效（嵌套 WithValue 只增不减）
	require.True(t, IsUnattended(WithUnattended(WithUnattended(context.Background()))))
}

func TestParamCompletionUnattendedReturnsRetryableToolError(t *testing.T) {
	withEmptySessionStore(t)

	missing := []CompletionParamInfo{{Param: "authCertName", Name: "授权凭证名称"}}
	opts := []CompletionOption{{Label: "tVUcRFKwOC", Value: `{"authCertName":"tVUcRFKwOC"}`}}
	args := &probeCompletionArgs{Command: "free -h"}

	err := InterruptOrResumeParamCompletion(
		WithUnattended(context.Background()),
		"MachineCommandExec", args, "缺少机器信息，请完善参数", "machine", missing, opts,
	)
	require.Error(t, err, "无人值守下缺参必须报错，而不是发起一个永不应答的中断")

	// 必须是「工具错误 + 可重试」：SafeToolMiddleware 会把 RecoverRetry 的工具错误转成
	// [tool error] 前缀字符串回传给模型；换成 RecoverNone 就是硬失败，整轮照样白烧
	var toolErr *ToolError
	require.True(t, errors.As(err, &toolErr), "返回的必须是 ToolError，否则不会被当作工具结果回传给模型")
	require.Equal(t, RecoverRetry, toolErr.Strategy, "必须可重试：目标是让模型补全参数再调一次，而不是终止运行")

	msg := toolErr.Msg
	require.Contains(t, msg, "authCertName", "必须点名缺哪个参数，否则模型无从补起")
	require.Contains(t, msg, "授权凭证名称")
	require.Contains(t, msg, "tVUcRFKwOC", "必须把已算好的候选值回传：这正是补全卡片会展示的内容")
	require.Contains(t, msg, "无人值守")

	// 拿它去喂模型时必须带工具错误前缀，模型才知道这是「一次失败的调用」而非最终答复
	require.True(t, IsToolErrorMsg(GetToolErrorMsg(err)))
}

// TestParamCompletionUnattendedBranchPrecedesInterrupt 分支必须落在「发起中断」之前。
//
// 顺序颠倒等于白写：中断已经把 eino 的状态机推进到 interrupted，之后的判定不会再被执行
func TestParamCompletionUnattendedBranchPrecedesInterrupt(t *testing.T) {
	fn := interruptOrResumeSource(t)
	at := indexOf(t, fn, "IsUnattended(ctx)", "无人值守判定")
	before := indexOf(t, fn, "return InterruptParamCompletion(", "发起中断")
	require.Less(t, at, before, "无人值守判定必须在发起补全中断之前，否则永远不会被走到")

	// 判定只能作用于「未恢复」分支：已经有人补过参数时绝不能把恢复结果也当成错误回传
	resumeGuard := indexOf(t, fn, "if !isResume {", "未恢复分支起点")
	require.Less(t, resumeGuard, at, "无人值守判定必须在 !isResume 分支内")
}

// TestHeadlessRunnerMarksUnattended 标记必须由无头运行入口单点注入。
//
// 判定的唯一真源是「这次运行有没有连着的前端」；在别处（按 profile/按工具名）推断都会漂移，
// 而漂移的后果是把对话路径的补全卡片也一起关掉——那才是真正的功能性破坏
func TestHeadlessRunnerMarksUnattended(t *testing.T) {
	source := readAgentRunnerSource(t)
	require.Contains(t, source, "tools.WithUnattended(runCtx)",
		"RunHeadless 必须注入无人值守标记：该路径 publisher 传 nil，没有任何应答者")
	// 注入点必须在 Agent.Run 之前（runCtx 一旦被用于运行再改就晚了）
	runAt := indexOf(t, source, "tools.WithUnattended(runCtx)", "标记注入")
	agentAt := indexOf(t, source, "agent.WithTurnId(turnId)", "运行参数装配")
	require.Less(t, runAt, agentAt, "标记须在运行开始前注入")
}

// readSourceFile 读取指定源文件（接线守卫用）
func readSourceFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoErrorf(t, err, "读取 %s 失败（改路径/改文件名必须同步本守卫）", path)
	return string(raw)
}

// interruptOrResumeSource 取 InterruptOrResumeParamCompletion 的函数体源码
func interruptOrResumeSource(t *testing.T) string {
	t.Helper()
	source := readSourceFile(t, filepath.Join(".", "interrupt_param_completion.go"))
	start := indexOf(t, source, "func InterruptOrResumeParamCompletion(", "InterruptOrResumeParamCompletion")
	rest := source[start:]
	end := strings.Index(rest, "\n}")
	require.Greater(t, end, 0, "函数体闭合括号未顶格，本守卫的截取方式已失配")
	return rest[:end]
}

// readAgentRunnerSource 无头运行适配器源码（本包不依赖 application，只按相对路径读文件）
func readAgentRunnerSource(t *testing.T) string {
	t.Helper()
	return readSourceFile(t, filepath.Join("..", "application", "agent_runner.go"))
}

// indexOf 定位必需片段：找不到直接红，而不是返回 -1 让后续位置断言恒真
func indexOf(t *testing.T, source, needle, what string) int {
	t.Helper()
	idx := strings.Index(source, needle)
	require.GreaterOrEqualf(t, idx, 0, "源码里找不到「%s」必需片段：%s（改名/删除必须同步本守卫）", what, needle)
	return idx
}
