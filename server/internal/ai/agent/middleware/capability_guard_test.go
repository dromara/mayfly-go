package middleware

import (
	"context"
	"strings"
	"testing"

	"mayfly-go/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
)

// grantCtx 带一次性许可的运行上下文
func grantCtx(names ...string) context.Context {
	return tools.WithRunGrant(context.Background(), &tools.RunGrant{AllowedToolNames: names})
}

// --- 能力维度（既有行为不变） ---

func TestCapabilityGuard_BlockedDecision(t *testing.T) {
	m := NewCapabilityGuardMiddleware(tools.ReadOnlyPolicy(), false)

	// 只读工具放行
	reason, cap := m.Decision(context.Background(), "DbQueryData", "")
	require.Equal(t, denyNone, reason)
	require.Equal(t, tools.CapabilityRead, cap)

	// 高危工具拦截
	reason, cap = m.Decision(context.Background(), "ExecSql", "")
	require.Equal(t, denyByCapability, reason)
	require.Equal(t, tools.CapabilityDangerous, cap)

	// 未知工具（默认 write）在只读档下拦截
	reason, _ = m.Decision(context.Background(), "some_unknown_tool", "")
	require.Equal(t, denyByCapability, reason)
}

// 无策略且无许可 = 全放行（默认对话档行为不变）
func TestCapabilityGuard_UnrestrictedNoGrantAllowsAll(t *testing.T) {
	m := NewCapabilityGuardMiddleware(nil, false)
	require.Equal(t, denyNone, mustReason(m, context.Background(), "ExecSql"))
	require.Equal(t, denyNone, mustReason(m, context.Background(), "any_unknown_tool"))
}

// --- 一次性许可维度 ---

// 许可是天花板：下发了许可就只许清单内的，即使该档策略本来放行其它工具
func TestCapabilityGuard_GrantNarrowsUnrestrictedProfile(t *testing.T) {
	m := NewCapabilityGuardMiddleware(nil, false)

	reason := mustReason(m, grantCtx("ExecSql"), "ExecSql")
	require.Equal(t, denyNone, reason)
	// 未列入许可的只读工具同样被拒：许可表达的是「本次运行只做这一件事」
	require.Equal(t, denyByGrant, mustReason(m, grantCtx("ExecSql"), "DbQueryData"))
}

// 许可不能越过能力策略：两个维度取交集，只读档不因许可获得写能力
func TestCapabilityGuard_GrantCannotBypassCapabilityPolicy(t *testing.T) {
	m := NewCapabilityGuardMiddleware(tools.ReadOnlyPolicy(), false)

	// 许可里写了高危工具，但只读策略不放行 → 仍按能力拦截
	require.Equal(t, denyByCapability, mustReason(m, grantCtx("ExecSql"), "ExecSql"))
	// 只读工具未列入许可 → 按许可拦截
	require.Equal(t, denyByGrant, mustReason(m, grantCtx("ExecSql"), "DbQueryData"))
}

// 声明「必须由许可驱动」的档（受控执行）：没带许可就什么都不许做（fail-closed）
func TestCapabilityGuard_RequireGrantFailsClosed(t *testing.T) {
	m := NewCapabilityGuardMiddleware(nil, true)

	require.Equal(t, denyMissingGrant, mustReason(m, context.Background(), "ExecSql"))
	require.Equal(t, denyMissingGrant, mustReason(m, context.Background(), "DbQueryData"))
	// 带上许可后仅该工具可用
	require.Equal(t, denyNone, mustReason(m, grantCtx("ExecSql"), "ExecSql"))
	require.Equal(t, denyByGrant, mustReason(m, grantCtx("ExecSql"), "MachineCommandExec"))
}

// --- 调用链拦截 ---

func TestCapabilityGuard_WrapInvokableBlocksDangerous(t *testing.T) {
	m := NewCapabilityGuardMiddleware(tools.ReadOnlyPolicy(), false)

	endpointCalled := false
	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		endpointCalled = true
		return "real-result", nil
	}

	wrapped, err := m.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{Name: "ExecSql", CallID: "c1"})
	require.NoError(t, err)

	// 高危工具：拦截，端点不被调用，回传引导改走提案的错误消息（非 error 中断）
	out, err := wrapped(context.Background(), "{}")
	require.NoError(t, err)
	require.False(t, endpointCalled, "被拦截工具的端点不应执行")
	require.True(t, tools.IsToolErrorMsg(out), "拦截应回传工具错误消息，引导模型改走变更提案")
	require.Contains(t, out, "ExecSql")
}

