package agent

import (
	"context"
	"strings"

	"mayfly-go/internal/ai/protocol"
	"mayfly-go/internal/ai/session"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"

	"github.com/cloudwego/eino/schema"
)

// EventMapper 将 eino AgentEvent 映射为 protocol.EventMsg
//
// 非并发安全：回调均由 Agent.Run 同步调用（runTurn 单 goroutine），每个 turn
// 经 NewEventMapper 创建独立实例，无跨 turn 状态残留问题
type EventMapper struct {
	// 跟踪当前 streaming item 的 ID
	currentMessageId   string
	currentReasoningId string
	// 累积流式内容
	accumulatedContent   string
	accumulatedReasoning string
	// toolCallId -> itemId 映射，确保 item_started 和 item_completed 使用相同的 itemId
	toolCallItemIds map[string]string
	// toolCallId -> arguments 缓存：终态 item_completed 事件必须携带完整参 数，
	// 否则落库覆盖后历史重建的 tool_calls 无 arguments，后续 LLM 请求将被网关拒绝
	toolCallArgs map[string]string
	// toolCallId -> 挂起时已落库的 itemId：恢复路径工具完成时复用原 item_id，
	// ItemCompleted 复用原 item_id、更新同一行至真实终态
	resumedToolCallItemIds map[string]string
}

// NewEventMapper 创建事件映射器
func NewEventMapper() *EventMapper {
	return &EventMapper{
		toolCallItemIds:        make(map[string]string),
		toolCallArgs:           make(map[string]string),
		resumedToolCallItemIds: make(map[string]string),
	}
}

// TrackResumedToolCall 注册恢复路径的挂起工具调用（api 层启动恢复前，
// 从持久化层加载该 turn 的 interrupted tool_call item 后调用）
func (m *EventMapper) TrackResumedToolCall(toolCallId, itemId string) {
	if toolCallId == "" || itemId == "" {
		return
	}
	m.resumedToolCallItemIds[toolCallId] = itemId
}

// MapChunk 将流式 chunk 映射为 EventMsg
// 返回一个或多个事件（首次 chunk 会产生 item_started + item_updated）
func (m *EventMapper) MapChunk(turnId string, chunk *session.Message) []*protocol.EventMsg {
	var events []*protocol.EventMsg

	// 处理推理内容（reasoning）
	if chunk.ReasoningContent != "" {
		m.accumulatedReasoning += chunk.ReasoningContent
		if m.currentReasoningId == "" {
			// 首次收到推理内容，发送 item_started。
			// item_id 用 SortableUUID（UUIDv7，时间有序）在触发时刻生成：
			// item 持久化顺序与触发顺序解耦，历史查询按 item_id 排序即还原真实时序（new_item_id）
			m.currentReasoningId = stringx.SortableUUID()
			item := protocol.NewReasoningTurnItem(m.currentReasoningId, "")
			events = append(events, protocol.NewItemStartedEvent(turnId, item))
		}
		// 发送增量更新
		events = append(events, protocol.NewItemUpdatedEvent(turnId, m.currentReasoningId, &protocol.ItemDelta{
			Text: chunk.ReasoningContent,
		}))
	}

	// 处理正文内容
	if chunk.Content != "" {
		m.accumulatedContent += chunk.Content
		// 累积仍是空白时不开 item：实测模型在推理闭合后会先吐出一串换行（“\n\n\n”），
		// 旧逻辑只要 chunk 非空就开一个 message item，界面表现为一条没有任何内容的空气泡，
		// 回放与消息统计也会被这类空 item 污染。内容真出现时才建项（累积量已暂存，不会丢字）
		if strings.TrimSpace(m.accumulatedContent) == "" {
			return events
		}
		if m.currentMessageId == "" {
			// 首次收到正文内容，发送 item_started（item_id 触发时刻生成，时间有序）
			m.currentMessageId = stringx.SortableUUID()
			item := protocol.NewMessageTurnItem(m.currentMessageId, string(schema.Assistant), []protocol.ContentSegment{
				protocol.NewOutputTextSegment(""),
			})
			events = append(events, protocol.NewItemStartedEvent(turnId, item))
		}
		// 发送增量更新（开头累积的空白不单独下发一条增量，直接随首条有效增量一并给出）
		events = append(events, protocol.NewItemUpdatedEvent(turnId, m.currentMessageId, &protocol.ItemDelta{
			Text: chunk.Content,
		}))
	}

	return events
}

// CompleteStreaming 完成当前 streaming item，发送 item_completed 事件
// 在 streaming 结束后调用
func (m *EventMapper) CompleteStreaming(turnId string, msg *session.Message) []*protocol.EventMsg {
	return m.completeActiveItems(turnId)
}

