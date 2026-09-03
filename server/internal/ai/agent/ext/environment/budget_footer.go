package environmentext

import (
	"context"
	"fmt"
	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/prompt"
	"mayfly-go/pkg/logx"
)

// 默认提醒阈值比例（剩余低于窗口 10% 时提醒，对齐 tokhub
// PreambleBudgetStatus::DEFAULT_REMINDER_THRESHOLD_RATIO）
const defaultReminderThresholdRatio = 0.1

// ContextBudgetFooterExtension 上下文窗口余量提醒扩展（PreambleFooter 通道）
//
// 对齐 tokhub CoreContextExtension 的内置提醒：preamble 与 history 均就绪后，
// 剩余 token 低于窗口 10% 时在 preamble 尾部注入预算提醒，促使模型收敛输出。
type ContextBudgetFooterExtension struct{}

// NewBudgetFooterExtension 创建预算提醒扩展
func NewBudgetFooterExtension() *ContextBudgetFooterExtension {
	return &ContextBudgetFooterExtension{}
}

var _ contributor.PreambleFooterContributor = (*ContextBudgetFooterExtension)(nil)

func (e *ContextBudgetFooterExtension) Id() string { return "context_budget_footer" }

func (e *ContextBudgetFooterExtension) ContributePreambleFooter(ctx context.Context, fc *contributor.PreambleFooterContext) string {
	if fc == nil || fc.Budget == nil || fc.Budget.ContextWindow <= 0 {
		return ""
	}
	budget := fc.Budget
	// 阈值判断归贡献者（测量与呈现分离）
	if !budget.BelowThreshold(defaultReminderThresholdRatio) {
		return ""
	}
	text, err := prompt.GetPrompt("token_budget/reminder.md", map[string]any{
		"RemainingTokens": budget.RemainingTokens(),
		"UsagePercent":    budget.UsedTokens() * 100 / budget.ContextWindow,
	})
	if err != nil {
		logx.WarnfContext(ctx, "[ext] render token budget reminder failed: %v", err)
		return fmt.Sprintf("[Token Budget] 剩余约 %d tokens，请优先完成核心任务，避免冗余输出。", budget.RemainingTokens())
	}
	return text
}
