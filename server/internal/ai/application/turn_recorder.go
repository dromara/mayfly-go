package application

import (
	"context"
	"sync"

	"mayfly-go/internal/ai/agent"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/internal/ai/session"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"

	"github.com/cloudwego/eino/schema"
)

// TurnRecorder 轮次记录器：统一承担「agent 消息 → protocol.EventMsg 映射下发 +
// TurnItem 收集 → 收尾去重持久化」全链路，api 层仅保留 WS 传输职责。
//
// 职责收敛（消除此前 api 层三条并行收集路径：OnEvent collect 闭包、
// runningTurn.CollectItems 用户消息预收集、markInterruptedToolCall 直改切片）：
//   - 消息映射：经 agent.EventMapper 将 session.Message 映射为 protocol.EventMsg
//     （流式 chunk / 工具调用事件 / 工具结果 / 中断事件统一分发）
//   - item 收集：单一 Collect 入口（并发安全），中断信息统一落到被中断
//     tool_call item 的 extra 列（不产生独立 internal item，对齐 tokhub）
//   - 恢复预加载：加载挂起 item 注册 item_id 复用映射，恢复决策按 request_id 索引
//   - 收尾持久化：按 ItemId 去重，恢复路径复用行走按行更新，新 item 批量插入
//
// 并发契约：事件回调由 Agent.Run 同步调用（单 goroutine），用户消息预收集在
// WS 循环 goroutine —— Collect 以互斥锁覆盖两处触达；publish 与 Flush 由
// runTurn 串行调用。
type TurnRecorder struct {
	turnItemApp TurnItem
	convId      uint64
	turnId      string
	// publisher 事件下发通道（api 层注入 turn 事件总线；nil 时仅收集不发布）
	publisher func(*protocol.EventMsg)

	mu    sync.Mutex
	items []*entity.TurnItem

	eventMapper    *agent.EventMapper
	resumeByItemId map[string]*protocol.InterruptResume
}

// NewTurnRecorder 创建轮次记录器
func NewTurnRecorder(turnItemApp TurnItem, convId uint64, turnId string, publisher func(*protocol.EventMsg)) *TurnRecorder {
	return &TurnRecorder{
		turnItemApp:    turnItemApp,
		convId:         convId,
		turnId:         turnId,
		publisher:      publisher,
		eventMapper:    agent.NewEventMapper(),
		resumeByItemId: make(map[string]*protocol.InterruptResume),
	}
}

// OnChunk 流式增量回调（agent.WithOnChunk 接线点）：过滤工具调用/结果消息，
// 正文与推理内容映射为流式事件下发
func (r *TurnRecorder) OnChunk(ctx context.Context, m *session.Message) error {
	if len(m.ToolCalls) > 0 || m.Role == schema.Tool {
		return nil
	}
	for _, evt := range r.eventMapper.MapChunk(r.turnId, m) {
		r.publish(evt)
	}
	return nil
}

// OnEvent 完整事件回调（agent.WithOnEvent 接线点）：按消息形态分发映射，
// 事件下发并收集对应 TurnItem
func (r *TurnRecorder) OnEvent(ctx context.Context, ae *agent.AgentEvent, m *session.Message) error {
	currentTurnId := agent.GetTurnId(m)
	if currentTurnId == "" {
		currentTurnId = r.turnId
	}
	if len(m.ToolCalls) > 0 {
		for _, evt := range r.eventMapper.MapToolCallEvent(currentTurnId, m) {
			r.publish(evt)
			// 收集 TurnItem（MapToolCallEvent 会先发 reasoning/message 的 item_completed，
			// 终态事件落 success，进行中的 tool_call 落 active）
			if evt.Item != nil {
				status := entity.ItemStatusActive
				if evt.Type == protocol.EventTypeItemCompleted {
					status = entity.ItemStatusSuccess
				}
				r.CollectProtocolItem(evt.TurnId, evt.Item, status)
			}
		}
		return nil
	}
	if m.Role == schema.Tool {
		for _, evt := range r.eventMapper.MapToolResultEvent(ctx, currentTurnId, m) {
			r.publish(evt)
			// 收集 TurnItem（使用事件中的实际状态，而非硬编码 success）
			if evt.Item != nil {
				r.CollectProtocolItem(evt.TurnId, evt.Item, toolCallItemStatus(evt.Item))
			}
		}
		return nil
	}
	if m.Role == session.RoleInternal {
		extra := collx.M(m.Extra)
		if extra != nil && tools.IsInterruptContent(extra["content"]) {
			// 通过中断元数据泛化判断（IsInterruptContent），新增中断类型无需修改此处
			interruptEvents := r.eventMapper.MapInterruptEvent(currentTurnId, m)
			for _, evt := range interruptEvents {
				r.publish(evt)
			}
			// 对齐 tokhub：中断信息统一存到被中断工具调用的 tool_call item
			// extra 列（{"interrupt": InterruptInfo}），不产生独立 internal item
			if len(interruptEvents) > 0 && interruptEvents[0].Interrupt != nil {
				if !r.MarkInterrupted(interruptEvents[0].Interrupt) {
					logx.Warnf("interrupted tool_call item not found, toolCallId=%s",
						interruptEvents[0].Interrupt.ToolCallId)
				}
			}
		}
		return nil
	}
	return nil
}

