// Package historyext 会话历史扩展
//
// 实现 HistoryContributor 通道，作为短期记忆进入对话历史的**唯一通道**
// （对齐 tokhub-ext-memory 的 MemoryHistoryExtension）：
//   - 按 SessionKey 加载短期历史（压缩摘要 + 覆盖点之后的近期消息）
//   - 压缩摘要以 [之前的对话摘要] 前缀消息注入历史顶部（session.Manager 组装）
//   - mid-turn 紧急压缩（含 preamble 占用口径，实现 MidTurnCompactor 可选能力）
//
// ## 无状态设计
//
// 实例仅持有 *session.Manager（启动时经装配注入），不缓存任何轮次/会话数据；
// 当轮历史状态按 SessionKey 每次从共享存储加载，页面刷新、服务重启、
// 跨实例处理均不受影响。
package historyext

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/adk"
)

// SessionHistoryExtension 会话历史扩展（短期记忆 → adk.Message 历史）
type SessionHistoryExtension struct {
	// sessionManager 会话管理器（历史加载 + 摘要 + 窗口压缩的域服务）
	sessionManager *session.Manager
}

// NewExtension 创建会话历史扩展（sessionManager 允许为 nil，未装配时自动跳过注册）
func NewExtension(sessionManager *session.Manager) *SessionHistoryExtension {
	return &SessionHistoryExtension{sessionManager: sessionManager}
}

var _ contributor.HistoryContributor = (*SessionHistoryExtension)(nil)

var _ contributor.MidTurnCompactor = (*SessionHistoryExtension)(nil)

func (e *SessionHistoryExtension) Id() string { return "session_history" }

func (e *SessionHistoryExtension) ContributeMessages(ctx context.Context, bc *contributor.HistoryBuildContext) ([]adk.Message, error) {
	if e.sessionManager == nil || bc.SessionKey == "" {
		return nil, nil
	}
	// 跳过读取时基础窗口检查：宿主在 preamble 就绪后执行含 preamble
	// 占用口径的 mid-turn 压缩（TryMidTurnCompaction），此处不重复触发
	msgs, err := e.sessionManager.GetHistory(ctx, bc.SessionKey, session.WithSkipWindowCheck())
	if err != nil {
		// fail-open：历史加载失败降级为空历史（不阻断对话），口径与旧 assembler 一致
		logx.WarnfContext(ctx, "[ext_history] load history failed for session=%s: %v", bc.SessionKey, err)
		return nil, nil
	}
	return msgs, nil
}

// TryMidTurnCompaction mid-turn 紧急压缩（含 preamble 占用口径）
//
// 总量 = preamble + history + 输出预留（与预算追踪同一字符口径），
// 超阈值时就地压缩合并后的历史；未触发时返回原切片与 nil。
func (e *SessionHistoryExtension) TryMidTurnCompaction(ctx context.Context, history []adk.Message, params *contributor.MidTurnCompactionParams) ([]adk.Message, *contributor.MidTurnCompactionInfo) {
	if e.sessionManager == nil || params == nil || params.ContextWindow <= 0 {
		return history, nil
	}
	compacted, didCompact := e.sessionManager.CompactMidTurn(ctx, params.SessionKey, history, params.PreambleTokens)
	if !didCompact {
		return history, nil
	}
	return compacted, &contributor.MidTurnCompactionInfo{
		OriginalTokens:   params.HistoryTokens,
		CompressedTokens: session.EstimateHistoryTokens(compacted),
	}
}

// Install 注册扩展到 Builder（sessionManager 为 nil 时跳过，历史通道降级为空，
// 与无记忆部署的现状一致）
func Install(b *contributor.Builder, sessionManager *session.Manager) {
	if sessionManager == nil {
		return
	}
	b.RegisterHistory(NewExtension(sessionManager))
}
