package vo

import (
	"mayfly-go/internal/ai/protocol"
	"time"
)

// ConversationVO 会话列表 VO
type ConversationVO struct {
	Id               uint64 `json:"id"`
	Code             string `json:"code"`
	Title            string `json:"title"`
	Status           int    `json:"status"`
	MessageCount     int    `json:"totalMessageCount"`
	PromptTokens     int64  `json:"promptTokens"`
	CompletionTokens int64  `json:"completionTokens"`
	TotalTokens      int64  `json:"totalTokens"`
	CreateTime       string `json:"createTime"`
	UpdateTime       string `json:"updateTime"`
}

// TurnGroupVO turn 分组 VO
type TurnGroupVO struct {
	TurnId string        `json:"turnId"`
	Items  []*TurnItemVO `json:"items"`
}

// TurnItemVO turn item VO
type TurnItemVO struct {
	Id         uint64             `json:"id"`
	TurnId     string             `json:"turnId"`
	ItemType   string             `json:"itemType"`
	ItemId     string             `json:"itemId"`
	Item       *protocol.TurnItem `json:"item,omitempty"`
	Status     string             `json:"status"`
	ToolCallId string             `json:"toolCallId,omitempty"`
	// Extra 扩展列（对齐 tokhub）：tool_call item 携带中断信息
	// {"interrupt": {...}}，前端据此恢复恢复类型徽章（resume.type）
	Extra      map[string]any `json:"extra,omitempty"`
	CreateTime *time.Time     `json:"createTime"`
}

// RunningTurnVO 运行中 turn VO（会话列表执行中指示器数据源）
type RunningTurnVO struct {
	ConversationId uint64 `json:"conversationId"`
	TurnId         string `json:"turnId"`
}
