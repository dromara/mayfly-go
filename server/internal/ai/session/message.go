package session

import (
	"strings"

	"mayfly-go/pkg/utils/collx"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// RoleInternal 内部系统消息角色（中断、恢复等内部消息）
// 定义在 session 包以避免 application 层反向依赖 agent 包。
// 该角色仅在内存/持久化层流转，绝不进入 LLM 请求（ToAgenticMessages 统一过滤）
const RoleInternal = schema.RoleType("internal")

// Message 会话消息（全链路统一内存消息结构）
//
// api / agent / event_mapper / memory 各层均以本结构流转；eino 的
// *schema.AgenticMessage（AgenticMessage 路径，eino v0.9 Typed API）类型被
// 收敛于 ToAgenticMessage / FromAgenticMessage 两个转换边界——eino 消息
// 形态演进只需修改转换层，上层零改动（开闭边界）。
//
// 字段与 turn_item 存储协议（扁平变体）一一对应：message item 的 content
// 段数组、tool_call item 的 arguments/output 均在重建时填充到本结构。
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
	// ImageUrls 用户消息附图 URL 列表（多模态输入：截图分析等场景；
	// 由 turn_item 的 image 段重建，转 AgenticMessage 时生成 UserInputImage block）
	ImageUrls []string `json:"imageUrls,omitempty"`
	// ReasoningContent 思考增量（仅流式 chunk 回调传递用；终态思考内容经
	// reasoning item 独立落库，不持久化到本字段）
	ReasoningContent string `json:"-"`
	// Extra 额外信息
	Extra collx.M `json:"extra,omitempty"`
	// ResponseMeta 响应元信息（归一化承载 Usage；模型未上报时为 nil）
	ResponseMeta *schema.ResponseMeta `json:"responseMeta,omitempty"`
}

// toAgenticRole 内存角色映射为 Agentic 角色
//
// AgenticRoleType 仅有 system/user/assistant 三值：tool 角色由
// FunctionToolResult block 表达（宿主 role 对齐 adk 内部桥接行为，用 user）；
// internal 为内部角色，不进入 LLM 请求，仅为转换完整性做字符串映射。
func toAgenticRole(role schema.RoleType) schema.AgenticRoleType {
	switch role {
	case schema.Assistant:
		return schema.AgenticRoleTypeAssistant
	case schema.System:
		return schema.AgenticRoleTypeSystem
	default:
		return schema.AgenticRoleType(schema.RoleType(role))
	}
}

