package agent

import (
	"context"
	"os"
	"testing"

	"mayfly-go/internal/ai/session"

	"github.com/cloudwego/eino/schema"
)

// TestContextManager_Basic 测试 ContextManager 基本功能
func TestContextManager_Basic(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "session_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 session store
	store, err := session.NewStoreJSONL(tempDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	// 创建 context manager
	sessionManager := session.NewManager(store)
	ctxManager, err := NewContextManager(&ContextManagerConfig{
		SessionManager: sessionManager,
	})
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	// 创建测试会话
	sessionKey := "test:user1"
	ctx := session.WithSessionKey(context.Background(), sessionKey)

	// 测试追加消息
	testMessages := []*session.Message{
		&session.Message{Role: schema.User, Content: "你好"},
		&session.Message{Role: schema.Assistant, Content: "你好！有什么可以帮助你的？"},
		&session.Message{Role: schema.User, Content: "如何查看 Linux 系统负载？"},
	}

	for _, msg := range testMessages {
		if err := ctxManager.AppendMsgs(ctx, msg); err != nil {
			t.Fatalf("append message failed: %v", err)
		}
	}

	// 测试获取历史消息
	messages, err := ctxManager.BuildMessages(ctx)
	if err != nil {
		t.Fatalf("build messages failed: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}

	t.Logf("Successfully built %d messages", len(messages))
}

// TestContextManager_GetSessionMeta 测试获取会话元数据
func TestContextManager_GetSessionMeta(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "session_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 session store 和 context manager
	store, _ := session.NewStoreJSONL(tempDir)
	manager := session.NewManager(store)
	ctxManager, err := NewContextManager(&ContextManagerConfig{
		SessionManager: manager,
	})
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	// 创建测试会话
	sessionKey := "test:user2"
	ctx := session.WithSessionKey(context.Background(), sessionKey)

	// 添加一些消息
	msgs := []*session.Message{
		&session.Message{Role: schema.User, Content: "测试消息"},
	}

	for _, msg := range msgs {
		if err := ctxManager.AppendMsgs(ctx, msg); err != nil {
			t.Fatalf("append message failed: %v", err)
		}
	}

	// 获取元数据
	meta, err := ctxManager.GetSessionMeta(ctx)
	if err != nil {
		t.Fatalf("get session meta failed: %v", err)
	}

	if meta == nil {
		t.Fatal("meta should not be nil")
	}

	if meta.Count != 1 {
		t.Errorf("expected count 1, got %d", meta.Count)
	}

	t.Logf("Session meta - Count: %d, TokenCount: %d", meta.Count, meta.TokenCount)
}

// TestContextManager_ClearHistory 测试清空历史
func TestContextManager_ClearHistory(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "session_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 session store 和 context manager
	store, _ := session.NewStoreJSONL(tempDir)
	manager := session.NewManager(store)
	ctxManager, err := NewContextManager(&ContextManagerConfig{
		SessionManager: manager,
	})
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	// 创建测试会话
	sessionKey := "test:user3"
	ctx := session.WithSessionKey(context.Background(), sessionKey)

	// 添加一些消息
	msgs := []*session.Message{
		&session.Message{Role: schema.User, Content: "消息1"},
		&session.Message{Role: schema.User, Content: "消息2"},
	}

	for _, msg := range msgs {
		if err := ctxManager.AppendMsgs(ctx, msg); err != nil {
			t.Fatalf("append message failed: %v", err)
		}
	}

	// 清空历史
	if err := ctxManager.ClearHistory(ctx); err != nil {
		t.Fatalf("clear history failed: %v", err)
	}

	// 验证历史已清空
	messages, err := ctxManager.BuildMessages(ctx)
	if err != nil {
		t.Fatalf("build messages failed: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(messages))
	}

	t.Log("History cleared successfully")
}
