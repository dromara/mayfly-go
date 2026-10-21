package middleware

import (
	"context"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// CapabilityGuardMiddleware 工具调用运行期护栏（防御纵深）
//
// 它把两个维度的收窄放在同一条调用链上裁决：
//   - 能力策略（档级、静态）：与装配期过滤（filterToolsByPolicy）互补——过滤负责从清单移除
//     不放行的工具，护栏负责拦截模型仍尝试调用的越权工具（幻觉出的工具名、或经 tool_search
//     延迟加载绕过装配裁剪的那条路）。
//   - 一次性运行许可（运行级、动态）：本次运行只许调用许可清单里的工具，清单外一律拒绝。
//     受控执行档（审批后按锁定参数执行）靠它把「只能做这一件事」变成机制而不是提示词。
//
// 与内容级治理（CheckSqlsWithoutTicket/CheckMachineCmd 的 SQL/命令审批）正交：内容级治理管
// 「同一工具内哪些参数危险」，本护栏管「这一档/这一次运行能否调用此工具」。
type CapabilityGuardMiddleware struct {
	*adk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]

	policy *tools.ToolPolicy
	// requireGrant 该档必须由一次性许可驱动：许可缺失时拒绝一切工具调用（fail-closed）
	requireGrant bool
}

// NewCapabilityGuardMiddleware 创建护栏中间件。
// policy 可为 nil（表示该档不按能力收窄）；requireGrant=true 时该档没有许可就什么都不许做。
func NewCapabilityGuardMiddleware(policy *tools.ToolPolicy, requireGrant bool) *CapabilityGuardMiddleware {
	return &CapabilityGuardMiddleware{
		TypedBaseChatModelAgentMiddleware: &adk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]{},
		policy:                            policy,
		requireGrant:                      requireGrant,
	}
}

// guardDeny 拦截原因
type guardDeny int

const (
	denyNone guardDeny = iota
	denyByCapability
	denyByGrant
	denyMissingGrant
)

// Decision 护栏判定（纯函数，便于单测覆盖三种维度组合与降级分支）：
// 返回拦截原因与工具能力级别（供日志与提示文案）
func (m *CapabilityGuardMiddleware) Decision(ctx context.Context, toolName, args string) (guardDeny, tools.Capability) {
	// 内容级：MachineCommandExec 这类「一通道多能力」的工具按入参下探（只读命令算 Read），
	// 其余工具回退静态能力表；判定器与工具体审批白名单同源，避免两处口径分叉
	cap := tools.CapabilityForArgs(toolName, args)
	grant := tools.RunGrantFrom(ctx)

	// 许可维度：下发了就只许清单内的（即使能力策略本来放行该工具）
	if grant != nil {
		if !grant.Allows(toolName) {
			return denyByGrant, cap
		}
	} else if m.requireGrant {
		// 该档声明「必须由许可驱动」，却没有许可 → 什么都不许做（不能因为「没带约束」就放开全量工具）
		return denyMissingGrant, cap
	}

	// 能力维度：按已下探出的实际能力判定（内容级工具的危险调用据此拦下），而非按工具名恒放行
	if !m.policy.AllowsCapability(cap) {
		return denyByCapability, cap
	}
	return denyNone, cap
}

func (m *CapabilityGuardMiddleware) WrapInvokableToolCall(
	ctx context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tc *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		if reason, cap := m.Decision(ctx, tc.Name, args); reason != denyNone {
			logx.WarnfContext(ctx, "[capability-guard] blocked invokable tool call: tool=%s, reason=%s, capability=%s", tc.Name, reason, cap)
			return guardDeniedMsg(tc.Name, reason, cap), nil
		}
		return endpoint(ctx, args, opts...)
	}, nil
}

func (m *CapabilityGuardMiddleware) WrapStreamableToolCall(
	ctx context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tc *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		if reason, cap := m.Decision(ctx, tc.Name, args); reason != denyNone {
			logx.WarnfContext(ctx, "[capability-guard] blocked streamable tool call: tool=%s, reason=%s, capability=%s", tc.Name, reason, cap)
			return singleChunkReader(guardDeniedMsg(tc.Name, reason, cap)), nil
		}
		return endpoint(ctx, args, opts...)
	}, nil
}

func (r guardDeny) String() string {
	switch r {
	case denyByCapability:
		return "capability"
	case denyByGrant:
		return "grant-not-allowed"
	case denyMissingGrant:
		return "grant-missing"
	default:
		return "none"
	}
}

// guardDeniedMsg 按拦截原因给出对应提示（回传给模型，引导其改走只读诊断/提案路径）。
// 措辞区分两类拒绝：能力越权应改走提案；许可外说明「本次运行只允许已批准的那一个动作」
func guardDeniedMsg(toolName string, reason guardDeny, cap tools.Capability) string {
	return tools.GetToolErrorMsg(&capabilityDeniedError{toolName: toolName, capability: cap, reason: reason})
}

type capabilityDeniedError struct {
	toolName   string
	capability tools.Capability
	reason     guardDeny
}

func (e *capabilityDeniedError) Error() string {
	switch e.reason {
	case denyByGrant, denyMissingGrant:
		return "工具 " + e.toolName + " 不在本次运行的一次性许可范围内（本次运行只允许执行已批准的那一个动作）。" +
			"禁止改用其他工具扩大变更范围；请严格按许可内工具与锁定参数执行，并如实回报结果。"
	default:
		return "工具 " + e.toolName + "（能力级别：" + e.capability.String() +
			"）不在当前 Agent 的工具策略允许范围内。当前为只读诊断档，任何变更类操作" +
			"必须通过变更提案（propose_remediation）登记并经人工审批后由执行档受控执行。" +
			"请改用只读工具完成诊断，或将该操作登记为变更提案。"
	}
}
