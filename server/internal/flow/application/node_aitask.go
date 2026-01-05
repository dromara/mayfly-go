package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/ai/agent"
	"mayfly-go/internal/ai/prompt"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/flow/infra/persistence"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/jsonx"
	"time"

	"github.com/spf13/cast"
)

/******************* AI任务节点 *******************/

const (
	FlowNodeTypeAiTask entity.FlowNodeType = "aitask"
)

// AiTaskNode AI任务节点
type AiTaskNode struct {
	entity.FlowNode

	AuditRule string `json:"auditRule" form:"auditRule"` // 审批规则
}

// ToUserTaskNode 将标准节点转换成用户任务节点(方便取值)
func ToAiTaskNode(node *entity.FlowNode) *AiTaskNode {
	return &AiTaskNode{
		FlowNode:  *node,
		AuditRule: node.GetExtraString("auditRule"),
	}
}

type FlowNodeAiTaskApprovalMode string

// AiTaskNodeBehavior Ai任务节点行为处理器
type AiTaskNodeBehavior struct {
	DefaultNodeBehavior
}

var _ NodeBehavior = (*AiTaskNodeBehavior)(nil)

func (h *AiTaskNodeBehavior) GetType() entity.FlowNodeType {
	return FlowNodeTypeAiTask
}

func (h *AiTaskNodeBehavior) Validate(ctx context.Context, flowDef *entity.FlowDef, node *entity.FlowNode) error {
	aitaskNode := ToAiTaskNode(node)
	if aitaskNode.AuditRule == "" {
		return errorx.NewBizI(ctx, imsg.ErrAiTaskNodeAuditRuleNotEmpty, "name", node.Name)
	}

	return nil
}

func (h *AiTaskNodeBehavior) IsAsync() bool {
	return true
}

func (u *AiTaskNodeBehavior) Execute(ctx *ExecutionCtx) error {
	ctx.parent = context.Background() // 该节点为异步操作，需重新赋值父上下文

	flowNode := ctx.GetFlowNode()
	aitaskNode := ToAiTaskNode(flowNode)

	aiagent, err := agent.NewAiAgent(ctx, agent.ToolTypeDb)
	if err != nil {
		return err
	}

	auditRule := aitaskNode.AuditRule
	sysPrompt := prompt.GetPrompt(prompt.FLOW_BIZ_AUDIT, auditRule)

	procinst := ctx.Procinst
	now := time.Now()
	procinstTask := &entity.ProcinstTask{
		ProcinstId:  procinst.Id,
		ExecutionId: ctx.Execution.Id,
		NodeKey:     flowNode.Key,
		NodeName:    flowNode.Name,
		NodeType:    flowNode.Type,
	}
	procinstTask.CreateTime = &now

	allowExecute := false
	suggestion := ""

	cancelCtx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelFunc()
	res, err := aiagent.GetChatMsg(cancelCtx, sysPrompt, jsonx.ToStr(procinst.BizForm))
	if err != nil {
		suggestion = fmt.Sprintf("AI agent response failed: %v", err)
		logx.Error(suggestion)
	} else {
		resJson, err := jsonx.ToMap(res)
		if err != nil {
			suggestion = fmt.Sprintf("AI agent response parsing to JSON failed: %v, response: %s", err, res)
			logx.Error(suggestion)
		} else {
			allowExecute = cast.ToBool(resJson["allowExecute"])
			suggestion = cast.ToString(resJson["suggestion"])
		}
	}

	procinstTask.Remark = suggestion
	procinstTask.SetEnd()

	procinstApp := GetProcinstApp()
	executionApp := GetExecutionApp()
	procinstTaskApp := GetProcinstTaskApp()

	return procinstTaskApp.Tx(ctx, func(c context.Context) error {
		if !allowExecute {
			// 流程实例退回
			procinst.Status = entity.ProcinstStatusBack
			ctx.OpExtra.Set("approvalResult", entity.ProcinstTaskStatusBack)
			procinstTask.Status = entity.ProcinstTaskStatusBack

			if err := procinstApp.Save(c, procinst); err != nil {
				return err
			}
		} else {
			ctx.OpExtra.Set("approvalResult", entity.ProcinstTaskStatusCompleted)
			procinstTask.Status = entity.ProcinstTaskStatusCompleted
		}

		// 保存任务与任务候选者信息，兼容usertask展示
		if err := procinstTaskApp.Save(c, procinstTask); err != nil {
			return err
		}
		handler := "AI"
		procinstTaskCandidate := &entity.ProcinstTaskCandidate{
			TaskId:     procinstTask.Id,
			ProcinstId: procinst.Id,
			Candidate:  handler,
			Handler:    &handler,
			Status:     procinstTask.Status,
		}
		procinstTaskCandidate.CreateTime = &now
		procinstTaskCandidate.SetEnd()
		if err := persistence.GetProcinstTaskCandidateRepo().Save(c, procinstTaskCandidate); err != nil {
			return err
		}

		if !allowExecute {
			// 跳转至开始节点，重新修改提交
			ctx.Execution.State = entity.ExectionStateSuspended // 执行流挂起
			return executionApp.MoveTo(ctx, ctx.GetFlowDef().GetNodeByType(FlowNodeTypeStart)[0])

		}
		return u.Leave(ctx)
	})

}
