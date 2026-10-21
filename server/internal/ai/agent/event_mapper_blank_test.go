package agent

import (
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/ai/session"
)

// TestMapChunkDoesNotOpenItemForBlankContent 纯空白正文不得开启 message item。
//
// 实测模型在推理闭合后会先吐一串换行（"\n\n\n"），旧逻辑只要 chunk 非空就开项，界面表现为
// 一条没有任何内容的空气泡，回放与消息统计也会被这类空 item 污染
func TestMapChunkDoesNotOpenItemForBlankContent(t *testing.T) {
	m := NewEventMapper()

	require.Empty(t, m.MapChunk("turn-1", &session.Message{Role: schema.Assistant, Content: "\n\n\n"}),
		"纯换行不应开启 item")
	require.Empty(t, m.MapChunk("turn-1", &session.Message{Role: schema.Assistant, Content: "   "}),
		"纯空格不应开启 item")
	require.Empty(t, m.currentMessageId, "未建项时不应残留 item id")

	// 有效内容出现时才建项（此前累积的空白已暂存，不会丢字）
	events := m.MapChunk("turn-1", &session.Message{Role: schema.Assistant, Content: "你好"})
	require.NotEmpty(t, events, "出现有效正文后才建项")
	require.NotEmpty(t, m.currentMessageId)

	completed := m.completeActiveItems("turn-1")
	require.Len(t, completed, 1, "有效正文正常产出一条完成事件")
}

// TestCompleteActiveItemsSkipsBlankAssistantMessage 定格侧的兜底：即使 item 已被开过，
// 累积正文仍是空白就不产出消息（两层拦截任一层失效，另一层仍能挡住空气泡）。
func TestCompleteActiveItemsSkipsBlankAssistantMessage(t *testing.T) {
	m := NewEventMapper()
	m.currentMessageId = "msg-blank"
	m.accumulatedContent = "  \n\t "

	require.Empty(t, m.completeActiveItems("turn-2"), "纯空白正文不产出 message item")
	require.Empty(t, m.currentMessageId, "定格后重置状态，不能留下悬挂 item id")
}

// TestMapChunkKeepsReasoningWhenContentBlank 推理通道不受正文空白拦截影响（否则思维链会整段丢失）。
func TestMapChunkKeepsReasoningWhenContentBlank(t *testing.T) {
	m := NewEventMapper()

	events := m.MapChunk("turn-3", &session.Message{Role: schema.Assistant, ReasoningContent: "先看看有哪些表"})
	require.NotEmpty(t, events, "推理内容照常下发")
	require.NotEmpty(t, m.currentReasoningId)
	require.Empty(t, m.currentMessageId, "推理不应连带开启正文 item")

	completed := m.completeActiveItems("turn-3")
	require.Len(t, completed, 1, "只完成推理 item，不产出空正文")
}

// TestMapChunkRealAnswerAfterBlankChunks 真实形态：先空白若干串再出正文，最终内容必须完整。
func TestMapChunkRealAnswerAfterBlankChunks(t *testing.T) {
	m := NewEventMapper()

	m.MapChunk("turn-4", &session.Message{Role: schema.Assistant, Content: "\n\n"})
	m.MapChunk("turn-4", &session.Message{Role: schema.Assistant, Content: "\n"})
	m.MapChunk("turn-4", &session.Message{Role: schema.Assistant, Content: "共 83 张表"})

	completed := m.completeActiveItems("turn-4")
	require.Len(t, completed, 1)
	item := completed[0].Item
	require.NotNil(t, item)
	require.NotEmpty(t, item.Content, "开头空白不应吃掉后续正文")
	require.Contains(t, item.Content[0].Text, "共 83 张表")
}
