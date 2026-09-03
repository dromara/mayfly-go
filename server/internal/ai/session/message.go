package session

import (
	"mayfly-go/pkg/utils/collx"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// RoleInternal 内部系统消息角色（中断、恢复等内部消息）
// 定义在 session 包以避免 application 层反向依赖 agent 包
const RoleInternal = schema.RoleType("internal")

// Message 会话消息，扩展自 adk.Message
type Message struct {
	// Id 消息ID
	Id       int64  `json:"id"`
	TurnId   string `json:"turnId"` // 所属 turn ID
	ActionId string `json:"actionId"`
	// Role 消息角色
	Role    schema.RoleType `json:"role"`
	MsgType string          `json:"msgType"` // 消息类型，如 "user", "assistant", "tool_call", "tool_result"等
	// Content 消息内容
	Content    string `json:"content"`
	ToolCallId string `json:"toolCallId"` // 工具调用ID
	// ToolCalls 工具调用列表
	ToolCalls []schema.ToolCall `json:"toolCalls"`
	// ToolName 工具名称（当 role=tool 时使用）
	ToolName string `json:"toolName,omitempty"`
	// Extra 额外信息
	Extra collx.M `json:"extra,omitempty"`
	// ResponseMeta 响应元信息
	ResponseMeta *schema.ResponseMeta `json:"responseMeta,omitempty"`
}

// ToAdkMessage 转换为 adk.Message
func (m *Message) ToAdkMessage() adk.Message {
	return &schema.Message{
		Role:         m.Role,
		Content:      m.Content,
		ToolCalls:    m.ToolCalls,
		ToolName:     m.ToolName,
		ToolCallID:   m.ToolCallId,
		Extra:        m.Extra,
		ResponseMeta: m.ResponseMeta,
	}
}

func (m *Message) GetToolCall(callId string) *schema.ToolCall {
	for _, toolCall := range m.ToolCalls {
		if toolCall.ID == callId {
			return &toolCall
		}
	}
	return nil
}

// FromAdkMessage 从 adk.Message 创建 Message
func FromAdkMessage(msg *schema.Message) *Message {
	extra := collx.M(msg.Extra)
	extra.Delete("reasoning-content") // 思考内容可能很多
	return &Message{
		TurnId:       extra.GetStr("turnId"), // 从 Extra 中获取 TurnId
		ActionId:     extra.GetStr("actionId"),
		ToolCallId:   msg.ToolCallID, // 工具调用ID
		Role:         msg.Role,
		Content:      msg.Content,
		ToolCalls:    msg.ToolCalls,
		ToolName:     msg.ToolName,
		Extra:        msg.Extra,
		ResponseMeta: msg.ResponseMeta,
	}
}

// ToAdkMessages 将 Message 切片转换为 schema.Message 切片
// 注意：internal 为内部运行时消息（中断/恢复等），非 LLM 协议合法角色，
// 网关会拒绝（role must be one of system/assistant/user/tool/function），
// 不参与 LLM 上下文，在此统一过滤（中断恢复走 GetMessage 单独查询，不受影响）
func ToAdkMessages(msgs []*Message) []adk.Message {
	result := make([]adk.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == RoleInternal {
			continue
		}
		result = append(result, m.ToAdkMessage())
	}
	return result
}

// FromAdkMessages 将 adk.Message 切片转换为 Message 切片
func FromAdkMessages(msgs []adk.Message) []*Message {
	result := make([]*Message, len(msgs))
	for i, m := range msgs {
		result[i] = FromAdkMessage(m)
	}
	return result
}
