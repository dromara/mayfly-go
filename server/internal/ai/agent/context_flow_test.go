package agent

import (
	"context"
	"os"
	"strings"
	"testing"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/session"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// ── 测试用 fake 贡献者（registry_test.go 在 contributor 包内，此处单独定义） ──

type flowHistory struct {
	id   string
	msgs []adk.Message
}

func (f *flowHistory) Id() string { return f.id }

func (f *flowHistory) ContributeMessages(ctx context.Context, bc *contributor.HistoryBuildContext) ([]adk.Message, error) {
	return f.msgs, nil
}

type flowCompactor struct {
	flowHistory
	calls     int
	compacted []adk.Message
	info      *contributor.MidTurnCompactionInfo
}

func (f *flowCompactor) TryMidTurnCompaction(ctx context.Context, history []adk.Message, params *contributor.MidTurnCompactionParams) ([]adk.Message, *contributor.MidTurnCompactionInfo) {
	f.calls++
	if f.info == nil {
		return history, nil
	}
	return f.compacted, f.info
}

type flowContext struct {
	id    string
	frags []contributor.PromptFragment
}

func (f *flowContext) Id() string { return f.id }

func (f *flowContext) ContributeTurnContext(ctx context.Context, in *contributor.TurnInput) ([]contributor.PromptFragment, error) {
	return f.frags, nil
}

type flowFooter struct {
	id   string
	text string
}

func (f *flowFooter) Id() string { return f.id }

func (f *flowFooter) ContributePreambleFooter(ctx context.Context, fc *contributor.PreambleFooterContext) string {
	return f.text
}

// newCtxManagerWithRegistry 创建持有指定注册中心的上下文管理器
// （注册中心显式注入实例，不再依赖全局单例替换）
func newCtxManagerWithRegistry(t *testing.T, sessionManager *session.Manager, b *contributor.Builder, contextWindow int) *ContextManager {
	t.Helper()
	cm, err := NewContextManager(&ContextManagerConfig{
		SessionManager: sessionManager,
		ContextWindow:  contextWindow,
		Registry:       b.Build(),
	})
	if err != nil {
		t.Fatalf("create context manager: %v", err)
	}
	return cm
}

// newFlowManager 创建带 JSONL store 的会话管理器（压缩/摘要配置不启用）
func newFlowManager(t *testing.T) *session.Manager {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "session_flow_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	store, err := session.NewStoreJSONL(tempDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return session.NewManager(store)
}

// ── BuildMessages 组装顺序 ──────────────────────────────────────

// TestBuildMessages_AssemblyOrder 全流程组装顺序对齐 tokhub start 路径：
// 历史（贡献者通道）→ preamble 片段 → mid-turn 压缩 → footer 注入 → system 前置
func TestBuildMessages_AssemblyOrder(t *testing.T) {
	sessionManager := newFlowManager(t)

	b := contributor.NewBuilder()
	b.RegisterHistory(&flowHistory{id: "hist", msgs: []adk.Message{
		&schema.Message{Role: schema.User, Content: "hist-user"},
		&schema.Message{Role: schema.Assistant, Content: "hist-assistant"},
	}})
	b.RegisterContext(&flowContext{id: "ctx", frags: []contributor.PromptFragment{
		{Source: "env", Content: "ENV-FRAGMENT"},
	}})
	b.RegisterPreambleFooter(&flowFooter{id: "footer", text: "FOOTER-TEXT"})
	ctxManager := newCtxManagerWithRegistry(t, sessionManager, b, 0)

	ctx := session.WithSessionKey(context.Background(), "conv:1")
	messages, err := ctxManager.BuildMessages(ctx, &schema.Message{Role: schema.User, Content: "new question"})
	if err != nil {
		t.Fatalf("build messages: %v", err)
	}

	// [system(preamble+footer), hist-user, hist-assistant]
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d: %v", len(messages), messages)
	}
	first := messages[0]
	if first.Role != schema.System {
		t.Fatalf("first message should be system preamble, got %s", first.Role)
	}
	if !strings.Contains(first.Content, "ENV-FRAGMENT") || !strings.Contains(first.Content, "FOOTER-TEXT") {
		t.Errorf("preamble should contain fragment and footer, got: %s", first.Content)
	}
	if messages[1].Content != "hist-user" || messages[2].Content != "hist-assistant" {
		t.Errorf("history messages should follow preamble, got: %v", messages[1:])
	}
}

