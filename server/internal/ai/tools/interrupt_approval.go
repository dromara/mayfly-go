package tools

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type ApprovalInfo struct {
	BaseInterruptInfo
}

var _ InterruptMetadata = (*ApprovalInfo)(nil)

func NewArrpovalInfo(ctx context.Context, toolInfo *schema.ToolInfo, arguments string) *ApprovalInfo {
	ti := &ToolInfo{
		Name: toolInfo.Name,
		Desc: toolInfo.Desc,
	}
	toolJsonSchema, err := toolInfo.ParamsOneOf.ToJSONSchema()
	if err != nil {
		ti.JsonSchema = jsonx.ToStr(toolJsonSchema)
	}

	ai := &ApprovalInfo{
		BaseInterruptInfo: BaseInterruptInfo{
			Type:        InterruptTypeApproval,
			ToolInfo:    ti,
			ToolCallId:  compose.GetToolCallID(ctx),
			Arguments:   arguments,
			Description: i18n.T(imsg.ApprovalDesc),
			Title:       i18n.T(imsg.ApprovalTitle),
		}}
	return ai
}

func InterruptOrResumeApproval(ctx context.Context, toolDesc string, arguments any, reason string) error {
	isApprovalResume, err := ResumeApproval(ctx, toolDesc)
	if !isApprovalResume {
		return InterruptApproval(ctx, toolDesc, arguments, reason)
	}
	if err == nil {
		return nil
	}
	return NewToolError(err, RecoverRetry)
}

func InterruptApproval(ctx context.Context, toolDesc string, arguments any, reason string) error {
	argumentsInJSON := jsonx.ToStr(arguments)
	ai := &ApprovalInfo{
		BaseInterruptInfo: BaseInterruptInfo{
			Type:        InterruptTypeApproval,
			ToolCallId:  compose.GetToolCallID(ctx),
			Arguments:   argumentsInJSON,
			Description: reason,
			ToolInfo:    &ToolInfo{Name: toolDesc},
			Title:       i18n.T(imsg.ApprovalTitle),
		}}
	return tool.StatefulInterrupt(ctx, ai, argumentsInJSON)
}

func ResumeApproval(ctx context.Context, toolDesc string) (bool, error) {
	// 首先检查是否已审批过
	messages, err := session.DefaultSessionStore.GetMessage(ctx, &session.MessageQuery{MessageType: string(InterruptTypeApproval), ToolCallId: compose.GetToolCallID(ctx)})
	if err != nil {
		logx.DebugfContext(ctx, "query approval messages: %v", err)
	}
	if len(messages) > 0 {
		for _, msg := range messages {
			var resumeInfo ApprovalResume
			if err := msg.Extra.Unmarshal("resumeInfo", &resumeInfo); err != nil {
				continue
			}
			return true, handleApprovalResult(ctx, toolDesc, &resumeInfo)
		}
	}

	wasInterrupted, _, _ := tool.GetInterruptState[string](ctx)
	if !wasInterrupted {
		return false, nil
	}

	isResumeTarget, hasData, data := tool.GetResumeContext[*ApprovalResume](ctx)
	if !isResumeTarget || !hasData {
		return false, nil
	}

	AppendResumeInfo(ctx, data.InterruptId, data)
	return true, handleApprovalResult(ctx, toolDesc, data)
}

func handleApprovalResult(ctx context.Context, toolDesc string, data *ApprovalResume) error {
	if data.Action == "approve" {
		return nil
	}

	if data.Action == "reject" {
		reason := cmp.Or(data.Payload.GetStr("reason"), i18n.T(imsg.RejectReasonDefault))
		// 构建更清晰的拒绝消息
		msg := fmt.Sprintf(
			"[OPERATION_REJECTED] The tool '%s' was explicitly rejected by the user.\nReason: %s\nPlease do not retry this action automatically. Ask the user for further instructions if needed.",
			toolDesc,
			reason,
		)
		return errors.New(msg)
	}

	return fmt.Errorf("[OPERATION_CANCELLED] The tool '%s' execution was cancelled due to invalid action: %s", toolDesc, data.Action)
}

// approvalExtension 审批中断的扩展实现（经中断扩展贡献者统一装配，见 agent/ext/interrupt）
type approvalExtension struct{}

var _ InterruptExtension = (*approvalExtension)(nil)

func (e *approvalExtension) ConvertResume(resume *InterruptResume) any {
	return &ApprovalResume{InterruptResume: resume}
}

func (e *approvalExtension) PrepareResumeCtx(ctx context.Context, resume *InterruptResume) context.Context {
	return ctx // 审批类型无需额外预处理
}

func (e *approvalExtension) ExtraEventMetadata(info InterruptMetadata) collx.M {
	return nil // 审批类型无特有 metadata
}

// NewApprovalExtension 创建审批中断扩展（经中断扩展贡献者在装配期装载）
func NewApprovalExtension() InterruptExtension { return &approvalExtension{} }
