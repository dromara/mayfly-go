package memory

import (
	"context"
	"testing"

	"mayfly-go/internal/ai/llm/llmtest"
	"mayfly-go/internal/ai/session"

	"github.com/cloudwego/eino/schema"
)

// TestLLMExtractorUsesCurrentModelPerCall 记忆提取必须每次按当前生效模型提取
//
// 缺陷类：提取器和进程同生命周期，若装配期固化 ChatModel 实例，模型配置变更后
// 主对话走新配置、长期记忆提取仍打旧地址（且提取失败只是静默跳过，很难察觉）
func TestLLMExtractorUsesCurrentModelPerCall(t *testing.T) {
	firstModel := &llmtest.Model{Reply: `[]`}
	provider := &llmtest.Provider{Model: firstModel}
	extractor := NewLLMExtractor().WithConfig(&LLMExtractorConfig{
		Enabled:       true,
		MinConfidence: 0.7,
		ModelProvider: provider,
	})

	ctx := context.Background()
	msgs := []*session.Message{{Role: schema.User, Content: "记住：我偏好 PostgreSQL"}}
	if _, err := extractor.ExtractFromMessages(ctx, "u1", msgs); err != nil {
		t.Fatalf("extract: %v", err)
	}

	// 模型配置变更：Provider 背后的实例被替换
	secondModel := &llmtest.Model{Reply: `[]`}
	provider.Model = secondModel

	if _, err := extractor.ExtractFromMessages(ctx, "u1", msgs); err != nil {
		t.Fatalf("extract after reload: %v", err)
	}

	if firstModel.Calls() != 1 || secondModel.Calls() != 1 {
		t.Fatalf("model calls: first=%d second=%d, want 1/1（提取器未使用当前生效模型）",
			firstModel.Calls(), secondModel.Calls())
	}
	if provider.ChatModelCalls() != 2 {
		t.Fatalf("provider calls = %d, want 2（每次提取都应重新取模型）", provider.ChatModelCalls())
	}
}

// TestLLMExtractorSkipsWithoutProvider 未注入模型入口时跳过提取（不阻断摘要链路）
func TestLLMExtractorSkipsWithoutProvider(t *testing.T) {
	extractor := NewLLMExtractor().WithConfig(&LLMExtractorConfig{Enabled: true})

	memories, err := extractor.ExtractFromMessages(context.Background(), "u1", []*session.Message{
		{Role: schema.User, Content: "任意内容"},
	})
	if err != nil {
		t.Fatalf("extract should skip instead of failing: %v", err)
	}
	if len(memories) != 0 {
		t.Fatalf("expected no memories, got %d", len(memories))
	}
}
