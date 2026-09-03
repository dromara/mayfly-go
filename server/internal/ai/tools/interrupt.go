package tools

import (
	"context"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
)

// InterruptType 定义中断类型
type InterruptType string

const (
	InterruptTypeApproval        InterruptType = "interrupt_approval"         // 人工审批
	InterruptTypeParamCompletion InterruptType = "interrupt_param_completion" // 参数补全
)

type ToolInfo struct {
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	JsonSchema string `json:"jsonSchema"`
}

type InterruptState struct {
	Name       string `json:"name"`
	ToolCallId string `json:"toolCallId"`
	Args       string `json:"args"`
}

// InterruptMetadata 定义中断元数据接口
// 任何需要触发人机交互中断的信息都应实现此接口
type InterruptMetadata interface {
	// GetType 返回中断类型，前端根据此类型渲染不同组件
	GetType() InterruptType

	// GetTitle 返回标题
	GetTitle() string

	// GetDescription 返回详细描述
	GetDescription() string

	// GetPayload 返回中断业务负载数据
	GetPayload() any

	// GetToolInfo 获取工具信息
	GetToolInfo() *ToolInfo

	// GetToolCallId 获取工具调用ID
	GetToolCallId() string

	// GetArgument 获取工具参数
	GetArgument() string
}

// BaseInterruptInfo 基础中断信息结构体，实现 InterruptMetadata 接口
// 具体中断类型应嵌入此结构体
type BaseInterruptInfo struct {
	Type        InterruptType `json:"type"`              // 中断类型
	Title       string        `json:"title"`             // 中断标题
	Description string        `json:"description"`       // 中断描述
	Payload     any           `json:"payload,omitempty"` // 中断负载数据
	ToolCallId  string        `json:"toolCallId"`
	ToolInfo    *ToolInfo     `json:"toolInfo"`
	Arguments   string        `json:"arguments"` //原始参数字符串
}

var _ InterruptMetadata = (*BaseInterruptInfo)(nil)

// --- 实现 InterruptMetadata 接口 ---

func (b *BaseInterruptInfo) GetType() InterruptType {
	return b.Type
}

func (b *BaseInterruptInfo) GetTitle() string {
	return b.Title
}

func (b *BaseInterruptInfo) GetDescription() string {
	return b.Description
}

func (b *BaseInterruptInfo) GetPayload() any {
	return b.Payload
}

func (b *BaseInterruptInfo) GetToolInfo() *ToolInfo {
	return b.ToolInfo
}

func (b *BaseInterruptInfo) GetArgument() string {
	return b.Arguments
}

func (b *BaseInterruptInfo) GetToolCallId() string {
	return b.ToolCallId
}

// ApprovalResume 审批恢复参数
type ApprovalResume struct {
	*InterruptResume
}

// ParamCompletionResume 参数补全恢复参数
type ParamCompletionResume struct {
	*InterruptResume
}

// InterruptResume 中断恢复的信息
type InterruptResume struct {
	TurnId        string        `json:"turnId" binding:"required"`
	InterruptId   string        `json:"interruptId" binding:"required"`   // 中断id
	InterruptType InterruptType `json:"interruptType" binding:"required"` // 中断类型
	Action        string        `json:"action" binding:"required"`        // 操作
	Payload       collx.M       `json:"payload"`                          // 操作参数
}

// ToTarget 将 InterruptResume 转换为具体的恢复参数结构体
// （通过中断扩展注册表分发，新增中断类型无需修改此处，见 interrupt_registry.go）
func (i *InterruptResume) ToTarget() any {
	return ConvertInterruptResume(i)
}

func AppendResumeInfo(ctx context.Context, interruptId string, resumeInfo any) *session.Message {
	msgQuery := &session.MessageQuery{
		ActionId: interruptId,
	}

	msgs, err := session.DefaultSessionStore.GetMessage(ctx, msgQuery)
	if err != nil || len(msgs) == 0 {
		logx.InfofContext(ctx, "not found interrupt message: %v", err)
		return nil
	}
	msg := msgs[0]

	msg.Extra.Set("resumeInfo", resumeInfo)
	if err := session.DefaultSessionStore.UpdateMessage(ctx, msg); err != nil {
		logx.ErrorfContext(ctx, "update interrupt message resumeInfo failed: %v", err)
		return nil
	}
	return msg
}
