package session

import (
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// 转换层往返对称性回归测试（ToAgenticMessage ↔ FromAgenticMessage）：
// 事件消费链路（agent events → chat_ws 落库/过滤/日志分类）依赖 FromAgenticMessage
// 恢复准确的 Role 与工具协议字段，任一逆映射缺失都会静默失效（见 Role=Tool 逆映射修复）。

// TestFromAgenticMessage_ToolResultRole 工具结果消息（宿主 role 为 user，
// 对齐 adk 内部桥接 functionToolResultAgenticMessage 形态）逆映射必须恢复 Tool 角色，
// 否则下游 role==schema.Tool 分支（工具结果事件落库/日志分类）全部失效
func TestFromAgenticMessage_ToolResultRole(t *testing.T) {
	msg := &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeUser, // adk 事件流中工具结果宿主 role 为 user
		ContentBlocks: []*schema.ContentBlock{
			{
				Type: schema.ContentBlockTypeFunctionToolResult,
				FunctionToolResult: &schema.FunctionToolResult{
					CallID: "call-1",
					Name:   "shell_exec",
					Content: []*schema.FunctionToolResultContentBlock{
						{Type: schema.FunctionToolResultContentBlockTypeText, Text: &schema.UserInputText{Text: "command output"}},
					},
				},
			},
		},
	}

	m := FromAgenticMessage(msg)
	if m.Role != schema.Tool {
		t.Fatalf("tool result must map back to Role=schema.Tool, got %q", m.Role)
	}
	if m.ToolCallId != "call-1" {
		t.Errorf("ToolCallId should be restored, got %q", m.ToolCallId)
	}
	if m.ToolName != "shell_exec" {
		t.Errorf("ToolName should be restored, got %q", m.ToolName)
	}
	if m.Content != "command output" {
		t.Errorf("result text should aggregate into Content, got %q", m.Content)
	}
}

// TestToolResultRoundtrip tool 消息 → AgenticMessage → Message 往返还原
func TestToolResultRoundtrip(t *testing.T) {
	src := &Message{
		Role:       schema.Tool,
		Content:    "result text",
		ToolCallId: "call-9",
		ToolName:   "file_read",
	}

	m := FromAgenticMessage(src.ToAgenticMessage())
	if m.Role != schema.Tool || m.ToolCallId != "call-9" || m.ToolName != "file_read" || m.Content != "result text" {
		t.Errorf("tool result roundtrip mismatch, got role=%s callId=%s name=%s content=%q",
			m.Role, m.ToolCallId, m.ToolName, m.Content)
	}
}

// TestAssistantToolCallRoundtrip 助手工具调用消息往返还原
func TestAssistantToolCallRoundtrip(t *testing.T) {
	src := &Message{
		Role:    schema.Assistant,
		Content: "let me check",
		ToolCalls: []schema.ToolCall{{
			ID:       "call-2",
			Type:     "function",
			Function: schema.FunctionCall{Name: "shell_exec", Arguments: `{"cmd":"ls"}`},
		}},
	}

	m := FromAgenticMessage(src.ToAgenticMessage())
	if m.Role != schema.Assistant {
		t.Fatalf("role should be preserved, got %q", m.Role)
	}
	if m.Content != "let me check" {
		t.Errorf("content should be preserved, got %q", m.Content)
	}
	if len(m.ToolCalls) != 1 {
		t.Fatalf("tool calls should be restored, got %d", len(m.ToolCalls))
	}
	tc := m.ToolCalls[0]
	if tc.ID != "call-2" || tc.Function.Name != "shell_exec" || tc.Function.Arguments != `{"cmd":"ls"}` {
		t.Errorf("tool call mismatch, got %+v", tc)
	}
}

// TestPlainMessageRoundtrip 普通消息往返 + 多模态图片映射
func TestPlainMessageRoundtrip(t *testing.T) {
	user := &Message{Role: schema.User, Content: "hello", ImageUrls: []string{"http://img/a.jpg"}}
	m := FromAgenticMessage(user.ToAgenticMessage())
	if m.Role != schema.User || m.Content != "hello" {
		t.Errorf("user message roundtrip mismatch, got role=%s content=%q", m.Role, m.Content)
	}

	// ImageUrls 生成 UserInputImage block（多模态输入）
	am := user.ToAgenticMessage()
	var imageBlocks int
	for _, b := range am.ContentBlocks {
		if b.UserInputImage != nil {
			imageBlocks++
			if b.UserInputImage.URL != "http://img/a.jpg" {
				t.Errorf("image url mismatch, got %q", b.UserInputImage.URL)
			}
		}
	}
	if imageBlocks != 1 {
		t.Errorf("expected 1 image block, got %d", imageBlocks)
	}

	assistant := &Message{Role: schema.Assistant, Content: "reply"}
	m = FromAgenticMessage(assistant.ToAgenticMessage())
	if m.Role != schema.Assistant || m.Content != "reply" {
		t.Errorf("assistant message roundtrip mismatch, got role=%s content=%q", m.Role, m.Content)
	}
}

// TestToAgenticMessages_FiltersInternal internal 为内部运行时消息（中断/恢复），
// 非协议合法角色，绝不能进入 LLM 请求
func TestToAgenticMessages_FiltersInternal(t *testing.T) {
	msgs := []*Message{
		{Role: schema.User, Content: "q"},
		{Role: RoleInternal, Content: "resume info"},
		{Role: schema.Assistant, Content: "a"},
	}
	got := ToAgenticMessages(msgs)
	if len(got) != 2 {
		t.Fatalf("internal messages should be filtered, got %d", len(got))
	}
	if got[0].Role != schema.AgenticRoleTypeUser || got[1].Role != schema.AgenticRoleTypeAssistant {
		t.Errorf("roles mismatch: %s, %s", got[0].Role, got[1].Role)
	}
}

// TestFromAgenticMessage_Usage 归一化承载（ResponseMeta.TokenUsage → Message.ResponseMeta）
func TestFromAgenticMessage_Usage(t *testing.T) {
	msg := &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeAssistant,
		ContentBlocks: []*schema.ContentBlock{
			{Type: schema.ContentBlockTypeAssistantGenText, AssistantGenText: &schema.AssistantGenText{Text: "hi"}},
		},
		ResponseMeta: &schema.AgenticResponseMeta{
			TokenUsage: &schema.TokenUsage{CompletionTokens: 5, PromptTokens: 7, TotalTokens: 12},
		},
	}
	m := FromAgenticMessage(msg)
	if m.ResponseMeta == nil || m.ResponseMeta.Usage == nil {
		t.Fatal("usage should be normalized into ResponseMeta")
	}
	if m.ResponseMeta.Usage.TotalTokens != 12 {
		t.Errorf("usage mismatch, got %+v", m.ResponseMeta.Usage)
	}
}

var _ adk.AgenticMessage = (*schema.AgenticMessage)(nil) // 类型边界锚定：adk.AgenticMessage 即 *schema.AgenticMessage
