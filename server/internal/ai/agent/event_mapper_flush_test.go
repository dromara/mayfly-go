package agent

import (
	"context"
	"testing"

	"mayfly-go/internal/ai/protocol"
	"mayfly-go/internal/ai/session"

	"github.com/cloudwego/eino/schema"
)

// 验证 agent 多轮输出的正文按轮次分段为独立 message item（OutputItemDone → flush_active）：
// [正文1] [工具] [正文2] 应产生两个不同 item_id 的 message item，且正文1/2 的
// item_completed 先于工具调用的 item_started 发出
func TestMapToolCallEventFlushesActiveItems(t *testing.T) {
	m := NewEventMapper()
	const turnId = "turn-1"

	// 轮1：推理 + 正文
	for _, evt := range m.MapChunk(turnId, &session.Message{Role: schema.Assistant, ReasoningContent: "思考中", Content: "第一段正文"}) {
		_ = evt
	}
	firstMsgId := m.currentMessageId
	firstReasoningId := m.currentReasoningId
	if firstMsgId == "" || firstReasoningId == "" {
		t.Fatalf("轮1 未创建 streaming item: msgId=%q reasoningId=%q", firstMsgId, firstReasoningId)
	}

	// 轮1 结束：工具调用触发 flush
	events := m.MapToolCallEvent(turnId, &session.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			ID:   "call-1",
			Type: "function",
			Function: schema.FunctionCall{
				Name:      "DbQueryData",
				Arguments: "{}",
			},
		}},
	})

	// 断言 flush 事件先于 toolCall item_started，且内容完整
	completedIds := make(map[string]string) // itemId -> content
	toolCallStarted := false
	for _, evt := range events {
		switch evt.Type {
		case protocol.EventTypeItemCompleted:
			if evt.Item != nil {
				switch evt.Item.Type {
				case protocol.TurnItemTypeReasoning:
					completedIds[evt.Item.Id] = evt.Item.Text
				case protocol.TurnItemTypeMessage:
					completedIds[evt.Item.Id] = evt.Item.Content[0].Text
				}
			}
		case protocol.EventTypeItemStarted:
			if evt.Item != nil && evt.Item.Type == protocol.TurnItemTypeToolCall {
				toolCallStarted = true
			}
		}
	}
	if !toolCallStarted {
		t.Fatal("MapToolCallEvent 未产出 toolCall item_started")
	}
	if got := completedIds[firstReasoningId]; got != "思考中" {
		t.Fatalf("reasoning 终态内容错误: got %q", got)
	}
	if got := completedIds[firstMsgId]; got != "第一段正文" {
		t.Fatalf("message 终态内容错误: got %q", got)
	}

	// flush 后 active 状态应清空
	if m.currentMessageId != "" || m.currentReasoningId != "" || m.accumulatedContent != "" {
		t.Fatal("flush 后 streaming 状态未重置")
	}

	// 轮2：正文应新开 item（新 item_id）
	for _, evt := range m.MapChunk(turnId, &session.Message{Role: schema.Assistant, Content: "第二段正文"}) {
		_ = evt
	}
	if m.currentMessageId == "" {
		t.Fatal("轮2 未创建新 message item")
	}
	if m.currentMessageId == firstMsgId {
		t.Fatal("轮2 复用了轮1 的 message item_id，正文会被合并")
	}

	// 结束：轮2 正文定格
	finalEvents := m.CompleteStreaming(turnId, nil)
	var finalContent string
	for _, evt := range finalEvents {
		if evt.Type == protocol.EventTypeItemCompleted && evt.Item != nil && evt.Item.Type == protocol.TurnItemTypeMessage {
			finalContent = evt.Item.Content[0].Text
		}
	}
	if finalContent != "第二段正文" {
		t.Fatalf("轮2 message 终态内容错误: got %q", finalContent)
	}
}

// 无工具调用的单轮场景：CompleteStreaming 兜底定格，行为与旧版一致
func TestCompleteStreamingWithoutToolCalls(t *testing.T) {
	m := NewEventMapper()
	const turnId = "turn-1"

	for _, evt := range m.MapChunk(turnId, &session.Message{Role: schema.Assistant, Content: "最终回复"}) {
		_ = evt
	}
	events := m.CompleteStreaming(turnId, nil)
	if len(events) != 1 {
		t.Fatalf("期望 1 个 item_completed，实际 %d", len(events))
	}
	if events[0].Item.Content[0].Text != "最终回复" {
		t.Fatalf("终态内容错误: %q", events[0].Item.Content[0].Text)
	}
	// 重复调用应幂等（无 active item 时返回空）
	if again := m.CompleteStreaming(turnId, nil); len(again) != 0 {
		t.Fatalf("重复 CompleteStreaming 应为空，实际 %d 个事件", len(again))
	}
}

// 验证恢复路径 item 状态流转：注册挂起行后 MapToolResultEvent 复用原
// item_id 且仅发 item_completed（不重发 item_started），状态为真实执行结果；
// 用户拒绝的结果不视为 failed，状态为 cancelled
func TestMapToolResultEventResumesTrackedToolCall(t *testing.T) {
	// 桩掉 session 存储：恢复路径参数从 session 恢复，单测环境无存储实例
	session.DefaultSessionStore = &stubSessionStore{}
	m := NewEventMapper()
	const turnId = "turn-1"
	const resumedItemId = "item-pending-1"

	m.TrackResumedToolCall("call-1", resumedItemId)
	events := m.MapToolResultEvent(context.Background(), turnId, &session.Message{
		Role: schema.Tool, ToolCallId: "call-1", ToolName: "DbQueryData", Content: "查询成功",
	})
	if len(events) != 1 || events[0].Type != protocol.EventTypeItemCompleted {
		t.Fatalf("恢复路径应仅发 1 个 item_completed，实际 %d 个事件", len(events))
	}
	item := events[0].Item
	if item == nil || item.Id != resumedItemId {
		t.Fatalf("应复用原 item_id %q，实际 %q", resumedItemId, item.Id)
	}
	if item.Status != protocol.TurnItemStatusSuccess {
		t.Fatalf("恢复后应为真实执行结果 success，实际 %q", item.Status)
	}

	// 用户拒绝：状态 cancelled（拒绝 → Cancelled）
	m.TrackResumedToolCall("call-2", "item-pending-2")
	events = m.MapToolResultEvent(context.Background(), turnId, &session.Message{
		Role: schema.Tool, ToolCallId: "call-2", ToolName: "DbQueryData",
		Content: "[OPERATION_REJECTED] The tool 'DbQueryData' was explicitly rejected by the user.",
	})
	if len(events) != 1 || events[0].Item == nil {
		t.Fatalf("拒绝路径应仅发 1 个 item_completed")
	}
	if events[0].Item.Status != protocol.TurnItemStatusCancelled {
		t.Fatalf("拒绝后状态应为 cancelled，实际 %q", events[0].Item.Status)
	}
}

// stubSessionStore 空实现（仅恢复路径单测用）
type stubSessionStore struct {
	session.Store
}

func (s *stubSessionStore) GetMessage(ctx context.Context, query *session.MessageQuery) ([]*session.Message, error) {
	return nil, nil
}