// completeActiveItems 定格当前进行中的 reasoning/message item（如有），并重置累积状态
func (m *EventMapper) completeActiveItems(turnId string) []*protocol.EventMsg {
	var events []*protocol.EventMsg

	// 完成推理 item（使用累积的内容）
	if m.currentReasoningId != "" {
		item := protocol.NewReasoningTurnItem(m.currentReasoningId, m.accumulatedReasoning)
		events = append(events, protocol.NewItemCompletedEvent(turnId, item))
		m.currentReasoningId = ""
	}

	// 完成消息 item（使用累积的内容）。仅空白正文不产出项：否则会在历史上留下空消息
	if m.currentMessageId != "" {
		if strings.TrimSpace(m.accumulatedContent) != "" {
			content := []protocol.ContentSegment{
				protocol.NewOutputTextSegment(m.accumulatedContent),
			}
			item := protocol.NewMessageTurnItem(m.currentMessageId, string(schema.Assistant), content)
			events = append(events, protocol.NewItemCompletedEvent(turnId, item))
		}
		m.currentMessageId = ""
	}

	// 重置累积内容
	m.accumulatedContent = ""
	m.accumulatedReasoning = ""

	return events
}

// MapToolCallEvent 将工具调用事件映射为 EventMsg
func (m *EventMapper) MapToolCallEvent(turnId string, msg *session.Message) []*protocol.EventMsg {
	var events []*protocol.EventMsg

	// 工具调用意味着本轮模型响应结束：先定格进行中的 reasoning/message item
	// （OutputItemDone → flush_active），下一轮响应的正文将新开 item（新 item_id），
	// 避免 agent 多轮输出的正文被合并成一条长文本、时序也被压到最后
	events = append(events, m.completeActiveItems(turnId)...)

	for _, tc := range msg.ToolCalls {
		// item_id 触发时刻生成（时间有序），历史查询按 item_id 排序即还原时序
		itemId := stringx.SortableUUID()
		// 记录 toolCallId -> itemId/arguments 映射，供 MapToolResultEvent 使用
		m.toolCallItemIds[tc.ID] = itemId
		m.toolCallArgs[tc.ID] = tc.Function.Arguments
		item := protocol.NewToolCallTurnItem(itemId, tc.ID, tc.Function.Name, tc.Function.Arguments)
		events = append(events, protocol.NewItemStartedEvent(turnId, item))
	}

	return events
}

// MapToolResultEvent 将工具结果事件映射为 EventMsg
func (m *EventMapper) MapToolResultEvent(ctx context.Context, turnId string, msg *session.Message) []*protocol.EventMsg {
	var events []*protocol.EventMsg

	// 恢复路径：工具从 checkpoint 恢复执行，复用挂起时已落库的原 item_id，
	// 不重发 item_started（ItemCompleted 复用原 item_id，仅发终态）
	itemId, isResumed := m.resumedToolCallItemIds[msg.ToolCallId]
	delete(m.resumedToolCallItemIds, msg.ToolCallId)
	isNewItem := false
	if !isResumed {
		// 使用 item_started 时记录的 itemId，确保前后端 ID 一致
		itemId = m.toolCallItemIds[msg.ToolCallId]
		if itemId == "" {
			// resume 流程中，工具从 checkpoint 恢复执行，MapToolCallEvent 未被调用
			// 需要同时发送 item_started 和 item_completed
			itemId = stringx.SortableUUID()
			isNewItem = true
		}
	}
	// 取回并清理映射。
	//
	// 参数以 **session 里的值为准**：参数补全场景下 ResumeParamCompletion 已把用户补全的最终
	// 参数回写进 tool_call 消息，而这份缓存记的是模型最初提交的原始参数（正是那个把必填留空、
	// 导致系统弹补全的版本）。原先只在缓存为空时才读 session，会让终态 item 用原始空值覆盖掉
	// 已回写的真值（真机核实：payload 里 authCertName 为空，output 里却是实际使用的凭证）。
	// session 查不到时回退缓存只对「正常调用事件」有效（其缓存在 MapToolCallEvent 时点填过原始参数）；
	// 恢复项走 TrackResumedToolCall 不填缓存，此时 session 若也读失败会落空 args——但不会 400，
	// 下游 itemToMessages 对空 args 自愈成 {}，代价仅审计 payload 丢参数（只在 DB 读失败边缘触发）
	args := m.toolCallArgs[msg.ToolCallId]
	delete(m.toolCallItemIds, msg.ToolCallId)
	delete(m.toolCallArgs, msg.ToolCallId)
	fresh := m.loadToolCallArgs(ctx, msg.ToolCallId)
	if fresh != "" {
		args = fresh
	}

	item := &protocol.TurnItem{
		Type:       protocol.TurnItemTypeToolCall,
		Id:         itemId,
		ToolCallId: msg.ToolCallId,
		ToolName:   msg.ToolName,
		Status:     protocol.TurnItemStatusSuccess,
		// 终态必须携带完整参数：api 层按 ItemId 去重保留最后一条，
		// 若终态丢参数，历史重建的 tool_calls 无 arguments 会导致后续请求 400
		Arguments: args,
		Output:    msg.Content,
	}

	// 检查是否有工具状态标记
	extra := collx.M(msg.Extra)
	if extra != nil {
		toolStatus := extra.GetStr(session.ExtraKeyToolStatus)
		if toolStatus == tools.ToolStatusError || toolStatus == tools.ToolStatusInterrupted {
			item.Status = protocol.TurnItemStatusFailed
			if toolStatus == tools.ToolStatusInterrupted {
				item.Status = protocol.TurnItemStatusInterrupted
			}
		}
	}
	// SafeToolMiddleware 将 RecoverRetry 错误转为字符串结果，
	// 此时 Extra 中无 toolStatus，需通过内容前缀检测
	if item.Status == protocol.TurnItemStatusSuccess && tools.IsToolErrorMsg(msg.Content) {
		item.Status = protocol.TurnItemStatusFailed
	}
	// 用户拒绝/取消的工具调用不视为执行失败：拒绝/取消 → Cancelled，
	// 徽章由 extra.interrupt.resume 表达，状态圆点用取消色。
	// 审批拒绝与参数补全取消时 eino 均将错误转为工具结果，无论错误标记是否透传，
	// 只要内容为拒绝/取消文案即归类为取消（与 turn 级 IsUserTerminated 的判据保持同源，
	// 否则同一轮里 turn 是 cancelled、item 却是 failed）
	if item.Status != protocol.TurnItemStatusInterrupted &&
		(tools.IsRejectedToolResult(msg.Content) || tools.IsCancelledToolResult(msg.Content)) {
		item.Status = protocol.TurnItemStatusCancelled
	}
	// 中断项不得把「请审批」说明当成工具结果落库：中断时工具根本没执行，
	// msg.Content 是中断说明（exec sql 场景为「执行SQL是高危操作，请审批」）。
	// 一旦落进 Output，历史重建与摘要都会把它当作执行结果——模型据此认为"还没批"，
	// 每次批准后又会发起下一次插入，形成自激循环（真机：一个 turn 内 6 次 ExecSql）
	if item.Status == protocol.TurnItemStatusInterrupted {
		item.Output = ""
	}

	// resume 流程：先发送 item_started，再发送 item_completed
	if isNewItem {
		startedItem := &protocol.TurnItem{
			Type:       protocol.TurnItemTypeToolCall,
			Id:         itemId,
			ToolCallId: msg.ToolCallId,
			ToolName:   msg.ToolName,
			// resume 流程中工具从 checkpoint 恢复执行，参数从 session 的 tool_call 消息恢复
			// （参数补全场景该消息已在 resume 时更新为用户补全后的参数）
			Arguments: m.loadToolCallArgs(ctx, msg.ToolCallId),
			Status:    protocol.TurnItemStatusPending,
		}
		events = append(events, protocol.NewItemStartedEvent(turnId, startedItem))
	}

	events = append(events, protocol.NewItemCompletedEvent(turnId, item))
	return events
}

