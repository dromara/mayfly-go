package protocol

// EventMsg 类型常量
const (
	EventTypeItemStarted   = "item_started"
	EventTypeItemUpdated   = "item_updated"
	EventTypeItemCompleted = "item_completed"
	EventTypeTurnStarted   = "turn_started"
	EventTypeTurnCompleted = "turn_completed"
	EventTypeInterrupted   = "interrupted"
	EventTypeError         = "error"
	EventTypeEnd           = "end"
	EventTypeHeartbeat     = "heartbeat"

	// turn 运行时订阅协议（turn 与 WS 连接解耦：刷新/断线重连后 attach 续流）
	EventTypeTurnAttached        = "turn_attached"        // attach 成功：回放缓存快照完毕，后续为实时事件
	EventTypeTurnNotRunning      = "turn_not_running"     // attach 目标无运行中 turn（走历史加载）
	EventTypeConversationCreated = "conversation_created" // 新会话创建成功（携带 conversationId）
)

// TurnCompletedEvent 的 status 取值（turn 级终态，与 item 级 TurnItemStatus 是另一值域）
const (
	TurnStatusSuccess = "success"
	TurnStatusStopped = "stopped" // 显式 stop 触发的 ctx 取消
	TurnStatusFailed  = "failed"
)

// EventMsg 结构化事件协议
// 对齐 tokhub 的 EventMsg，用于 WebSocket 推送
type EventMsg struct {
	Type   string `json:"type"` // item_started | item_updated | item_completed | turn_started | turn_completed | interrupted | error | end | heartbeat
	TurnId string `json:"turnId,omitempty"`

	// item 级事件
	Item   *TurnItem  `json:"item,omitempty"`   // item_started / item_completed 携带
	ItemId string     `json:"itemId,omitempty"` // item_updated 携带
	Delta  *ItemDelta `json:"delta,omitempty"`  // item_updated 增量

	// turn 级事件
	ConversationId uint64     `json:"conversationId,omitempty"`
	Status         string     `json:"status,omitempty"`
	Usage          *TurnUsage `json:"usage,omitempty"`

	// 中断事件
	Interrupt *InterruptEvent `json:"interrupt,omitempty"`

	// 错误
	Error     string `json:"error,omitempty"`
	ErrSource string `json:"errSource,omitempty"`
}

// ItemDelta item 增量更新
type ItemDelta struct {
	Text      string `json:"text,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// TurnUsage turn 级别的 token 使用统计
type TurnUsage struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
	TotalTokens  int64 `json:"totalTokens"`
}

// NewItemStartedEvent 创建 item 开始事件
func NewItemStartedEvent(turnId string, item *TurnItem) *EventMsg {
	return &EventMsg{
		Type:   EventTypeItemStarted,
		TurnId: turnId,
		Item:   item,
	}
}

// NewItemUpdatedEvent 创建 item 更新事件
func NewItemUpdatedEvent(turnId, itemId string, delta *ItemDelta) *EventMsg {
	return &EventMsg{
		Type:   EventTypeItemUpdated,
		TurnId: turnId,
		ItemId: itemId,
		Delta:  delta,
	}
}

// NewItemCompletedEvent 创建 item 完成事件
func NewItemCompletedEvent(turnId string, item *TurnItem) *EventMsg {
	return &EventMsg{
		Type:   EventTypeItemCompleted,
		TurnId: turnId,
		Item:   item,
	}
}

// NewTurnStartedEvent 创建 turn 开始事件
func NewTurnStartedEvent(turnId string, conversationId uint64) *EventMsg {
	return &EventMsg{
		Type:           EventTypeTurnStarted,
		TurnId:         turnId,
		ConversationId: conversationId,
	}
}

// NewConversationCreatedEvent 创建新会话已创建事件
func NewConversationCreatedEvent(conversationId uint64) *EventMsg {
	return &EventMsg{
		Type:           EventTypeConversationCreated,
		ConversationId: conversationId,
	}
}

// NewTurnCompletedEvent 创建 turn 完成事件
func NewTurnCompletedEvent(turnId string, status string, usage *TurnUsage) *EventMsg {
	return &EventMsg{
		Type:   EventTypeTurnCompleted,
		TurnId: turnId,
		Status: status,
		Usage:  usage,
	}
}

// NewInterruptedEvent 创建中断事件
func NewInterruptedEvent(turnId string, interrupt *InterruptEvent) *EventMsg {
	return &EventMsg{
		Type:      EventTypeInterrupted,
		TurnId:    turnId,
		Interrupt: interrupt,
	}
}

// NewErrorEvent 创建错误事件
func NewErrorEvent(errMsg string, source string) *EventMsg {
	return &EventMsg{
		Type:      EventTypeError,
		Error:     errMsg,
		ErrSource: source,
	}
}

// NewEndEvent 创建结束事件
func NewEndEvent() *EventMsg {
	return &EventMsg{
		Type: EventTypeEnd,
	}
}

// NewTurnAttachedEvent 创建 turn 订阅成功事件（回放后、实时事件前写入）
func NewTurnAttachedEvent(turnId string, conversationId uint64) *EventMsg {
	return &EventMsg{
		Type:           EventTypeTurnAttached,
		TurnId:         turnId,
		ConversationId: conversationId,
	}
}

// NewTurnNotRunningEvent 创建 turn 未运行事件（attach 目标无运行中 turn）
func NewTurnNotRunningEvent(conversationId uint64) *EventMsg {
	return &EventMsg{
		Type:           EventTypeTurnNotRunning,
		ConversationId: conversationId,
	}
}
