package contributor

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ── 测试用 fake 贡献者 ────────────────────────────────────────────

// fakeHistory 历史贡献者 fake（可选携带压缩能力见 fakeCompactor）
type fakeHistory struct {
	id   string
	msgs []adk.Message
	err  error
}

func (f *fakeHistory) Id() string { return f.id }

func (f *fakeHistory) ContributeMessages(ctx context.Context, bc *HistoryBuildContext) ([]adk.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.msgs, nil
}

// fakeCompactor 携带 MidTurnCompactor 可选能力的历史贡献者
type fakeCompactor struct {
	fakeHistory
	calls     int
	compacted []adk.Message
	info      *MidTurnCompactionInfo
}

func (f *fakeCompactor) TryMidTurnCompaction(ctx context.Context, history []adk.Message, params *MidTurnCompactionParams) ([]adk.Message, *MidTurnCompactionInfo) {
	f.calls++
	if f.info == nil {
		return history, nil
	}
	return f.compacted, f.info
}

type fakeContext struct {
	id    string
	frags []PromptFragment
}

func (f *fakeContext) Id() string { return f.id }

func (f *fakeContext) ContributeTurnContext(ctx context.Context, in *TurnInput) ([]PromptFragment, error) {
	return f.frags, nil
}

type fakeFooter struct {
	id   string
	text string
}

func (f *fakeFooter) Id() string { return f.id }

func (f *fakeFooter) ContributePreambleFooter(ctx context.Context, fc *PreambleFooterContext) string {
	return f.text
}

// ── 用例 ─────────────────────────────────────────────────────────

// TestRegistry_CollectHistory_ConcatsAndNormalizes 多贡献者按注册顺序拼接，
// 且配对修复对合并后的全量列表统一执行（跨贡献者的缺失结果在末尾合成）
func TestRegistry_CollectHistory_ConcatsAndNormalizes(t *testing.T) {
	b := NewBuilder()
	b.RegisterHistory(&fakeHistory{id: "h1", msgs: []adk.Message{
		&schema.Message{Role: schema.User, Content: "hi"},
		// assistant 声明了 tool call 但没有结果（跨贡献者配对缺失）
		&schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{
			ID: "call-1", Function: schema.FunctionCall{Name: "shell_exec", Arguments: "{}"},
		}}},
	}})
	b.RegisterHistory(&fakeHistory{id: "h2", msgs: []adk.Message{
		&schema.Message{Role: schema.User, Content: "next"},
	}})
	r := b.Build()

	history := r.CollectHistory(context.Background(), &HistoryBuildContext{SessionKey: "k"})
	if len(history) != 4 {
		t.Fatalf("expected 4 messages (2 + 1 + synthesized result), got %d", len(history))
	}
	if history[0].Content != "hi" || history[2].Content != "next" {
		t.Errorf("unexpected concat order: %v", history)
	}
	last := history[3]
	if last.Role != schema.Tool || last.ToolCallID != "call-1" {
		t.Errorf("synthesized tool result should be appended at tail, got role=%s toolCallId=%s", last.Role, last.ToolCallID)
	}
}

// TestRegistry_CollectHistory_FailOpen 单个贡献者失败不阻断其余贡献
func TestRegistry_CollectHistory_FailOpen(t *testing.T) {
	b := NewBuilder()
	b.RegisterHistory(&fakeHistory{id: "bad", err: errors.New("boom")})
	b.RegisterHistory(&fakeHistory{id: "good", msgs: []adk.Message{
		&schema.Message{Role: schema.User, Content: "ok"},
	}})
	history := b.Build().CollectHistory(context.Background(), &HistoryBuildContext{})
	if len(history) != 1 || history[0].Content != "ok" {
		t.Errorf("fail-open violated, got %v", history)
	}
}

// TestRegistry_TryMidTurnCompaction_ReverseOrderWinner 逆序单赢家：后注册者覆盖内置
func TestRegistry_TryMidTurnCompaction_ReverseOrderWinner(t *testing.T) {
	builtin := &fakeCompactor{
		fakeHistory: fakeHistory{id: "builtin"},
		compacted:   []adk.Message{&schema.Message{Role: schema.User, Content: "builtin-result"}},
		info:        &MidTurnCompactionInfo{OriginalTokens: 100, CompressedTokens: 50},
	}
	plugin := &fakeCompactor{
		fakeHistory: fakeHistory{id: "plugin"},
		compacted:   []adk.Message{&schema.Message{Role: schema.User, Content: "plugin-result"}},
		info:        &MidTurnCompactionInfo{OriginalTokens: 100, CompressedTokens: 40},
	}
	b := NewBuilder()
	b.RegisterHistory(builtin)
	b.RegisterHistory(plugin)

	compacted, info := b.Build().TryMidTurnCompaction(context.Background(), nil, &MidTurnCompactionParams{ContextWindow: 1000})
	if info == nil || info.CompressedTokens != 40 {
		t.Fatalf("plugin (later registered) should win, got info=%v", info)
	}
	if len(compacted) != 1 || compacted[0].Content != "plugin-result" {
		t.Errorf("unexpected compacted history: %v", compacted)
	}
	if builtin.calls != 0 {
		t.Errorf("builtin should not be invoked after plugin wins, calls=%d", builtin.calls)
	}
}

// fakeTool 最小可执行工具（同名覆盖协议测试用）
type fakeTool struct{ name string }

func (f *fakeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: f.name}, nil
}

func (f *fakeTool) InvokableRun(ctx context.Context, argumentsInJSON string, option ...tool.Option) (string, error) {
	return "run:" + f.name, nil
}

// fakeToolProvider 返回固定工具列表的工具贡献者
type fakeToolProvider struct {
	id    string
	tools []tool.BaseTool
	err   error
}