// ToAgenticMessage 转换为 eino AgenticMessage（LLM 请求/Agent 执行的边界转换）
func (m *Message) ToAgenticMessage() adk.AgenticMessage {
	msg := &schema.AgenticMessage{
		Role:  toAgenticRole(m.Role),
		Extra: m.Extra,
	}

	switch {
	case m.Role == schema.Tool:
		// 工具结果：FunctionToolResult block（对齐 adk toolMessageToAgenticMessage：
		// 宿主 role 为 user，结果文本经 UserInputText 承载）
		msg.Role = schema.AgenticRoleTypeUser
		msg.ContentBlocks = []*schema.ContentBlock{
			{
				Type: schema.ContentBlockTypeFunctionToolResult,
				FunctionToolResult: &schema.FunctionToolResult{
					CallID: m.ToolCallId,
					Name:   m.ToolName,
					Content: []*schema.FunctionToolResultContentBlock{
						{Type: schema.FunctionToolResultContentBlockTypeText, Text: &schema.UserInputText{Text: m.Content}},
					},
				},
			},
		}
	case len(m.ToolCalls) > 0:
		// 助手工具调用：FunctionToolCall block（与正文混排时正文在前）
		if m.Content != "" {
			msg.ContentBlocks = append(msg.ContentBlocks, newTextBlock(m.Content, false))
		}
		for _, tc := range m.ToolCalls {
			msg.ContentBlocks = append(msg.ContentBlocks, &schema.ContentBlock{
				Type: schema.ContentBlockTypeFunctionToolCall,
				FunctionToolCall: &schema.FunctionToolCall{
					CallID:    tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	default:
		// 普通消息：正文 block（user 输入经 UserInputText，assistant 生成经 AssistantGenText）
		if m.Content != "" || len(m.ImageUrls) > 0 {
			textRole := m.Role == schema.Assistant
			if m.Content != "" {
				msg.ContentBlocks = append(msg.ContentBlocks, newTextBlock(m.Content, textRole))
			}
			for _, url := range m.ImageUrls {
				msg.ContentBlocks = append(msg.ContentBlocks, &schema.ContentBlock{
					Type:           schema.ContentBlockTypeUserInputImage,
					UserInputImage: &schema.UserInputImage{URL: url},
				})
			}
		}
	}
	return msg
}

// newTextBlock 按角色构造正文 block（assistant 生成用 AssistantGenText，其余用 UserInputText）
func newTextBlock(text string, assistant bool) *schema.ContentBlock {
	if assistant {
		return &schema.ContentBlock{Type: schema.ContentBlockTypeAssistantGenText, AssistantGenText: &schema.AssistantGenText{Text: text}}
	}
	return &schema.ContentBlock{Type: schema.ContentBlockTypeUserInputText, UserInputText: &schema.UserInputText{Text: text}}
}

// FromAgenticMessage 从 eino AgenticMessage 创建 Message（事件消费的边界转换）
//
// 将 ContentBlocks 解构回本结构的扁平字段：正文/工具调用/工具结果分别映射
// Content、ToolCalls、ToolCallId/ToolName；Usage 归一化到 ResponseMeta。
func FromAgenticMessage(msg *schema.AgenticMessage) *Message {
	if msg == nil {
		return nil
	}
	m := &Message{
		Role:  schema.RoleType(msg.Role),
		Extra: msg.Extra,
	}

	var sb strings.Builder
	for _, block := range msg.ContentBlocks {
		if block == nil {
			continue
		}
		switch {
		case block.Reasoning != nil:
			// 思考内容：仅流式 chunk 增量传递用（终态经 reasoning item 独立落库）
			m.ReasoningContent += block.Reasoning.Text
			if ext := block.Reasoning.OpenAIExtension; ext != nil {
				for _, rc := range ext.Content {
					if rc != nil {
						m.ReasoningContent += rc.Text
					}
				}
			}
		case block.AssistantGenText != nil:
			sb.WriteString(block.AssistantGenText.Text)
		case block.UserInputText != nil:
			sb.WriteString(block.UserInputText.Text)
		case block.FunctionToolCall != nil:
			ftc := block.FunctionToolCall
			m.ToolCalls = append(m.ToolCalls, schema.ToolCall{
				ID:       ftc.CallID,
				Type:     "function",
				Function: schema.FunctionCall{Name: ftc.Name, Arguments: ftc.Arguments},
			})
		case block.FunctionToolResult != nil:
			ftr := block.FunctionToolResult
			// 工具结果：宿主 role 为 user（ToAgenticMessage 与 adk 内部桥接一致，
			// tool 语义由 FunctionToolResult block 承载），此处逆映射恢复 Tool 角色，
			// 否则下游 role==Tool 分支（工具结果事件落库/日志分类）全部失效
			m.Role = schema.Tool
			m.ToolCallId = ftr.CallID
			m.ToolName = ftr.Name
			// 结果文本：聚合同一 block 的全部 text 段（多模态结果仅保留文本，
			// 图像等结果内容经工具侧落库，不经消息结构回传）
			for _, c := range ftr.Content {
				if c != nil && c.Text != nil {
					sb.WriteString(c.Text.Text)
				}
			}
		}
	}
	m.Content = sb.String()

	// Usage 归一化（FinishReason 由 agent 层从组件扩展读取，不在此归一化）
	if msg.ResponseMeta != nil && msg.ResponseMeta.TokenUsage != nil {
		m.ResponseMeta = &schema.ResponseMeta{Usage: msg.ResponseMeta.TokenUsage}
	}
	m.TurnId = m.Extra.GetStr("turnId") // 从 Extra 中获取 TurnId
	m.ActionId = m.Extra.GetStr("actionId")
	return m
}

// ExtractText 提取 AgenticMessage 中的全部文本块内容（正文聚合，忽略工具/思考块）
func ExtractText(msg *schema.AgenticMessage) string {
	if msg == nil {
		return ""
	}
	var sb strings.Builder
	for _, block := range msg.ContentBlocks {
		if block == nil {
			continue
		}
		if block.AssistantGenText != nil {
			sb.WriteString(block.AssistantGenText.Text)
		} else if block.UserInputText != nil {
			sb.WriteString(block.UserInputText.Text)
		}
	}
	return sb.String()
}

func (m *Message) GetToolCall(callId string) *schema.ToolCall {
	for _, toolCall := range m.ToolCalls {
		if toolCall.ID == callId {
			return &toolCall
		}
	}
	return nil
}

// ToAgenticMessages 将 Message 切片转换为 AgenticMessage 切片
// 注意：internal 为内部运行时消息（中断/恢复等），非 LLM 协议合法角色，
// 网关会拒绝，不参与 LLM 上下文，在此统一过滤（中断恢复走 GetMessage 单独查询，不受影响）
func ToAgenticMessages(msgs []*Message) []adk.AgenticMessage {
	result := make([]adk.AgenticMessage, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == RoleInternal {
			continue
		}
		result = append(result, m.ToAgenticMessage())
	}
	return result
}

// FromAgenticMessages 将 AgenticMessage 切片转换为 Message 切片
func FromAgenticMessages(msgs []adk.AgenticMessage) []*Message {
	result := make([]*Message, len(msgs))
	for i, m := range msgs {
		result[i] = FromAgenticMessage(m)
	}
	return result
}
