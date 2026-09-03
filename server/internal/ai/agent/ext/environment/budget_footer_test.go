package environmentext

import (
	"context"
	"testing"

	"mayfly-go/internal/ai/agent/contributor"
)

// budget 构造预算状态（窗口 1000）
func budget(preamble, history, reserved int) *contributor.PreambleBudgetStatus {
	return &contributor.PreambleBudgetStatus{
		ContextWindow:        1000,
		PreambleTokens:       preamble,
		HistoryTokens:        history,
		ReservedOutputTokens: reserved,
	}
}

// TestBudgetFooter_BelowThresholdInjects 剩余低于窗口 10% 时注入提醒，
// 提醒内容携带剩余 token 数（真实渲染 prompt 模板）
func TestBudgetFooter_BelowThresholdInjects(t *testing.T) {
	ext := NewBudgetFooterExtension()
	// used=890 + reserved=100 → remaining=10 < 100（10% × 1000）
	text := ext.ContributePreambleFooter(context.Background(), &contributor.PreambleFooterContext{
		Budget: budget(800, 90, 100),
	})
	if text == "" {
		t.Fatal("below threshold should inject reminder")
	}
}

// TestBudgetFooter_AboveThresholdSilent 剩余充足时不注入
func TestBudgetFooter_AboveThresholdSilent(t *testing.T) {
	ext := NewBudgetFooterExtension()
	// used=400 + reserved=100 → remaining=500 ≥ 100
	text := ext.ContributePreambleFooter(context.Background(), &contributor.PreambleFooterContext{
		Budget: budget(400, 0, 100),
	})
	if text != "" {
		t.Errorf("above threshold should not inject, got %q", text)
	}
}

// TestBudgetFooter_MissingBudget 未携带预算信息（nil / 无窗口）时安全跳过
func TestBudgetFooter_MissingBudget(t *testing.T) {
	ext := NewBudgetFooterExtension()
	if text := ext.ContributePreambleFooter(context.Background(), &contributor.PreambleFooterContext{}); text != "" {
		t.Errorf("nil budget should skip, got %q", text)
	}
	if text := ext.ContributePreambleFooter(context.Background(), nil); text != "" {
		t.Errorf("nil context should skip, got %q", text)
	}
	if text := ext.ContributePreambleFooter(context.Background(), &contributor.PreambleFooterContext{
		Budget: &contributor.PreambleBudgetStatus{},
	}); text != "" {
		t.Errorf("zero window should skip, got %q", text)
	}
}
