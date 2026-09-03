package historyext

import (
	"context"
	"os"
	"testing"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/session"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// newTestManager 创建带 JSONL store 的会话管理器（指定窗口）
func newTestManager(t *testing.T, window int) *session.Manager {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "history_ext_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	store, err := session.NewStoreJSONL(tempDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	m := session.NewManager(store)
	if window > 0 {
		m.WithContextWindow(window)
	}
	return m
}

func msg(role schema.RoleType, content string) adk.Message {
	return &schema.Message{Role: role, Content: content}
}

// TestSessionHistoryExtension_ContributeMessages 历史扩展真实加载会话历史，
// 未装配（nil manager）时 fail-open 返回空
func TestSessionHistoryExtension_ContributeMessages(t *testing.T) {
	m := newTestManager(t, 0)
	ctx := context.Background()
	key := "conv:1"

	if err := m.AppendMsgs(ctx, key,
		msg(schema.User, "u1"), msg(schema.Assistant, "a1")); err != nil {
		t.Fatalf("append: %v", err)
	}

	bc := &contributor.HistoryBuildContext{SessionKey: key}
	msgs, err := NewExtension(m).ContributeMessages(ctx, bc)
	if err != nil {
		t.Fatalf("contribute: %v", err)
	}
	if len(msgs) != 2 || msgs[0].Content != "u1" {
		t.Errorf("expected loaded history, got %v", msgs)
	}

	empty, err := NewExtension(nil).ContributeMessages(ctx, bc)
	if err != nil || len(empty) != 0 {
		t.Errorf("nil manager should fail-open with empty history, got %v, err=%v", empty, err)
	}
}

// TestSessionHistoryExtension_MidTurnCompaction 真实压缩触发链路：
// 窗口充足不压缩；含 preamble 口径超限时就地压缩并返回压缩统计
func TestSessionHistoryExtension_MidTurnCompaction(t *testing.T) {
	ctx := context.Background()
	history := []adk.Message{msg(schema.User, "hello")}

	// 未配置窗口：不压缩
	_, info := NewExtension(newTestManager(t, 0)).TryMidTurnCompaction(ctx, history,
		&contributor.MidTurnCompactionParams{})
	if info != nil {
		t.Fatalf("no window should not compact, info=%v", info)
	}

	// 窗口充足（可用预算远大于历史估算）：不压缩
	_, info = NewExtension(newTestManager(t, 0)).TryMidTurnCompaction(ctx, history,
		&contributor.MidTurnCompactionParams{ContextWindow: 100000, PreambleTokens: 10})
	if info != nil {
		t.Fatalf("within budget should not compact, info=%v", info)
	}

	// 窗口极小：3 条消息（各 est≈79，尾部可分裁剪，总 est≈237 > usable 160）
	long := string(make([]rune, 150))
	bigHistory := []adk.Message{msg(schema.User, long), msg(schema.User, long), msg(schema.User, long)}
	compacted, info := NewExtension(newTestManager(t, 200)).TryMidTurnCompaction(ctx, bigHistory,
		&contributor.MidTurnCompactionParams{ContextWindow: 200, PreambleTokens: 0})
	if info == nil {
		t.Fatal("over-budget history should be compacted with info")
	}
	if info.CompressedTokens <= 0 {
		t.Errorf("compressed tokens should be positive, got %d", info.CompressedTokens)
	}
	if len(compacted) >= len(bigHistory) {
		t.Errorf("compacted history should shrink: %d -> %d", len(bigHistory), len(compacted))
	}
}

// TestSessionHistoryExtension_Install 装配行为：nil 跳过注册（fail-open），
// 正常装配后历史通道就绪
func TestSessionHistoryExtension_Install(t *testing.T) {
	b := contributor.NewBuilder()
	Install(b, nil)
	if b.Build().HasHistoryContributors() {
		t.Error("nil sessionManager should skip registration")
	}

	b2 := contributor.NewBuilder()
	Install(b2, newTestManager(t, 0))
	if !b2.Build().HasHistoryContributors() {
		t.Fatal("history contributor should be registered")
	}
}
