package contributor

import (
	"context"
)

// PreambleBudgetStatus 轮次 token 预算测量
//
// 宿主用与压缩决策同一口径的字符估算统一测量后传入，
// 贡献者据此决定是否注入提醒（自行决定阈值比例与文案）。
type PreambleBudgetStatus struct {
	// ContextWindow 模型上下文窗口（token）
	ContextWindow int
	// PreambleTokens preamble 部分 token 数（静态指令 + 扩展片段）
	PreambleTokens int
	// HistoryTokens 对话历史 token 数（压缩摘要后）
	HistoryTokens int
	// ReservedOutputTokens 为模型输出预留的空间（通常为 ContextWindow / 5）
	ReservedOutputTokens int
}

// UsedTokens 实际已使用 token 数（preamble + history，不含输出预留）
func (s *PreambleBudgetStatus) UsedTokens() int {
	return s.PreambleTokens + s.HistoryTokens
}

// RemainingTokens 剩余可用 token 数（窗口减去已使用与输出预留）
func (s *PreambleBudgetStatus) RemainingTokens() int {
	remaining := s.ContextWindow - s.UsedTokens() - s.ReservedOutputTokens
	if remaining < 0 {
		return 0
	}
	return remaining
}

// BelowThreshold 是否低于指定阈值比例（0 < ratio < 1，贡献者据此决定是否注入提醒）
func (s *PreambleBudgetStatus) BelowThreshold(thresholdRatio float64) bool {
	return float64(s.RemainingTokens()) < float64(s.ContextWindow)*thresholdRatio
}

// PreambleFooterContext preamble 后置注入上下文
type PreambleFooterContext struct {
	// Budget 轮次 token 预算测量（宿主统一口径估算，与压缩决策同源）
	Budget *PreambleBudgetStatus
}

// PreambleFooterContributor preamble 后置注入贡献者（聚合通道）
//
// 返回非空文本时调度方将其追加到 preamble 尾部；返回空串表示本轮无注入。
// 典型用途：上下文窗口余量提醒（如剩余低于阈值时提醒用户开启新会话）。
// fail-open：单个贡献者失败记日志跳过，不阻断主流程。
// 调用点：preamble 片段与合并后 history 均就绪后统一调度一次。
type PreambleFooterContributor interface {
	Contributor
	// ContributePreambleFooter 返回追加到 preamble 尾部的文本（空串表示无注入）
	ContributePreambleFooter(ctx context.Context, fc *PreambleFooterContext) string
}
