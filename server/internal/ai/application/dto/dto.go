package dto

import (
	"mayfly-go/internal/ai/protocol"
	"time"
)

// ConversationQuery 会话列表查询
type ConversationQuery struct {
	UserId uint64
}

// CreateConversationReq 创建会话请求
type CreateConversationReq struct {
	Title string `json:"title"`
}

// TurnGroupDTO turn 分组 DTO
type TurnGroupDTO struct {
	TurnId string         `json:"turnId"`
	Items  []*TurnItemDTO `json:"items"`
}

// TurnItemDTO turn item DTO
type TurnItemDTO struct {
	Id             uint64             `json:"id"`
	ConversationId uint64             `json:"conversationId"`
	TurnId         string             `json:"turnId"`
	ItemType       string             `json:"itemType"`
	ItemId         string             `json:"itemId"`
	Item           *protocol.TurnItem `json:"item,omitempty"`
	Status         string             `json:"status"`
	ToolCallId     string             `json:"toolCallId,omitempty"`
	// Extra 扩展列：tool_call item 携带中断信息
	// {"interrupt": {...}, "resumeInfo": {...}}，前端据此恢复中断卡片
	Extra      map[string]any `json:"extra,omitempty"`
	CreateTime *time.Time     `json:"createTime"`
}