// FinishStreaming 完成流式输出：映射收尾事件并收集最终 TurnItem
func (r *TurnRecorder) FinishStreaming() {
	for _, evt := range r.eventMapper.CompleteStreaming(r.turnId, nil) {
		if evt.Item != nil {
			r.CollectProtocolItem(evt.TurnId, evt.Item, entity.ItemStatusSuccess)
		}
		r.publish(evt)
	}
}

// TrackResumedToolCalls 恢复路径预加载：从持久化层加载该 turn 挂起的 interrupted
// tool_call item，注册 toolCallId → 原 itemId 到 EventMapper（工具完成时复用原
// item_id、更新同一行，对齐 tokhub execute.rs 的第一阶段挂起行加载）；同时按
// request_id 匹配恢复决策，建立 itemId → 决策映射供收尾持久化时 merge 到
// interrupt.resume
func (r *TurnRecorder) TrackResumedToolCalls(ctx context.Context, resumeParams []any) {
	items, err := r.turnItemApp.SelectByTurnId(ctx, r.convId, r.turnId)
	if err != nil {
		logx.Errorf("load interrupted items for resume error: %v", err)
		return
	}

	// 恢复决策按 interruptId（即 extra.interrupt.request_id）索引
	resumeByRequestId := make(map[string]*tools.InterruptResume)
	for _, p := range resumeParams {
		if resume, ok := p.(*tools.InterruptResume); ok {
			resumeByRequestId[resume.InterruptId] = resume
		}
	}

	for _, item := range items {
		if item.ItemType != entity.ItemTypeToolCall || item.Status != protocol.TurnItemStatusInterrupted {
			continue
		}
		info, err := jsonx.ToByStr[protocol.InterruptInfo](jsonx.ToStr(item.Extra["interrupt"]))
		if err != nil || info == nil || info.Kind == "" {
			continue
		}
		r.eventMapper.TrackResumedToolCall(item.ToolCallId, item.ItemId)
		if resume, ok := resumeByRequestId[info.RequestId]; ok {
			r.resumeByItemId[item.ItemId] = protocol.NewResumeFromResumeInfo(resume)
		}
	}
}

// Collect 收集 TurnItem（并发安全：用户消息预收集在 WS 循环 goroutine，
// 事件回调在 runTurn goroutine，两处可能先后触达）
func (r *TurnRecorder) Collect(items ...*entity.TurnItem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, items...)
}

// CollectProtocolItem 收集 protocol.TurnItem（统一转换 entity 落库口径）
func (r *TurnRecorder) CollectProtocolItem(turnId string, item *protocol.TurnItem, status string) {
	r.Collect(toEntityTurnItem(r.convId, turnId, item, status))
}

