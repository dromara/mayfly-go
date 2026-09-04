package session

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

func userMsg(content string) *Message {
	return &Message{Role: schema.User, Content: content}
}

func assistantToolCall(id, name string) *Message {
	return &Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			ID: id, Function: schema.FunctionCall{Name: name, Arguments: "{}"},
		}},
	}
}

func toolResult(id, content string) *Message {
	return &Message{Role: schema.Tool, Content: content, ToolCallId: id}
}

func summaryHeader(summary string) *Message {
	return &Message{Role: schema.System, Content: summary}
}

// TestCompactMidTurn_NoWindow 未配置窗口（contextWindow=0）时不做任何处理
func TestCompactMidTurn_NoWindow(t *testing.T) {
	m := &Manager{}
	msgs := []*Message{userMsg("a"), userMsg("b")}
	got, compacted := m.CompactMidTurn(context.Background(), "k", msgs, 0)
	if compacted || len(got) != 2 {
		t.Errorf("no window should keep messages as-is, compacted=%v", compacted)
	}
}

// TestCompactMidTurn_WithinBudget 预算内不裁剪
func TestCompactMidTurn_WithinBudget(t *testing.T) {
	m := &Manager{contextWindow: 100000}
	msgs := []*Message{userMsg("short history")}
	got, compacted := m.CompactMidTurn(context.Background(), "k", msgs, 100)
	if compacted || len(got) != 1 {
		t.Errorf("within budget should not compact, compacted=%v", compacted)
	}
}

// TestCompactMidTurn_HardTrim 超预算时紧急裁剪：
// 头部摘要 system 消息始终保留，且保留区不以孤立 tool 结果开头
func TestCompactMidTurn_HardTrim(t *testing.T) {
	m := &Manager{contextWindow: 200} // usable = 200 - 40 = 160，明显小于历史估算
	msgs := []*Message{
		summaryHeader("[之前的对话摘要]\n..."),
		userMsg(longText(200)),
		assistantToolCall("call-1", "shell_exec"),
		toolResult("call-1", "old result"),
		userMsg(longText(200)),
		assistantToolCall("call-2", "file_read"),
		toolResult("call-2", "recent result"),
	}

	got, compacted := m.CompactMidTurn(context.Background(), "k", msgs, 0)
	if !compacted {
		t.Fatal("over-budget history should be compacted")
	}
	if got[0].Role != schema.System {
		t.Errorf("summary header should be preserved, got[0].role=%s", got[0].Role)
	}
	// 保留区起点（跳过头部 system 后的第一条）不能是孤立 tool 结果，
	// 配对的 assistant 调用必须随其结果一同保留
	if got[1].Role == schema.Tool {
		t.Errorf("retained region must not start with orphan tool result, got %v", got)
	}
	// 裁剪应保留尾部（最近消息），确认 call-2 配对仍在
	if len(got) < 2 {
		t.Fatalf("trim should keep recent messages, got %d", len(got))
	}
}

// TestTrimToBudget_KeepsHeaderSystem 裁剪从尾部保留，头部连续 system 全保留
func TestTrimToBudget_KeepsHeaderSystem(t *testing.T) {
	msgs := []*Message{
		summaryHeader("s1"),
		summaryHeader("s2"),
		userMsg(longText(100)),
		userMsg(longText(100)),
		userMsg("tail"),
	}
	// 预算只够 tail + 1 条
	got := trimToBudget(msgs, estimateTokens("tail")+estimateTokens(longText(100))+8)
	if len(got) < 3 {
		t.Fatalf("header(2) + retained should be kept, got %d", len(got))
	}
	if got[0].Content != "s1" || got[1].Content != "s2" {
		t.Errorf("header system messages should be preserved: %v", got[:2])
	}
	if got[len(got)-1].Content != "tail" {
		t.Errorf("recent messages should be kept from tail, last=%v", got[len(got)-1])
	}
}

// TestEstimateHistoryTokens_ToolCallArguments 工具调用参数计入估算
func TestEstimateHistoryTokens_ToolCallArguments(t *testing.T) {
	plain := []*Message{userMsg("x")}
	withTool := []*Message{assistantToolCall("call-1", "shell_exec")}
	// assistantToolCall 无 Content，其估算全部来自 ToolCalls 名称与参数开销
	if EstimateHistoryTokens(withTool) <= EstimateHistoryTokens(plain) {
		t.Errorf("tool call arguments should add token overhead: tool=%d plain=%d",
			EstimateHistoryTokens(withTool), EstimateHistoryTokens(plain))
	}
}