func TestCapabilityGuard_WrapInvokableAllowsRead(t *testing.T) {
	m := NewCapabilityGuardMiddleware(tools.ReadOnlyPolicy(), false)

	endpointCalled := false
	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		endpointCalled = true
		return "query-result", nil
	}

	wrapped, err := m.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{Name: "DbQueryData", CallID: "c2"})
	require.NoError(t, err)

	out, err := wrapped(context.Background(), "{}")
	require.NoError(t, err)
	require.True(t, endpointCalled, "只读工具端点应正常执行")
	require.Equal(t, "query-result", out)
}

// 许可外的调用回传给模型的提示要说清「本次运行只允许已批准的那一个动作」，
// 与「只读档请改走提案」是两类不同的引导
func TestCapabilityGuard_GrantDenialMessage(t *testing.T) {
	m := NewCapabilityGuardMiddleware(nil, true)

	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		return "should-not-run", nil
	}
	wrapped, err := m.WrapInvokableToolCall(grantCtx("ExecSql"), endpoint, &adk.ToolContext{Name: "MachineCommandExec", CallID: "c4"})
	require.NoError(t, err)

	out, err := wrapped(grantCtx("ExecSql"), "{}")
	require.NoError(t, err)
	require.True(t, tools.IsToolErrorMsg(out))
	require.Contains(t, out, "已批准")
}

func TestCapabilityGuard_WrapStreamableBlocksDangerous(t *testing.T) {
	m := NewCapabilityGuardMiddleware(tools.ReadOnlyPolicy(), false)

	endpointCalled := false
	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		endpointCalled = true
		return nil, nil
	}

	wrapped, err := m.WrapStreamableToolCall(context.Background(), endpoint, &adk.ToolContext{Name: "MachineCommandExec", CallID: "c3"})
	require.NoError(t, err)

	sr, err := wrapped(context.Background(), "{}")
	require.NoError(t, err)
	require.False(t, endpointCalled, "被拦截工具的流式端点不应执行")
	require.NotNil(t, sr)
	// 读取单块回传内容
	chunk, readErr := sr.Recv()
	require.NoError(t, readErr)
	require.True(t, tools.IsToolErrorMsg(chunk))
}

// mustReason 判定辅助（不关心能力级别时用）
func mustReason(m *CapabilityGuardMiddleware, ctx context.Context, name string) guardDeny {
	reason, _ := m.Decision(ctx, name, "")
	return reason
}

// TestCapabilityGuard_ContentCapabilityBlockedByArgs 运行期护栏必须按「已下探出的实际能力」判定，
// 而不是因为工具是内容级就整工具恒放行（H1 修复前 policy.Allows(toolName) 对内容级工具无条件 true，
// 只读档下 MachineCommandExec("rm") 会漏过护栏、只靠工具体审批兜）。
func TestCapabilityGuard_ContentCapabilityBlockedByArgs(t *testing.T) {
	const probe = "probe_guard_content_tool"
	tools.DeclareContentCapability(probe, func(args string) tools.Capability {
		if strings.Contains(args, "danger") {
			return tools.CapabilityDangerous
		}
		return tools.CapabilityRead
	})
	m := NewCapabilityGuardMiddleware(tools.ReadOnlyPolicy(), false)

	// 只读入参 → Read → 放行
	reason, cap := m.Decision(context.Background(), probe, `{"command":"top"}`)
	require.Equal(t, denyNone, reason)
	require.Equal(t, tools.CapabilityRead, cap)

	// 危险入参 → Dangerous → 只读档按 cap 拦下
	reason, cap = m.Decision(context.Background(), probe, `{"command":"danger-rm"}`)
	require.Equal(t, denyByCapability, reason)
	require.Equal(t, tools.CapabilityDangerous, cap)
}