func (f *fakeToolProvider) Id() string { return f.id }

func (f *fakeToolProvider) Tools(ctx context.Context, tc *ToolContributionContext) ([]tool.BaseTool, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.tools, nil
}

// TestRegistry_BuildTools_OverrideByName 同名覆盖协议：
// 后注册者覆盖先注册者但保留先注册位置（内置在前、插件在后可替换内置实现）
func TestRegistry_BuildTools_OverrideByName(t *testing.T) {
	b := NewBuilder()
	b.RegisterTool(&fakeToolProvider{id: "builtin", tools: []tool.BaseTool{
		&fakeTool{name: "shell_exec"},
		&fakeTool{name: "file_read"},
	}})
	b.RegisterTool(&fakeToolProvider{id: "plugin", tools: []tool.BaseTool{
		&fakeTool{name: "shell_exec"}, // 同名覆盖
	}})

	tools := b.Build().BuildTools(context.Background(), nil)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools after override, got %d", len(tools))
	}
	ti0, err := tools[0].Info(context.Background())
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if ti0.Name != "shell_exec" {
		t.Errorf("shell_exec should keep builtin position, got %s", ti0.Name)
	}
}

// TestRegistry_BuildTools_FailOpen 单个贡献者报错不影响其它贡献者
func TestRegistry_BuildTools_FailOpen(t *testing.T) {
	b := NewBuilder()
	b.RegisterTool(&fakeToolProvider{id: "broken", err: errors.New("boom")})
	b.RegisterTool(&fakeToolProvider{id: "ok", tools: []tool.BaseTool{&fakeTool{name: "file_read"}}})

	tools := b.Build().BuildTools(context.Background(), nil)
	if len(tools) != 1 {
		t.Fatalf("broken contributor should be skipped, got %d tools", len(tools))
	}
}

// TestRegistry_BuildTools_NilRegistry nil registry 安全
func TestRegistry_BuildTools_NilRegistry(t *testing.T) {
	var r *Registry
	if tools := r.BuildTools(context.Background(), nil); tools != nil {
		t.Errorf("nil registry should return nil, got %v", tools)
	}
}

// TestRegistry_TryMidTurnCompaction_Decline 无人接手时原样返回
func TestRegistry_TryMidTurnCompaction_Decline(t *testing.T) {
	b := NewBuilder()
	decliner := &fakeCompactor{fakeHistory: fakeHistory{id: "decline"}} // info 为 nil 表示放弃
	b.RegisterHistory(&fakeHistory{id: "plain"})                        // 无压缩能力，不参与竞争
	b.RegisterHistory(decliner)

	orig := []adk.Message{&schema.Message{Role: schema.User, Content: "x"}}
	compacted, info := b.Build().TryMidTurnCompaction(context.Background(), orig, &MidTurnCompactionParams{})
	if info != nil || len(compacted) != 1 || compacted[0].Content != "x" {
		t.Errorf("history should be returned as-is when declined, info=%v", info)
	}
}

// TestRegistry_CollectPreambleFooters 按注册顺序聚合，空文本跳过
func TestRegistry_CollectPreambleFooters(t *testing.T) {
	b := NewBuilder()
	b.RegisterPreambleFooter(&fakeFooter{id: "f1", text: "footer-1"})
	b.RegisterPreambleFooter(&fakeFooter{id: "empty", text: ""})
	b.RegisterPreambleFooter(&fakeFooter{id: "f2", text: "footer-2"})

	footers := b.Build().CollectPreambleFooters(context.Background(), &PreambleFooterContext{})
	if len(footers) != 2 || footers[0] != "footer-1" || footers[1] != "footer-2" {
		t.Errorf("unexpected footers: %v", footers)
	}
}

// TestRegistry_WithFilter 按 Id 裁剪后等价于未安装该贡献者
func TestRegistry_WithFilter(t *testing.T) {
	b := NewBuilder()
	b.RegisterContext(&fakeContext{id: "ctx-a", frags: []PromptFragment{{Source: "a", Content: "A"}}})
	b.RegisterHistory(&fakeHistory{id: "hist-a", msgs: []adk.Message{
		&schema.Message{Role: schema.User, Content: "u"},
	}})
	r := b.Build()

	filtered := r.WithFilter(map[string]struct{}{"hist-a": {}})
	if filtered.HasHistoryContributors() {
		t.Error("filtered registry should have no history contributors")
	}
	if frags := filtered.CollectContext(context.Background(), &TurnInput{}); len(frags) != 1 {
		t.Errorf("unfiltered channel should be kept, got %v", frags)
	}
}

// TestRegistry_NilSafety 未初始化（Default 为 nil）时各 dispatch 不 panic
func TestRegistry_NilSafety(t *testing.T) {
	var r *Registry
	ctx := context.Background()
	if r.CollectHistory(ctx, &HistoryBuildContext{}) != nil {
		t.Error("nil registry should return nil history")
	}
	if r.HasHistoryContributors() {
		t.Error("nil registry should have no history contributors")
	}
	if frags := r.CollectContext(ctx, &TurnInput{}); frags != nil {
		t.Error("nil registry should return nil fragments")
	}
	if footers := r.CollectPreambleFooters(ctx, &PreambleFooterContext{}); footers != nil {
		t.Error("nil registry should return nil footers")
	}
	hist, info := r.TryMidTurnCompaction(ctx, nil, &MidTurnCompactionParams{})
	if hist != nil || info != nil {
		t.Error("nil registry should not compact")
	}
	r.NotifyTurnStart(ctx, &TurnStartInput{})
	r.NotifyTurnEnd(ctx, &TurnEndInput{})
}
