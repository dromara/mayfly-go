package session

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// newFlowManager 创建 JSONL store 会话管理器（不启用自动摘要）
func newFlowManager(t *testing.T) *Manager {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "session_meta_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	store, err := NewStoreJSONL(tempDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return NewManager(store)
}

// TestManager_AppendMsgs_TokenCountPriority 元数据计数流转：
// Count 按消息条数累加；TokenCount 优先取 CompletionTokens（真实用量），
// 无 usage 的消息按内容长度估算
func TestManager_AppendMsgs_TokenCountPriority(t *testing.T) {
	m := newFlowManager(t)
	ctx := context.Background()
	key := "conv:1"

	// assistant 带 usage（CompletionTokens=100）+ user 无 usage（"12345678" 8 rune → est 4）
	err := m.AppendMsgs(ctx, key,
		&Message{Role: schema.User, Content: "12345678"},
		&Message{Role: schema.Assistant, Content: "reply", ResponseMeta: &schema.ResponseMeta{
			Usage: &schema.TokenUsage{PromptTokens: 500, CompletionTokens: 100, TotalTokens: 600},
		}})
	if err != nil {
		t.Fatalf("append msgs: %v", err)
	}

	meta, err := m.GetMeta(ctx, key)
	if err != nil || meta == nil {
		t.Fatalf("get meta: %v", err)
	}
	if meta.Count != 2 {
		t.Errorf("expected count 2, got %d", meta.Count)
	}
	if meta.TokenCount != 104 {
		t.Errorf("expected token count 104 (100 usage + 4 estimated), got %d", meta.TokenCount)
	}
}

// TestManager_GetHistory_PrependsSummary 摘要以 System 消息前置到历史顶部
func TestManager_GetHistory_PrependsSummary(t *testing.T) {
	m := newFlowManager(t)
	ctx := context.Background()
	key := "conv:2"

	for i := 0; i < 3; i++ {
		if err := m.AppendMsgs(ctx, key, &Message{Role: schema.User, Content: "msg"}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	meta, _ := m.GetMeta(ctx, key)
	meta.Summary = "old summary text"
	if err := m.SaveMeta(ctx, meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}

	history, err := m.GetHistory(ctx, key)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 4 {
		t.Fatalf("expected 4 messages (1 summary + 3 raw), got %d", len(history))
	}
	if history[0].Role != schema.System || !strings.Contains(history[0].Content, "old summary text") {
		t.Errorf("summary should be prepended as system message, got %v", history[0])
	}
}

// TestManager_GetHistory_AllSummarized 所有消息均已摘要（Count<=Skip）时
// 只返回摘要消息，不再读取原始历史
func TestManager_GetHistory_AllSummarized(t *testing.T) {
	m := newFlowManager(t)
	ctx := context.Background()
	key := "conv:3"

	if err := m.AppendMsgs(ctx, key, &Message{Role: schema.User, Content: "msg"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	meta, _ := m.GetMeta(ctx, key)
	meta.Summary = "compacted summary"
	meta.Skip = meta.Count // 全部已摘要
	if err := m.SaveMeta(ctx, meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}

	history, err := m.GetHistory(ctx, key)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 || history[0].Role != schema.System || !strings.Contains(history[0].Content, "compacted summary") {
		t.Errorf("expected only summary message, got %v", history)
	}
}

// TestManager_GetHistory_SkipOptimization skip 优化：Store 层只读未摘要消息，
// 摘要前置后返回（keep 之后的近期消息）
func TestManager_GetHistory_SkipOptimization(t *testing.T) {
	m := newFlowManager(t)
	ctx := context.Background()
	key := "conv:4"

	for i := 0; i < 5; i++ {
		if err := m.AppendMsgs(ctx, key, &Message{Role: schema.User, Content: string(rune('a' + i))}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	meta, _ := m.GetMeta(ctx, key)
	meta.Summary = "summary"
	meta.Skip = 3 // 前 3 条已摘要
	if err := m.SaveMeta(ctx, meta); err != nil {
		t.Fatalf("save meta: %v", err)
	}

	history, err := m.GetHistory(ctx, key)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	// 摘要 1 条 + 未摘要 2 条（skip 之后的原始消息）
	if len(history) != 3 {
		t.Fatalf("expected 3 messages (summary + 2 unsummarized), got %d: %v", len(history), history)
	}
	if history[0].Role != schema.System {
		t.Errorf("first message should be summary, got %v", history[0])
	}
	// 未摘要的是最后两条（d、e），验证 skip 偏移正确
	if history[1].Content != "d" || history[2].Content != "e" {
		t.Errorf("unsummarized messages should be the recent ones, got %v", history[1:])
	}
}
