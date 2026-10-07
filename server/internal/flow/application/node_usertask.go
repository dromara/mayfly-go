package application

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/flow/infra/persistence"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/spf13/cast"
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

	// CompletionCondition 节点完成条件：满足即认为该节点审批完成并推进流程。
	// 或签是「已完成人数 >= 1」，会签是「完成比例 >= 1」，过半通过是「完成比例 >= 0.5」，
	// 与连线跳转条件同一套条件树，字段字典换成会签计数
	CompletionCondition *entity.RuleNode
	Candidates          []string `json:"candidates" form:"candidates"` // 节点处理候选人
}

// ToUserTaskNode 将标准节点转换成用户任务节点(方便取值)。
//
// 完成条件解析失败必须返回 error：条件读不出来时无论按「未完成」还是「已完成」处理都会出错，
// 前者让节点永远卡住，后者等于跳过审批
func ToUserTaskNode(node *entity.FlowNode) (*UserTaskNode, error) {
	condition, err := entity.RuleNodeFromExtra(node.Extra, entity.FlowNodeCompletionConditionKey)
	if err != nil {
		return nil, err
	}
	return &UserTaskNode{
		FlowNode:            *node,
		CompletionCondition: condition,
		Candidates:          node.GetExtraStringSlice("candidates"),
	}, nil
}

// IsUserTaskComplete 判定用户任务节点在本次审批后是否已完成。
//
// 与连线跳转条件共用同一份字段字典与同一个求值上下文构造：会签计数、审批结果这些事实
// 只需要注册一次，两类判定就都能引用，不必各写一套取数
func IsUserTaskComplete(ctx context.Context, condition *entity.RuleNode, vars collx.M) (bool, error) {
	tc := newFlowConditionContext(ctx, vars, 0)
	matched, err := trigger.MatchCondition(condition, tc)
	if err != nil {
		return false, flowConditionError(ctx, "completionCondition", err)
	}
	// 引用了取不到的字段时条件按「不成立」处理，节点会停在待审批；不记一笔就没人知道停在哪
	for _, field := range tc.Unknown {
		logx.WarnfContext(ctx, "flow completion condition references a field that is unavailable: %s", field)
	}
	return matched, nil
}

// UserTaskNodeBehavior 用户任务节点行为处理器
type UserTaskNodeBehavior struct {
	DefaultNodeBehavior
}

var _ NodeBehavior = (*UserTaskNodeBehavior)(nil)

func (h *UserTaskNodeBehavior) GetType() entity.FlowNodeType {
	return FlowNodeTypeUserTask
}

func (h *UserTaskNodeBehavior) Validate(ctx context.Context, flowDef *entity.FlowDef, node *entity.FlowNode) error {
	usertaskNode, err := ToUserTaskNode(node)
	if err != nil {
		return flowConditionError(ctx, node.Name, err)
	}
	if len(usertaskNode.Candidates) == 0 {
		return errorx.NewBizI(ctx, imsg.ErrUserTaskNodeCandidateNotEmpty, "name", node.Name)
	}
	// 完成条件缺失会导致「第一个审批人就通过」与「永远无法完成」两种都无法预期的走向，
	// 因此保存流程时就要求配置，而不是留到运行时按默认语义猜
	if !trigger.HasCondition(usertaskNode.CompletionCondition) {
		return errorx.NewBizI(ctx, imsg.ErrUserTaskNodeCompletionRequired, "name", node.Name)
	}
	if err := trigger.ValidateCondition(FlowInstanceBizType, usertaskNode.CompletionCondition, trigger.WithSegmentResolver(ruleSegmentResolver(ctx))); err != nil {
		return flowConditionError(ctx, node.Name, err)
	}

	return nil
}

func (u *UserTaskNodeBehavior) Execute(ctx *ExecutionCtx) error {
	flowNode := ctx.GetFlowNode()
	usertaskNode, err := ToUserTaskNode(flowNode)
	if err != nil {
		return flowConditionError(ctx, flowNode.Name, err)
	}

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
				params := map[string]any{
					"creator":        procinst.Creator,
					"procdefName":    procinst.ProcdefName,
					"bizKey":         procinst.BizKey,
					"taskName":       flowNode.Name,
					"procinstRemark": procinst.Remark,
				}
				// 发送通知消息
				global.EventBus.Publish(context.Background(), event.EventTopicBizMsgTmplSend, &msgdto.BizMsgTmplSend{
					BizType:     FlowTaskNotifyBizKey,
					BizId:       procinst.ProcdefId,
					Params:      params,
					ReceiverIds: []uint64{cast.ToUint64(candidate)},
				})

				global.EventBus.Publish(context.Background(), event.EventTopicMsgTmplSend, &msgdto.MsgTmplSendEvent{
					TmplChannel: msgdto.MsgTmplFlowUserTaskTodo,
					Params:      params,
					ReceiverIds: []uint64{cast.ToUint64(candidate)},
				})
			}
		}

		return persistence.GetProcinstTaskCandidateRepo().BatchInsert(c, taskCandidates)
	})
}
