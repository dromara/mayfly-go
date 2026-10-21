package application

import (
	"context"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/internal/ai/pkg/utils"
	"mayfly-go/internal/ai/prompt"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/flow/infra/persistence"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"time"
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

	auditRule := aitaskNode.AuditRule
	sysPrompt, err := prompt.GetPrompt("flow/biz_audit.md", collx.Kvs("rule", auditRule))
	if err != nil {
		return err
	}

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

	// 复用泛化无头运行门面（经中立契约包 headless 调用，破除 flow → ai/application 循环依赖）：
	// 审计档（只读）以工单发起人身份运行，无痕模式（Persist=false，保留既有不留会话行为）；
	// 消除与 WS 对话并行的双无头路径
	res, runErr := headless.Run(ctx.parent, &headless.RunReq{
		ProfileId:      headless.ProfileAuditor,
		Instruction:    sysPrompt,
		Input:          jsonx.ToStr(procinst.BizForm),
		OwnerAccountId: procinst.CreatorId,
		OwnerUsername:  procinst.Creator,
		Timeout:        1 * time.Minute,
		RunKind:        headless.RunKindAudit,
	})
	if runErr != nil {
		suggestion = i18n.T(imsg.ErrAiTaskResponseFailed, "err", runErr)
		logx.Error(suggestion)
	} else {
		resJson, err := utils.ParseLLMJSON2Map(res.Output)
		if err != nil {
			suggestion = i18n.T(imsg.ErrAiTaskResponseInvalid, "err", err)
			logx.Error(suggestion)
		} else {
			allowExecute = resJson.GetBool("allowExecute")
			suggestion = resJson.GetStr("suggestion")
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
