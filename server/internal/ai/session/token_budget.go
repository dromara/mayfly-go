package session

import (
	"context"

	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/schema"
)

// Token 预算机制（token_budget.rs / compactor.rs 的窗口感知策略）：
//
//   - 预留 20% 输出空间后计算历史可用预算（reservedOutputRatio）
//   - 历史估算 token 超过可用预算 80% 时，后台触发一次增量摘要（LlmComp 前置检查）
//   - 超过可用预算（100%）时，就地紧急裁剪至窗口 50%（midTurnHistoryRatio），
//     摘要消息（如有）始终保留，避免请求直接报 context_length_exceeded
//
// token 估算采用字符口径（与 Manager.estimateTokens 同源），仅用于压缩决策。
const (
	// reservedOutputRatio 输出预留占上下文窗口的比例（1/5，输出预留 20%）
	reservedOutputRatio = 5
	// softBudgetRatio 软阈值：超过可用预算的 (softBudgetRatio-1)/softBudgetRatio（即 80%）时后台触发摘要
	softBudgetRatio = 4
	// midTurnHistoryRatio 紧急裁剪后历史目标占窗口比例
	midTurnHistoryRatio = 2
)

// WithContextWindow 设置模型上下文窗口大小（token），0 表示未配置（跳过窗口检查）
func (m *Manager) WithContextWindow(window int) *Manager {
	m.contextWindow = window
	return m
}

// enforceContextWindow 窗口检查与紧急裁剪（读取时基础预算，无 preamble 口径）
//
// 返回处理后的消息列表；未配置窗口时原样返回。
func (m *Manager) enforceContextWindow(ctx context.Context, key string, messages []*Message) []*Message {
	compacted, _ := m.CompactMidTurn(ctx, key, messages, 0)
	return compacted
}

// CompactMidTurn mid-turn 紧急压缩（try_mid_turn_compaction 通道的内置实现）
//
// 由历史贡献者在 preamble 与合并后 history 均就绪后调用（含 preamble 占用口径）：
//   - 估算总量（preamble + history + 输出预留）超过阈值时触发
//   - 软阈值（80%）：后台触发一次增量摘要（fail-open，不阻塞本轮）
//   - 硬阈值（100%）：就地紧急裁剪至窗口 50%（摘要消息始终保留，
//     避免 context_length_exceeded）
//
// 返回压缩后的消息列表与是否发生压缩。
func (m *Manager) CompactMidTurn(ctx context.Context, key string, messages []*Message, preambleTokens int) ([]*Message, bool) {
	if m.contextWindow <= 0 || len(messages) == 0 {
		return messages, false
	}

	window := m.contextWindow
	usable := window - window/reservedOutputRatio - preambleTokens
	est := EstimateHistoryTokens(messages)
	if est <= usable {
		return messages, false
	}

	// 软阈值：后台触发一次增量摘要（fail-open，不阻塞本轮）
	if m.summaryConfig != nil && m.summaryConfig.Enabled && est > usable/softBudgetRatio*(softBudgetRatio-1) {
		summaryCtx := context.WithoutCancel(ctx)
		gox.Go(func() {
			if err := m.CheckAndSummarize(summaryCtx, key); err != nil {
				logx.ErrorfContext(summaryCtx, "context window soft-threshold summarize error: %v", err)
			}
		})
	}

	// 硬阈值：就地紧急裁剪至窗口 50%（扣除 preamble 占用；保留摘要消息与 tool_call/tool_result 配对）
	budget := (window - preambleTokens) / midTurnHistoryRatio
	if budget <= 0 {
		budget = window / midTurnHistoryRatio
	}
	trimmed := trimToBudget(messages, budget)
	if len(trimmed) < len(messages) {
		logx.WarnfContext(ctx, "[token_budget] history exceeds window: est=%d, usable=%d, preamble=%d, trimmed %d -> %d messages",
			est, usable, preambleTokens, len(messages), len(trimmed))
		return trimmed, true
	}
	return messages, false
}

// trimToBudget 从尾部保留预算内的消息
//
//   - 头部摘要类 system 消息（若有）始终保留
//   - 保留区起点向后修正，避免保留区以孤立的 tool 结果消息开头
func trimToBudget(messages []*Message, budget int) []*Message {
	if len(messages) == 0 {
		return messages
	}

	// 计算头部需完整保留的消息数（连续的 system 消息，即摘要注入）
	header := 0
	for header < len(messages) && messages[header].Role == schema.System {
		header++
	}

	// 从尾部向前累计到预算内为止，得到保留区起点
	tokens := 0
	start := len(messages)
	for start > header {
		t := EstimateHistoryTokens(messages[start-1 : start])
		if tokens+t > budget {
			break
		}
		tokens += t
		start--
	}

	// 修正起点：保留区不能以孤立 tool 结果开头（其配对的 assistant ToolCalls 已被裁剪）
	for start < len(messages) && messages[start].Role == schema.Tool {
		start++
	}
	// 修正后可能超出预算，若保留区全部为 tool 结果则放弃裁剪防御（保持原样由上层兜底）
	if start <= header {
		start = header + 1
		if start > len(messages) {
			return messages
		}
	}

	if start >= len(messages) {
		return messages
	}
	return append(messages[:header:header], messages[start:]...)
}

// EstimateHistoryTokens 估算消息列表的 token 数
//
// 每条消息在内容估算之上叠加固定开销（role/工具协议字段），
// assistant 的 ToolCalls 参数按名称与 JSON 参数长度估算。
func EstimateHistoryTokens(messages []*Message) int {
	const perMessageOverhead = 4
	total := 0
	for _, msg := range messages {
		total += perMessageOverhead + estimateTokens(msg.Content)
		for _, tc := range msg.ToolCalls {
			total += estimateTokens(tc.Function.Name)
			total += estimateTokens(tc.Function.Arguments)
		}
	}
	return total
}
