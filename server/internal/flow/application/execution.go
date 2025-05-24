package application

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

type Execution interface {
	base.App[*entity.Execution]

	Init()

	// CreateExecution 创建执行流(初始化)
	CreateExecution(ctx context.Context, procinst *entity.Procinst) error

	// ContinueExecution 推进执行流继续执行后续节点
	ContinueExecution(ctx *ExecutionCtx) error

	// MoveTo 移动执行流到指定节点
	MoveTo(ctx *ExecutionCtx, node *entity.FlowNode) error

	CreateChildExecution(context.Context, *entity.Execution) (*entity.Execution, error)

	// CancelExecution 取消流程实例的所有执行流
	CancelExecution(ctx context.Context, procinstId uint64) error
}

type executionAppImpl struct {
	base.AppImpl[*entity.Execution, repository.Execution]

	hisProcinstOpApp HisProcinstOp `inject:"T"`
}

var _ (Execution) = (*executionAppImpl)(nil)

func (e *executionAppImpl) Init() {
	nodeBehaviorRegistry.Register(&StartNodeBehavior{})
	nodeBehaviorRegistry.Register(&EndNodeBehavior{})
	nodeBehaviorRegistry.Register(&UserTaskNodeBehavior{})

	const subId = "ExecutionApp"

	flowEventBus.Subscribe(EventTopicFlowProcinstCreate, subId, func(ctx context.Context, event *eventbus.Event[any]) error {
		procinst := event.Val.(*entity.Procinst)
		return e.CreateExecution(ctx, procinst)
	})

	flowEventBus.Subscribe(EventTopicFlowProcinstCancel, subId, func(ctx context.Context, event *eventbus.Event[any]) error {
		procinstId := event.Val.(uint64)
		return e.CancelExecution(ctx, procinstId)
	})

}

func (e *executionAppImpl) CreateExecution(ctx context.Context, procinst *entity.Procinst) error {
	flowDef := procinst.GetFlowDef()
	startNodes := flowDef.GetNodeByType(FlowNodeTypeStart)
	if len(startNodes) == 0 || len(startNodes) > 1 {
		return errorx.NewBiz("start node not found or more than one")
	}
	startNode := startNodes[0]

	// 创建执行流
	execution := &entity.Execution{
		ProcinstId:   procinst.Id,
		ParentId:     0,
		NodeKey:      startNode.Key,
		NodeName:     startNode.Name,
		NodeType:     startNode.Type,
		State:        entity.ExectionStateActive,
		IsConcurrent: -1,
	}

	return e.Tx(ctx, func(ctx context.Context) error {
		if err := e.Save(ctx, execution); err != nil {
			return err
		}
		return e.executeNode(NewExecutionCtx(ctx, procinst, execution))
	})
}

func (e *executionAppImpl) ContinueExecution(ctx *ExecutionCtx) error {
	// 合并流程实例与执行流变量，执行流变量优先级高
	vars := collx.MapMerge(ctx.ProcinsVars, ctx.ExecutionVars)

	nextNode, err := ctx.GetNextNode(vars)
	if err != nil {
		return err
	}

	return e.MoveTo(ctx, nextNode)
}

func (e *executionAppImpl) MoveTo(ctx *ExecutionCtx, nextNode *entity.FlowNode) error {
	execution := ctx.Execution
	if execution == nil {
		return errorx.NewBiz("execution is nil")
	}

	// 记录当前节点结束
	if err := e.hisProcinstOpApp.RecordEnd(ctx, "copmpleted"); err != nil {
		return err
	}

	// 下一个节点为空，说明流程已结束
	if nextNode == nil {
		return nil
	}

	// 执行流推进下一节点，并执行节点逻辑
	execution.NodeKey = nextNode.Key
	execution.NodeName = nextNode.Name
	execution.NodeType = nextNode.Type

	return e.Tx(ctx, func(c context.Context) error {
		// 更新当前执行流的最新节点信息
		if err := e.Save(c, execution); err != nil {
			return err
		}

		return e.executeNode(ctx)
	})
}

func (e *executionAppImpl) CreateChildExecution(ctx context.Context, parentExection *entity.Execution) (*entity.Execution, error) {
	execution := &entity.Execution{
		ProcinstId: parentExection.ProcinstId,
		ParentId:   parentExection.Id,
	}

	return execution, e.Save(ctx, execution)
}

func (e *executionAppImpl) CancelExecution(ctx context.Context, procinstId uint64) error {
	executions, err := e.ListByCond(model.NewCond().
		Eq("procinst_id", procinstId).
		In("state", collx.AsArray(entity.ExectionStateActive, entity.ExectionStateSuspended)))

	if err != nil {
		return err
	}

	return e.Tx(ctx, func(ctx context.Context) error {
		for _, execution := range executions {
			execution.State = entity.ExectionStateTerminated
			if err := e.UpdateById(ctx, execution); err != nil {
				return err
			}
		}
		return nil
	})
}

func (e *executionAppImpl) executeNode(ctx *ExecutionCtx) error {
	flowNode := ctx.GetFlowNode()
	node, err := nodeBehaviorRegistry.GetNode(flowNode.Type)
	if err != nil {
		return err
	}

	// 节点开始操作记录
	if err := e.hisProcinstOpApp.RecordStart(ctx); err != nil {
		return err
	}

	// 执行节点逻辑
	return node.Execute(ctx)
}
