package session

import (
	"context"
	"testing"

	"mayfly-go/internal/ai/llm/llmtest"

	"github.com/cloudwego/eino/schema"
)

// TestLLMSummarizerUsesCurrentModelPerCall 摘要器必须每次按当前生效模型生成摘要
//
// 缺陷类：摘要器与进程同生命周期，若装配期固化 ChatModel 实例，管理员改完模型
// 地址 / Key 后主对话已用新配置，而会话摘要仍打旧地址（编译、门禁、日志全绿）
func TestLLMSummarizerUsesCurrentModelPerCall(t *testing.T) {
	firstModel := &llmtest.Model{Reply: "摘要 A"}
	provider := &llmtest.Provider{Model: firstModel}
	summarizer := NewLLMSummarizer().WithModelProvider(provider)

	msgs := []*Message{{Role: schema.User, Content: "你好"}}
	got, err := summarizer.GenerateSummary(context.Background(), msgs)
	if err != nil {
		t.Fatalf("generate summary: %v", err)
	}
	if got != "摘要 A" {
		t.Fatalf("summary = %q, want %q", got, "摘要 A")
	}

	// 模型配置变更：Provider 背后的实例被替换（真实链路由 agent.ResetModelRuntime 完成）
	secondModel := &llmtest.Model{Reply: "摘要 B"}
	provider.Model = secondModel

	got, err = summarizer.GenerateSummary(context.Background(), msgs)
	if err != nil {
		t.Fatalf("generate summary after reload: %v", err)
	}
	if got != "摘要 B" {
		t.Fatalf("summary after reload = %q, want %q（摘要器未使用当前生效模型）", got, "摘要 B")
	}
	if firstModel.Calls() != 1 || secondModel.Calls() != 1 {
		t.Fatalf("model calls: first=%d second=%d, want 1/1", firstModel.Calls(), secondModel.Calls())
	}
}

// TestLLMSummarizerDegradesWithoutModel 模型不可用时降级为规则摘要（不阻断摘要链路）
func TestLLMSummarizerDegradesWithoutModel(t *testing.T) {
	provider := &llmtest.Provider{Err: context.Canceled}
	summarizer := NewLLMSummarizer().WithModelProvider(provider)

	got, err := summarizer.GenerateSummary(context.Background(), []*Message{
		{Role: schema.User, Content: "帮我看看数据库负载"},
	})
	if err != nil {
		t.Fatalf("summary should degrade instead of failing: %v", err)
	}
	if got == "" {
		t.Fatal("degraded summary should not be empty")
	}
	if provider.ChatModelCalls() != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.ChatModelCalls())
	}
}
