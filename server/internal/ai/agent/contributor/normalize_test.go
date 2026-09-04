package contributor

import (
	"testing"

	"mayfly-go/internal/ai/session"

	"github.com/cloudwego/eino/schema"
)

// assistantWithCall 构造带单个 tool_call 的 assistant 消息
func assistantWithCall(id, name string) *session.Message {
	return &session.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{
			{ID: id, Function: schema.FunctionCall{Name: name, Arguments: "{}"}},
		},
	}
}

// toolResult 构造 tool 结果消息
func toolResult(id, content string) *session.Message {
	return &session.Message{Role: schema.Tool, Content: content, ToolCallId: id}
}

func userMsg(content string) *session.Message {
	return &session.Message{Role: schema.User, Content: content}
}

// TestEnsureCallOutputsPresent_SynthesizesMissingResult 缺失结果被合成为 aborted 占位符
func TestEnsureCallOutputsPresent_SynthesizesMissingResult(t *testing.T) {
	msgs := []*session.Message{
		userMsg("hello"),
		assistantWithCall("call-1", "shell_exec"),
	}
	result := NormalizeHistory(msgs)
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	last := result[2]
	if last.Role != schema.Tool || last.ToolCallId != "call-1" {
		t.Fatalf("expected synthesized tool result for call-1, got role=%s toolCallID=%s", last.Role, last.ToolCallId)
	}
}

// TestNormalizeHistory_SkipsExistingResult 已有结果不合成
func TestNormalizeHistory_SkipsExistingResult(t *testing.T) {
	msgs := []*session.Message{
		userMsg("hello"),
		assistantWithCall("call-1", "shell_exec"),
		toolResult("call-1", "output"),
	}
	result := NormalizeHistory(msgs)
	if len(result) != 3 {
		t.Fatalf("expected 3 messages (unchanged), got %d", len(result))
	}
}

// TestNormalizeHistory_RemovesOrphans 孤儿 tool 结果被移除
func TestNormalizeHistory_RemovesOrphans(t *testing.T) {
	msgs := []*session.Message{
		userMsg("hello"),
		assistantWithCall("call-1", "shell_exec"),
		toolResult("call-1", "output"),
		toolResult("orphan-id", "stale result"),
	}
	result := NormalizeHistory(msgs)
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	for _, m := range result {
		if m.ToolCallId == "orphan-id" {
			t.Fatal("orphan tool result should be removed")
		}
	}
}

// TestNormalizeHistory_Combined 组合通道：孤儿移除 + 缺失合成同时生效
func TestNormalizeHistory_Combined(t *testing.T) {
	msgs := []*session.Message{
		userMsg("hello"),
		assistantWithCall("call-1", "shell_exec"),
		assistantWithCall("call-2", "file_read"),
		toolResult("call-1", "output"),
		toolResult("orphan", "stale"),
	}
	result := NormalizeHistory(msgs)

	hasOrphan := false
	hasSynthesized := false
	for _, m := range result {
		if m.ToolCallId == "orphan" {
			hasOrphan = true
		}
		if m.ToolCallId == "call-2" {
			hasSynthesized = true
		}
	}
	if hasOrphan {
		t.Fatal("orphan should be removed")
	}
	if !hasSynthesized {
		t.Fatal("call-2 should have synthesized result")
	}
}

// TestNormalizeHistory_Idempotent 幂等：重复 normalize 不再变化
func TestNormalizeHistory_Idempotent(t *testing.T) {
	msgs := []*session.Message{
		userMsg("hello"),
		assistantWithCall("call-1", "shell_exec"),
		toolResult("orphan", "stale"),
	}
	once := NormalizeHistory(msgs)
	twice := NormalizeHistory(once)
	if len(once) != len(twice) {
		t.Fatalf("normalize should be idempotent: %d vs %d", len(once), len(twice))
	}
}
