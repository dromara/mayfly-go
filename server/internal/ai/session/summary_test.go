package session

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

// TestLLMSummarizer_Basic 测试 LLM 摘要器基本功能
func TestLLMSummarizer_Basic(t *testing.T) {
	summarizer := NewLLMSummarizer()

	// 测试空消息
	summary, err := summarizer.GenerateSummary(context.Background(), nil)
	if err != nil {
		t.Errorf("GenerateSummary with nil messages failed: %v", err)
	}
	if summary != "" {
		t.Errorf("Expected empty summary for nil messages, got: %s", summary)
	}

	// 测试空数组
	summary, err = summarizer.GenerateSummary(context.Background(), []*Message{})
	if err != nil {
		t.Errorf("GenerateSummary with empty messages failed: %v", err)
	}
	if summary != "" {
		t.Errorf("Expected empty summary for empty messages, got: %s", summary)
	}
}

// TestLLMSummarizer_MultiTurn 测试多轮对话摘要
func TestLLMSummarizer_MultiTurn(t *testing.T) {
	summarizer := NewLLMSummarizer()

	messages := []*Message{
		&Message{
			Role:    schema.User,
			Content: "你好，我想查询一下订单状态",
		},
		&Message{
			Role:    schema.Assistant,
			Content: "好的，请提供订单号",
		},
		&Message{
			Role:    schema.User,
			Content: "订单号是 123456789",
		},
		&Message{
			Role:    schema.Assistant,
			Content: "您的订单已发货，预计明天到达",
		},
	}

	// 由于没有配置 ChatModel，会降级到简单摘要
	summary, err := summarizer.GenerateSummary(context.Background(), messages)
	if err != nil {
		t.Errorf("GenerateSummary failed: %v", err)
	}
	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	t.Logf("Generated summary: %s", summary)
}

// TestLLMSummarizer_WithMaxMessages 测试最大消息数限制
func TestLLMSummarizer_WithMaxMessages(t *testing.T) {
	// 创建大量消息
	var messages []*Message
	for i := 0; i < 100; i++ {
		messages = append(messages, &Message{
			Role:    schema.User,
			Content: fmt.Sprintf("Message %d", i),
		})
	}

	summarizer := NewLLMSummarizer()
	customSummarizer := NewLLMSummarizer().WithMaxMessages(20)

	// 默认应该使用 50 条消息
	_, _ = summarizer.GenerateSummary(context.Background(), messages)

	// 自定义应该使用 20 条消息
	_, _ = customSummarizer.GenerateSummary(context.Background(), messages)
}

// TestLLMSummarizer_ToolCalls 测试包含工具调用的对话
func TestLLMSummarizer_ToolCalls(t *testing.T) {
	summarizer := NewLLMSummarizer()

	messages := []*Message{
		&Message{
			Role:    schema.User,
			Content: "帮我查询天气",
		},
		&Message{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{
					Function: schema.FunctionCall{
						Name: "get_weather",
					},
				},
			},
		},
		&Message{
			Role:     schema.Tool,
			ToolName: "get_weather",
			Content:  "晴天，25度",
		},
		&Message{
			Role:    schema.Assistant,
			Content: "今天天气晴朗，温度25度",
		},
	}

	summary, err := summarizer.GenerateSummary(context.Background(), messages)
	if err != nil {
		t.Errorf("GenerateSummary failed: %v", err)
	}
	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	t.Logf("Generated summary with tool calls: %s", summary)
}

// TestSummaryConfig_Default 测试默认摘要配置
func TestSummaryConfig_Default(t *testing.T) {
	config := DefaultSummaryConfig()

	if config.MessageThreshold != DefaultMessageThreshold {
		t.Errorf("Expected MessageThreshold %d, got %d", DefaultMessageThreshold, config.MessageThreshold)
	}
	if config.TokenThreshold != DefaultTokenThreshold {
		t.Errorf("Expected TokenThreshold %d, got %d", DefaultTokenThreshold, config.TokenThreshold)
	}
	if config.KeepRecentCount != DefaultSummaryKeepCount {
		t.Errorf("Expected KeepRecentCount %d, got %d", DefaultSummaryKeepCount, config.KeepRecentCount)
	}
	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if config.Summarizer == nil {
		t.Error("Expected Summarizer to be set")
	}
}

// TestManager_CheckAndSummarize 测试自动摘要检查
func TestManager_CheckAndSummarize(t *testing.T) {
	// 创建临时目录用于测试
	tmpDir := t.TempDir()
	store, err := NewStoreJSONL(tmpDir)
	if err != nil {
		t.Fatalf("NewStoreJSONL failed: %v", err)
	}
	manager := NewManager(store)

	// 配置摘要（禁用，避免实际调用 LLM）
	config := &SummaryConfig{
		MessageThreshold: 3,
		TokenThreshold:   1000,
		KeepRecentCount:  2,
		Enabled:          false, // 禁用以避免实际摘要
	}
	manager.WithSummaryConfig(config)

	ctx := context.Background()
	sessionKey := "test:user1"

	// 添加消息
	msgs := []*Message{
		&Message{Role: schema.User, Content: "消息1"},
		&Message{Role: schema.Assistant, Content: "回复1"},
		&Message{Role: schema.User, Content: "消息2"},
	}

	for _, msg := range msgs {
		if err := manager.AppendMsgs(ctx, sessionKey, msg); err != nil {
			t.Fatalf("AppendMsgs failed: %v", err)
		}
	}

	// 检查摘要（应该不执行，因为 Enabled=false）
	err = manager.CheckAndSummarize(ctx, sessionKey)
	if err != nil {
		t.Errorf("CheckAndSummarize failed: %v", err)
	}

	// 验证元数据
	meta, err := manager.GetMeta(ctx, sessionKey)
	if err != nil {
		t.Fatalf("GetMeta failed: %v", err)
	}
	if meta.Count != 3 {
		t.Errorf("Expected count 3, got %d", meta.Count)
	}
}

