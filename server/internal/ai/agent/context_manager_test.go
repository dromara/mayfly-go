package agent

import (
	"context"
	"testing"

	"mayfly-go/internal/ai/session"

	"github.com/cloudwego/eino/schema"
)

// TestContextManagerDegradedPath 锁定 Registry 缺席时的降级路径：直读会话管理器
// 并执行统一配对修复（NormalizeHistory）。生产装配（bootstrap）恒注入 Registry，
// 此分支仅在显式构造且未传 Registry 时触达（Registry 为可选配置），
// 以测试锁定行为，防止主路径与降级路径漂移。
func TestContextManagerDegradedPath(t *testing.T) {
	store, err := session.NewStoreJSONL(t.TempDir())
	if err != nil {
		t.Fatalf("create jsonl store: %v", err)
	}
	cm, err := NewContextManager(&ContextManagerConfig{
		SessionManager: session.NewManager(store),
	})
	if err != nil {
		t.Fatalf("new context manager: %v", err)
	}
	if cm.registry != nil {
		t.Fatal("registry should be nil in degraded construction")
	}

	ctx := session.WithSessionKey(context.Background(), "conv:degraded")
	if err := cm.sessionManager.AppendMsgs(ctx, "conv:degraded",
		&session.Message{Role: schema.User, Content: "hello"},
		&session.Message{Role: schema.Assistant, Content: "hi"},
	); err != nil {
		t.Fatalf("append msgs: %v", err)
	}

	out, err := cm.BuildMessages(ctx, &session.Message{Role: schema.User, Content: "next"})
	if err != nil {
		t.Fatalf("build messages: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 history messages, got %d", len(out))
	}
	if out[0].Content != "hello" || out[1].Content != "hi" {
		t.Fatalf("unexpected history contents: %q, %q", out[0].Content, out[1].Content)
	}
}
