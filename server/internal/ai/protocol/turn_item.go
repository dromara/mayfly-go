package protocol

import (
	"mayfly-go/pkg/utils/jsonx"
)

// TurnItem 类型常量
const (
	TurnItemTypeMessage    = "message"
	TurnItemTypeReasoning  = "reasoning"
	TurnItemTypeToolCall   = "tool_call"
	TurnItemTypeCompaction = "context_compaction"
)

// AttachmentMeta 用户消息附件元数据（message 变体扩展字段，随 payload 持久化）
// 仅承担展示职责：历史回显卡片与点击预览，不参与 LLM 输入。
// 附件内容经统一文件服务落 local/S3（t_sys_file），此处只持 fileKey 引用，
// 展示端以 /sys/files/{fileKey} 访问
// 对齐前端 protocol/types.ts 的 MessageAttachment
type AttachmentMeta struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"` // image | text | file
	Mime    string `json:"mime,omitempty"`
	Size    int64  `json:"size,omitempty"`
	FileKey string `json:"fileKey,omitempty"` // 文件服务 key（内容已落 t_sys_file）
}

// TurnItem 状态常量
const (
	TurnItemStatusPending     = "pending"
	TurnItemStatusSuccess     = "success"
	TurnItemStatusFailed      = "failed"
	TurnItemStatusCancelled   = "cancelled"
	TurnItemStatusInterrupted = "interrupted"
)

// TurnItem 统一协议（扁平变体，item.rs 的 serde tagged enum：
// 变体字段直接位于顶层、snake_case 命名，实时流（WS item_* 事件）与历史接口
// （t_ai_turn_item 快照回放）共享同一协议形状）
type TurnItem struct {
	Type string `json:"type,omitempty"` // message | reasoning | tool_call | context_compaction
	Id   string `json:"id,omitempty"`

	// message 变体
	Role        string           `json:"role,omitempty"`
	Content     []ContentSegment `json:"content,omitempty"`
	Attachments []AttachmentMeta `json:"attachments,omitempty"` // 用户消息附件元数据（历史回显卡片/预览）

	// reasoning 变体
	Text string `json:"text,omitempty"`

	// tool_call 变体
	ToolCallId string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	Arguments  string `json:"arguments,omitempty"`
	Status     string `json:"status,omitempty"` // pending | success | failed | cancelled | interrupted
	Output     string `json:"output,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`

	// context_compaction 变体
	OriginalTokens         int `json:"original_tokens,omitempty"`
	CompressedTokens       int `json:"compressed_tokens,omitempty"`
	CompressedMessageCount int `json:"compressed_message_count,omitempty"`
}

// PayloadJSON 序列化为存储 payload（剥离 type/id——分别由 item_type / item_id 列承载，
// TurnItem::payload_json；与 FromPayload 互逆，收敛存储形状的序列化/组装逻辑）
func (t *TurnItem) PayloadJSON() string {
	clone := *t
	clone.Type = ""
	clone.Id = ""
	return jsonx.ToStr(&clone)
}

// FromPayload 从存储行组装 TurnItem（payload 已剥离 type/id，由 item_type / item_id 列回填）
func FromPayload(itemType, itemId, payload string) (*TurnItem, error) {
	ti, err := jsonx.ToByStr[TurnItem](payload)
	if err != nil {
		return nil, err
	}
	ti.Type = itemType
	ti.Id = itemId
	return ti, nil
}

// NewMessageTurnItem 创建消息类型 TurnItem
func NewMessageTurnItem(id, role string, content []ContentSegment) *TurnItem {
	return &TurnItem{
		Type:    TurnItemTypeMessage,
		Id:      id,
		Role:    role,
		Content: content,
	}
}

// NewReasoningTurnItem 创建推理类型 TurnItem
func NewReasoningTurnItem(id, text string) *TurnItem {
	return &TurnItem{
		Type: TurnItemTypeReasoning,
		Id:   id,
		Text: text,
	}
}

// NewToolCallTurnItem 创建工具调用类型 TurnItem
func NewToolCallTurnItem(id, toolCallId, toolName, arguments string) *TurnItem {
	return &TurnItem{
		Type:       TurnItemTypeToolCall,
		Id:         id,
		ToolCallId: toolCallId,
		ToolName:   toolName,
		Arguments:  arguments,
		Status:     TurnItemStatusPending,
	}
}

// NewCompactionTurnItem 创建上下文压缩事件项 TurnItem
func NewCompactionTurnItem(id string, originalTokens, compressedTokens, compressedMessageCount int) *TurnItem {
	return &TurnItem{
		Type:                   TurnItemTypeCompaction,
		Id:                     id,
		OriginalTokens:         originalTokens,
		CompressedTokens:       compressedTokens,
		CompressedMessageCount: compressedMessageCount,
	}
}