// ── mid-turn 压缩触发 ────────────────────────────────────────────

// TestBuildMessages_MidTurnCompaction_Applied 压缩贡献者接手时历史被替换，
// 且 preamble 仍前置在压缩后历史之前
func TestBuildMessages_MidTurnCompaction_Applied(t *testing.T) {
	sessionManager := newFlowManager(t)

	compactor := &flowCompactor{
		flowHistory: flowHistory{id: "hist+compactor"},
		compacted:   []adk.Message{&schema.Message{Role: schema.User, Content: "compacted"}},
		info:        &contributor.MidTurnCompactionInfo{OriginalTokens: 900, CompressedTokens: 100},
	}
	b := contributor.NewBuilder()
	b.RegisterHistory(compactor)
	b.RegisterContext(&flowContext{id: "ctx", frags: []contributor.PromptFragment{
		{Source: "env", Content: "ENV"},
	}})
	ctxManager := newCtxManagerWithRegistry(t, sessionManager, b, 128000)

	ctx := session.WithSessionKey(context.Background(), "conv:2")
	messages, err := ctxManager.BuildMessages(ctx)
	if err != nil {
		t.Fatalf("build messages: %v", err)
	}

	if compactor.calls != 1 {
		t.Fatalf("compactor should be invoked once, calls=%d", compactor.calls)
	}
	// [system(preamble), compacted]
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages after compaction, got %d: %v", len(messages), messages)
	}
	if messages[0].Role != schema.System || messages[1].Content != "compacted" {
		t.Errorf("unexpected assembly after compaction: %v", messages)
	}
}

// TestBuildMessages_MidTurnCompaction_Declined 压缩贡献者放弃时历史原样保留
func TestBuildMessages_MidTurnCompaction_Declined(t *testing.T) {
	sessionManager := newFlowManager(t)

	compactor := &flowCompactor{flowHistory: flowHistory{id: "hist+decline"}} // 无 info = 放弃
	b := contributor.NewBuilder()
	b.RegisterHistory(compactor)
	b.RegisterContext(&flowContext{id: "ctx", frags: []contributor.PromptFragment{
		{Source: "env", Content: "ENV"},
	}})
	ctxManager := newCtxManagerWithRegistry(t, sessionManager, b, 128000)

	ctx := session.WithSessionKey(context.Background(), "conv:3")
	compactor.msgs = []adk.Message{
		&schema.Message{Role: schema.User, Content: "keep-1"},
		&schema.Message{Role: schema.Assistant, Content: "keep-2"},
	}
	messages, err := ctxManager.BuildMessages(ctx)
	if err != nil {
		t.Fatalf("build messages: %v", err)
	}

	if compactor.calls != 1 {
		t.Fatalf("compactor should be consulted once, calls=%d", compactor.calls)
	}
	if len(messages) != 3 || messages[1].Content != "keep-1" || messages[2].Content != "keep-2" {
		t.Errorf("history should be unchanged when compaction declined: %v", messages)
	}
}

// TestBuildMessages_FallbackWithoutHistoryContributor 未装配历史扩展时
// 降级直读会话管理器（保生产兜底与既有行为兼容）
func TestBuildMessages_FallbackWithoutHistoryContributor(t *testing.T) {
	sessionManager := newFlowManager(t)
	// Registry 为 nil（未装配历史扩展）→ 走 sessionManager.GetHistory 降级
	ctxManager, err := NewContextManager(&ContextManagerConfig{SessionManager: sessionManager})
	if err != nil {
		t.Fatalf("create context manager: %v", err)
	}

	ctx := session.WithSessionKey(context.Background(), "conv:4")
	if err := ctxManager.AppendMsgs(ctx,
		&schema.Message{Role: schema.User, Content: "u1"},
		&schema.Message{Role: schema.Assistant, Content: "a1"},
	); err != nil {
		t.Fatalf("append msgs: %v", err)
	}

	messages, err := ctxManager.BuildMessages(ctx)
	if err != nil {
		t.Fatalf("build messages: %v", err)
	}
	if len(messages) != 2 || messages[0].Content != "u1" {
		t.Errorf("fallback should load raw history from session manager, got %v", messages)
	}
}