// TestSessionManager_AutoSummarize 测试 session.Manager 的自动摘要功能
func TestSessionManager_AutoSummarize(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "session_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 session store 和 manager
	store, err := NewStoreJSONL(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	manager := NewManager(store)

	// 配置摘要（设置较低的阈值以便快速触发）
	summaryConfig := &SummaryConfig{
		MessageThreshold: 3,
		TokenThreshold:   1000,
		KeepRecentCount:  2,
		Enabled:          true,
		Summarizer:       NewLLMSummarizer(), // 使用简单摘要器
	}
	manager.WithSummaryConfig(summaryConfig)

	ctx := context.Background()
	sessionKey := "test:auto_summarize"

	// 追加消息以达到阈值（MessageThreshold=3, KeepRecentCount=2, minTriggerCount=5）
	msgs := []*Message{
		&Message{Role: schema.User, Content: "消息1"},
		&Message{Role: schema.Assistant, Content: "回复1"},
		&Message{Role: schema.User, Content: "消息2"},
		&Message{Role: schema.Assistant, Content: "回复2"},
		&Message{Role: schema.User, Content: "消息3"},
		&Message{Role: schema.Assistant, Content: "回复3"},
	}

	for _, msg := range msgs {
		if err := manager.AppendMsgs(ctx, sessionKey, msg); err != nil {
			t.Fatalf("append message failed: %v", err)
		}
	}

	// 等待异步摘要完成
	time.Sleep(200 * time.Millisecond)

	// 验证元数据
	meta, err := manager.GetMeta(ctx, sessionKey)
	if err != nil {
		t.Fatalf("get meta failed: %v", err)
	}

	t.Logf("After auto-summarize - Count: %d, Skip: %d, Summary length: %d",
		meta.Count, meta.Skip, len(meta.Summary))

	// 验证触发了摘要
	if meta.Skip <= 0 {
		t.Error("expected Skip > 0 after auto-summarize")
	}

	if meta.Summary == "" {
		t.Error("expected non-empty summary after auto-summarize")
	}

	// 验证 GetHistory 利用了 Skip 优化并组装了摘要
	history, err := manager.GetHistory(ctx, sessionKey)
	if err != nil {
		t.Fatalf("get history failed: %v", err)
	}

	t.Logf("GetHistory returned %d messages (1 summary + %d history with Skip=%d optimization)",
		len(history), len(history)-1, meta.Skip)

	// 验证返回了摘要消息（第 1 条应该是系统消息）
	if len(history) == 0 {
		t.Fatal("expected at least 1 message (summary)")
	}

	if history[0].Role != schema.System {
		t.Errorf("expected first message to be System (summary), got role: %v", history[0].Role)
	}

	// 由于 Skip 优化，历史消息应该少于原始消息数
	historyCount := len(history) - 1 // 减去摘要消息
	if historyCount >= 4 {
		t.Errorf("Skip optimization not working: got %d history messages (expected < 4)", historyCount)
	}

	t.Log("✅ Auto-summarize, Skip optimization, and summary assembly working correctly!")
}

// TestManager_GetHistory_WithLimit 测试 GetHistory 的 limit 优化
func TestManager_GetHistory_WithLimit(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "session_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 session store 和 manager
	store, err := NewStoreJSONL(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	manager := NewManager(store)

	ctx := context.Background()
	sessionKey := "test:limit_optimization"

	// 追加 10 条消息
	for i := 1; i <= 10; i++ {
		msgs := []*Message{
			&Message{Role: schema.User, Content: fmt.Sprintf("消息%d", i)},
		}
		if err := manager.AppendMsgs(ctx, sessionKey, msgs...); err != nil {
			t.Fatalf("append message %d failed: %v", i, err)
		}
	}

	// 测试 1: 不指定 limit，应该返回所有消息
	history, err := manager.GetHistory(ctx, sessionKey)
	if err != nil {
		t.Fatalf("get history failed: %v", err)
	}
	t.Logf("Test 1 - No limit: got %d messages", len(history))
	if len(history) != 10 {
		t.Errorf("expected 10 messages, got %d", len(history))
	}

	// 测试 2: 指定 limit=3，应该只返回最近 3 条（验证 Store 层不会加载全部 10 条）
	history, err = manager.GetHistory(ctx, sessionKey, WithGetMessageLimit(3))
	if err != nil {
		t.Fatalf("get history with limit failed: %v", err)
	}
	t.Logf("Test 2 - Limit=3: got %d messages", len(history))
	if len(history) != 3 {
		t.Errorf("expected 3 messages with limit=3, got %d", len(history))
	}

	// 验证返回的是最近的 3 条
	if len(history) == 3 {
		lastMsg := history[2]
		if lastMsg.Content != "消息10" {
			t.Errorf("expected last message to be '消息10', got '%s'", lastMsg.Content)
		}
		firstMsg := history[0]
		if firstMsg.Content != "消息8" {
			t.Errorf("expected first message to be '消息8', got '%s'", firstMsg.Content)
		}
	}

	t.Log("✅ GetHistory limit optimization working correctly!")
}