// MapInterruptEvent 将中断信息映射为 EventMsg
func (m *EventMapper) MapInterruptEvent(turnId string, msg *session.Message) []*protocol.EventMsg {
	extra := collx.M(msg.Extra)
	if extra == nil {
		return nil
	}

	interruptType := extra.GetStr("type")
	actionId := GetActionId(msg)

	interrupt := &protocol.InterruptEvent{
		ActionId:    actionId,
		Type:        interruptType,
		Description: msg.Content,
		ToolName:    msg.ToolName,
		ToolCallId:  msg.ToolCallId,
	}

	// 尝试获取完整的中断元数据
	if info, ok := extra["content"].(tools.InterruptMetadata); ok {
		interrupt.Description = info.GetDescription()
		if toolInfo := info.GetToolInfo(); toolInfo != nil {
			interrupt.ToolName = toolInfo.Name
		}
		interrupt.ToolCallId = info.GetToolCallId()
		interrupt.Metadata = map[string]any{
			"title":   info.GetTitle(),
			"payload": info.GetPayload(),
		}
		// 类型特有的 metadata（如参数补全的 paramType/options）通过中断扩展注册表获取，
		// 新增中断类型无需修改此处
		for k, v := range tools.ExtraInterruptEventMetadata(info) {
			interrupt.Metadata[k] = v
		}
	}

	return []*protocol.EventMsg{
		protocol.NewInterruptedEvent(turnId, interrupt),
	}
}

// loadToolCallArgs 从 session 消息中恢复指定工具调用的参数
// （工具调用发生中断时，checkpoint 不含参数事件，但 session 的 tool_call 消息保存了参数，
// 且参数补全场景已在 resume 时更新为用户补全后的最终值）
func (m *EventMapper) loadToolCallArgs(ctx context.Context, toolCallId string) string {
	if toolCallId == "" {
		return ""
	}
	toolCallMsgs, err := session.DefaultSessionStore.GetMessage(ctx, &session.MessageQuery{MessageType: session.MsgTypeToolCall, ToolCallId: toolCallId})
	if err != nil || len(toolCallMsgs) == 0 {
		return ""
	}
	for _, tcMsg := range toolCallMsgs {
		for _, tc := range tcMsg.ToolCalls {
			if tc.ID == toolCallId {
				return tc.Function.Arguments
			}
		}
	}
	return ""
}