// MarkInterrupted 将中断信息落到被中断工具调用的 tool_call item 上
// （对齐 tokhub：extra["interrupt"] 存 InterruptInfo，item 状态置 interrupted）。
// 中断必然由工具审批/参数补全触发，本轮已收集的 tool_call item 中必能命中；
// 返回是否命中（未命中仅告警，中断信息随事件流下发、不落库）
func (r *TurnRecorder) MarkInterrupted(evt *protocol.InterruptEvent) bool {
	if evt == nil || evt.ToolCallId == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// 倒序取最后收集的同 toolCallId item（与落库去重“保留最后一条”口径一致）
	for i := len(r.items) - 1; i >= 0; i-- {
		item := r.items[i]
		if item.ItemType == entity.ItemTypeToolCall && item.ToolCallId == evt.ToolCallId {
			item.Status = protocol.TurnItemStatusInterrupted
			if item.Extra == nil {
				item.Extra = collx.M{}
			}
			item.Extra["interrupt"] = protocol.NewInterruptInfo(evt, r.convId)
			return true
		}
	}
	return false
}

// Flush 收尾持久化（runTurn 收尾调用，ctx 应为无取消 ctx）：
// 按 ItemId 去重（streaming 中间态 + 完成态保留最后一条）后分流保存 ——
// 恢复路径复用原 item_id 的工具调用走按行更新（BatchInsert 会产生重复行），
// 其余新 item 批量插入
func (r *TurnRecorder) Flush(ctx context.Context) {
	r.mu.Lock()
	items := r.items
	r.items = nil
	r.mu.Unlock()
	if len(items) == 0 {
		return
	}
	deduped := deduplicateTurnItems(items)
	inserts := make([]*entity.TurnItem, 0, len(deduped))
	for _, item := range deduped {
		if resume, ok := r.resumeByItemId[item.ItemId]; ok {
			if updErr := r.turnItemApp.UpdateResumedToolCallItem(ctx, r.convId, item, resume); updErr != nil {
				logx.Errorf("update resumed tool_call item error: %v", updErr)
			}
			continue
		}
		inserts = append(inserts, item)
	}
	if len(inserts) > 0 {
		if saveErr := r.turnItemApp.BatchSaveTurnItems(ctx, inserts); saveErr != nil {
			logx.Errorf("save turn items error: %v", saveErr)
		}
	}
}

// publish 事件下发（publisher 未注入时丢弃）
func (r *TurnRecorder) publish(evt *protocol.EventMsg) {
	if r.publisher == nil || evt == nil {
		return
	}
	r.publisher(evt)
}

// toEntityTurnItem 将 protocol.TurnItem 转换为 entity.TurnItem
// （payload 剥离 type/id，由 item_type / item_id 列承载，对齐 tokhub payload_json）
func toEntityTurnItem(convId uint64, turnId string, item *protocol.TurnItem, status string) *entity.TurnItem {
	return &entity.TurnItem{
		ConversationId: convId,
		TurnId:         turnId,
		ItemId:         item.Id,
		ItemType:       item.Type,
		Payload:        item.PayloadJSON(),
		Status:         status,
		ToolCallId:     item.ToolCallId,
	}
}

// toolCallItemStatus 从 TurnItem 提取工具调用的实际状态（取值统一用 entity.ItemStatus*）
func toolCallItemStatus(item *protocol.TurnItem) string {
	switch item.Status {
	case protocol.TurnItemStatusFailed:
		return entity.ItemStatusFailed
	case protocol.TurnItemStatusInterrupted:
		return entity.ItemStatusInterrupted
	case protocol.TurnItemStatusSuccess:
		return entity.ItemStatusSuccess
	case protocol.TurnItemStatusCancelled:
		// 恢复路径用户拒绝的真实终态（对齐 tokhub 拒绝 → Cancelled）
		return entity.ItemStatusCancelled
	default:
		return entity.ItemStatusActive
	}
}

// deduplicateTurnItems 按 ItemId 去重，保留每个 ItemId 的最后一条记录
func deduplicateTurnItems(items []*entity.TurnItem) []*entity.TurnItem {
	seen := make(map[string]int) // ItemId -> index in result
	result := make([]*entity.TurnItem, 0, len(items))
	for _, item := range items {
		if idx, ok := seen[item.ItemId]; ok {
			result[idx] = item // 替换为更新的版本
		} else {
			seen[item.ItemId] = len(result)
			result = append(result, item)
		}
	}
	return result
}