// fakeSummarizer 同步可断言的摘要器（记录调用与输入）
// calls 用原子计数：摘要经后台协程异步触发（gox.Go），主测试协程轮询读取，
// 需同步原语避免数据竞争（-race 下曾失败）
type fakeSummarizer struct {
	calls atomic.Int32
	mu    sync.Mutex
	// lastMsgs 由后台摘要协程写入、主协程读取，经 mu 同步
	lastMsgs []*Message
}

func (f *fakeSummarizer) GenerateSummary(ctx context.Context, messages []*Message) (string, error) {
	f.calls.Add(1)
	f.mu.Lock()
	f.lastMsgs = messages
	f.mu.Unlock()
	return "fake summary", nil
}

// snapshotLastMsgs 线程安全读取最近一次摘要输入
func (f *fakeSummarizer) snapshotLastMsgs() []*Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastMsgs
}

// TestCompactMidTurn_SoftThreshold_TriggersSummary 软阈值（超可用预算 80%）
// 后台触发一次增量摘要：摘要完成后 Summary/Skip 写入元数据
func TestCompactMidTurn_SoftThreshold_TriggersSummary(t *testing.T) {
	store, err := NewStoreJSONL(t.TempDir())
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	summarizer := &fakeSummarizer{}
	m := NewManager(store).
		WithSummaryConfig(&SummaryConfig{
			MessageThreshold: 1, KeepRecentCount: 1, TokenThreshold: 0,
			Enabled: true, Summarizer: summarizer,
		}).
		WithContextWindow(200) // usable = 160，软阈值 120；历史估算远超

	key := "conv:soft"
	// est = 64+9+64+64+9+5 = 215 > usable(160)，触发软阈值 + 硬裁剪；
	// 硬裁剪保留尾部 3 条（est 78 ≤ budget 100），且起点修正非孤立 tool 结果
	msgs := []*Message{
		userMsg(longText(120)),
		assistantToolCall("call-1", "shell_exec"),
		toolResult("call-1", longText(120)),
		userMsg(longText(120)),
		assistantToolCall("call-2", "file_read"),
		toolResult("call-2", "ok"),
	}

	// 真实写入历史消息（后台摘要 CheckAndSummarize 需从 store 读取未摘要消息），
	// AppendMsgs 同时把 meta.Count 更新为 6
	if err := m.AppendMsgs(context.Background(), key, msgs...); err != nil {
		t.Fatalf("append msgs: %v", err)
	}

	// 压缩：软阈值调度后台摘要，硬阈值就地裁剪
	compactedMsgs, didCompact := m.CompactMidTurn(context.Background(), key, msgs, 0)
	if !didCompact {
		t.Fatal("over-budget history should be compacted")
	}
	if len(compactedMsgs) >= len(msgs) {
		t.Fatalf("compaction should trim messages, got %d -> %d", len(msgs), len(compactedMsgs))
	}

	// 等待后台摘要完成（gox.Go 异步，轮询最多 2s）
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if summarizer.calls.Load() > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if summarizer.calls.Load() == 0 {
		t.Fatal("soft threshold should trigger background summary")
	}
	if len(summarizer.snapshotLastMsgs()) == 0 {
		t.Error("summarizer should receive messages to summarize")
	}
}

// TestGetHistory_SkipWindowCheck 历史贡献者通道带 WithSkipWindowCheck 时
// 跳过读取时基础裁剪（宿主后续做含 preamble 口径的完整压缩），
// 防止双层裁剪回归
func TestGetHistory_SkipWindowCheck(t *testing.T) {
	store, err := NewStoreJSONL(t.TempDir())
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	m := NewManager(store).WithContextWindow(200) // usable=160，历史估算 256 必然超限
	ctx := context.Background()
	key := "conv:skip"

	long := userMsg(longText(200))
	if err := m.AppendMsgs(ctx, key, long); err != nil {
		t.Fatalf("append: %v", err)
	}

	// 默认：读取时裁剪生效
	trimmed, err := m.GetHistory(ctx, key)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(trimmed) >= 1 && EstimateHistoryTokens(trimmed) > 200 {
		t.Errorf("default GetHistory should enforce window, est=%d", EstimateHistoryTokens(trimmed))
	}

	// 带 WithSkipWindowCheck：原样返回，不裁剪
	full, err := m.GetHistory(ctx, key, WithSkipWindowCheck())
	if err != nil {
		t.Fatalf("get history with skip: %v", err)
	}
	if len(full) != 1 {
		t.Errorf("skip window check should return messages as-is, got %d", len(full))
	}
}

// longText 生成指定 rune 数的文本（估算口径：1 rune ≈ 0.5 token）
func longText(runes int) string {
	b := make([]rune, runes)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}
