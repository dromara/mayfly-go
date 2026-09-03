package agent

import (
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// TestAccumulateUsage_MultiStep 多步工具循环各步 usage 累加（turn 级口径）
func TestAccumulateUsage_MultiStep(t *testing.T) {
	msgs := []adk.Message{
		&schema.Message{Role: schema.User, Content: "q"},
		{Role: schema.Assistant, Content: "step1", ResponseMeta: &schema.ResponseMeta{
			Usage: &schema.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120},
		}},
		{Role: schema.Tool, Content: "result"},
		{Role: schema.Assistant, Content: "step2", ResponseMeta: &schema.ResponseMeta{
			Usage: &schema.TokenUsage{PromptTokens: 150, CompletionTokens: 30, TotalTokens: 180},
		}},
	}

	usage := accumulateUsage(msgs)
	if usage == nil {
		t.Fatal("usage should not be nil")
	}
	if usage.PromptTokens != 250 || usage.CompletionTokens != 50 || usage.TotalTokens != 300 {
		t.Errorf("unexpected accumulated usage: %+v", usage)
	}
}

// TestAccumulateUsage_TotalFallback 模型未回 total 时按 prompt + completion 兜底
func TestAccumulateUsage_TotalFallback(t *testing.T) {
	msgs := []adk.Message{
		{Role: schema.Assistant, Content: "a", ResponseMeta: &schema.ResponseMeta{
			Usage: &schema.TokenUsage{PromptTokens: 10, CompletionTokens: 5},
		}},
	}
	usage := accumulateUsage(msgs)
	if usage == nil || usage.TotalTokens != 15 {
		t.Errorf("expected total=15 (prompt+completion), got %+v", usage)
	}
}

// TestAccumulateUsage_Nil 无 usage 上报时返回 nil（统计保持为 0 的依据）
func TestAccumulateUsage_Nil(t *testing.T) {
	msgs := []adk.Message{
		&schema.Message{Role: schema.User, Content: "no meta"},
		{Role: schema.Assistant, Content: "no usage", ResponseMeta: &schema.ResponseMeta{}},
	}
	if usage := accumulateUsage(msgs); usage != nil {
		t.Errorf("expected nil usage, got %+v", usage)
	}
}
