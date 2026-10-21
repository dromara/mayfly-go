package entity

import (
	"mayfly-go/pkg/model"
)

// TurnItem 类型常量
const (
	ItemTypeMessage    = "message"            // 普通消息
	ItemTypeReasoning  = "reasoning"          // 推理内容
	ItemTypeToolCall   = "tool_call"          // 工具调用
	ItemTypeCompaction = "context_compaction" // 上下文压缩（自动摘要落库的事件项）
	// ItemTypeTurnNotice 轮次级终止提示（目前只有「用户显式停止生成」）。
	//
	// 为什么必须落库而不是只在前端画：这条提示标的是「本轮被人为截断」。只活在客户端内存里时，
	// 刷新/重进会话后那一轮只剩一段在句子中途断开的回答，与模型正常答完完全同形——
	// 运维会把「被人停了」当成「就说到这里」
	ItemTypeTurnNotice = "turn_notice"
)

// TurnItem 落库状态常量（值域与 protocol.TurnItemStatus 一致，另含流式进行中态 active；
// 落库侧统一从此处取值，禁止字面量散落）
const (
	ItemStatusActive      = "active"      // 流式进行中（终态事件覆盖前的中间态）
	ItemStatusPending     = "pending"     // 待执行
	ItemStatusSuccess     = "success"     // 成功
	ItemStatusFailed      = "failed"      // 失败
	ItemStatusCancelled   = "cancelled"   // 取消（恢复路径用户拒绝的真实终态）
	ItemStatusInterrupted = "interrupted" // 中断挂起
)

// LLM 消息类型常量（session.Message.MsgType 取值，由 TurnItem 转换得到）
const (
	MsgTypeUser       = "user"        // 用户消息
	MsgTypeAssistant  = "assistant"   // AI 回复（普通回复）
	MsgTypeToolCall   = "tool_call"   // 工具调用（assistant 带 tool_calls）
	MsgTypeToolResult = "tool_result" // 工具结果（role = tool）
)

// TurnItem AI Turn 内原子产出单元（替代 SessionMessage）
// TurnItemRow：payload 为唯一事实源，仅 tool_call_id 建查询列；
// 中断信息存于 tool_call item 的 extra 列（{"interrupt": InterruptEvent}，
// resumeInfo 经 UpdateMessage 回写同列），actionId 自含于中断信息中
// ExtraData 提供 extra JSON 附加字段（业务扩展，随 item 生命周期透传）
type TurnItem struct {
	model.CreateModel
	model.ExtraData

	ConversationId uint64 `gorm:"column:conversation_id;not null;comment:会话ID" json:"conversationId"`
	TurnId         string `gorm:"column:turn_id;size:64;not null;comment:Turn ID" json:"turnId"`
	ItemType       string `gorm:"column:item_type;size:50;not null;comment:message|reasoning|tool_call|context_compaction|turn_notice" json:"itemType"`
	ItemId         string `gorm:"column:item_id;size:64;not null;default:'';comment:TurnItem 业务 ID（UUID）" json:"itemId"`
	Payload        string `gorm:"column:payload;type:mediumtext;comment:TurnItem JSON 快照" json:"payload"`
	Status         string `gorm:"column:status;size:50;not null;default:pending;comment:pending|success|failed|cancelled|interrupted" json:"status"`
	ToolCallId     string `gorm:"column:tool_call_id;size:64;comment:ToolCall item 冗余（工具调用/中断决策定位）" json:"toolCallId"`
}

// 轮次终止提示的种类（TurnItem.ItemType=turn_notice 时生效）。
//
// 入库的是种类而不是文案：文案得按读取方当时的语言现渲，落死句子会把写入那一刻的语言
// 固化进历史（中文界面停的一轮，英文界面回看仍是中文），也与本仓「模型/事实进库、文案归 i18n」口径一致
const (
	TurnNoticeKindStopped = "stopped" // 用户显式停止本轮生成
)

func (t *TurnItem) TableName() string {
	return "t_ai_turn_item"
}
