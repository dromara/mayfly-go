package application

import (
	"context"
	"mayfly-go/internal/event"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/flow/infrastructure/persistence"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/global"
	"strings"

	"github.com/may-fly/cast"
)

/******************* 用户任务节点 *******************/

const (
	FlowNodeTypeUserTask entity.FlowNodeType = "usertask" // 用户任务

	NrOfCompleted = "nrOfCompleted" // number of completed
	NrOfAll       = "nrOfAll"       // 总候选人处理数
)

// UserTaskNode 用户任务节点
type UserTaskNode struct {
	entity.FlowNode

	CompletionCondition string   `json:"completionCondition" form:"completionCondition"` // 完成条件，如会签{{.nrOfAll == .nrOfCompleted}}
	Candidates          []string `json:"candidates" form:"candidates"`                   // 节点处理候选人
}

// ToUserTaskNode 将标准节点转换成用户任务节点(方便取值)
func ToUserTaskNode(node *entity.FlowNode) *UserTaskNode {
	return &UserTaskNode{
		FlowNode:            *node,
		CompletionCondition: node.GetExtraString("completionCondition"),
		Candidates:          node.GetExtraStringSlice("candidates"),
	}
}

type FlowNodeUserTaskApprovalMode string

// UserTaskNodeBehavior 用户任务节点行为处理器
type UserTaskNodeBehavior struct {
	DefaultNodeBehavior
}

var _ NodeBehavior = (*UserTaskNodeBehavior)(nil)

func (h *UserTaskNodeBehavior) GetType() entity.FlowNodeType {
	return FlowNodeTypeUserTask
}

func (h *UserTaskNodeBehavior) Validate(ctx context.Context, flowDef *entity.FlowDef, node *entity.FlowNode) error {
	usertaskNode := ToUserTaskNode(node)
	if len(usertaskNode.Candidates) == 0 {
		return errorx.NewBizI(ctx, imsg.ErrUserTaskNodeCandidateNotEmpty, "name", node.Name)
	}

	return nil
}

func (u *UserTaskNodeBehavior) Execute(ctx *ExecutionCtx) error {
	flowNode := ctx.GetFlowNode()
	usertaskNode := ToUserTaskNode(flowNode)

	candidates := usertaskNode.Candidates
	if len(candidates) == 0 {
		return errorx.NewBiz("candidates cannot be empty")
	}

	taskApp := GetProcinstTaskApp()
	task := &entity.ProcinstTask{
		ProcinstId:  ctx.GetProcinst().Id,
		ExecutionId: ctx.Execution.Id,
		NodeKey:     flowNode.Key,
		NodeName:    flowNode.Name,
		NodeType:    flowNode.Type,
		Status:      entity.ProcinstTaskStatusProcess,
	}

	// 赋值总的候选人审批数量，会签时需要该值
	task.Vars.Set(NrOfAll, len(candidates))
	procinst := ctx.GetProcinst()
	procinstId := procinst.Id

	// 创建审批任务与审批候选人
	return taskApp.Tx(ctx, func(c context.Context) error {
		if err := taskApp.Save(c, task); err != nil {
			return err
		}

		taskCandidates := make([]*entity.ProcinstTaskCandidate, 0, len(usertaskNode.Candidates))
		for _, candidate := range usertaskNode.Candidates {
			taskCandidates = append(taskCandidates, &entity.ProcinstTaskCandidate{
				ProcinstId: procinstId,
				TaskId:     task.Id,
				Candidate:  candidate,
				Status:     entity.ProcinstTaskStatusProcess,
			})

			// 用户账号类型
			if !strings.Contains(candidate, ":") {
				// 发送通知消息
				global.EventBus.Publish(ctx, event.EventTopicBizMsgTmplSend, msgdto.BizMsgTmplSend{
					BizType: FlowTaskNotifyBizKey,
					BizId:   procinst.ProcdefId,
					Params: map[string]any{
						"creator":        procinst.Creator,
						"procdefName":    procinst.ProcdefName,
						"bizKey":         procinst.BizKey,
						"taskName":       flowNode.Name,
						"procinstRemark": procinst.Remark,
					},
					ReceiverIds: []uint64{cast.ToUint64(candidate)},
				})
			}
		}

		return persistence.GetProcinstTaskCandidateRepo().BatchInsert(c, taskCandidates)
	})
}
