package application

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
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

var _ Execution = (*executionAppImpl)(nil)

var _ (Execution) = (*executionAppImpl)(nil)

func (e *executionAppImpl) Init() {
	nodeBehaviorRegistry.Register(&StartNodeBehavior{})
	nodeBehaviorRegistry.Register(&EndNodeBehavior{})
	nodeBehaviorRegistry.Register(&UserTaskNodeBehavior{})
	nodeBehaviorRegistry.Register(&AiTaskNodeBehavior{})

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
		return errorx.NewBizI(ctx, imsg.ErrStartNodeNotUnique)
	}
	startNode := startNodes[0]

	execution := &entity.Execution{
		ProcinstId: procinst.Id,
		ParentId:   0,
		NodeKey:    startNode.Key,
		NodeName:   startNode.Name,
		NodeType:   startNode.Type,
		State:      entity.ExectionStateSuspended,
	}
	if err := e.GetByCond(execution); err == nil {
		// 已存在挂起的流程实例，则直接激活继续执行
		execution.State = entity.ExectionStateActive
	} else {
		execution.State = entity.ExectionStateActive
		execution.IsConcurrent = -1
	}

	return e.Tx(ctx, func(ctx context.Context) error {
		if err := e.Save(ctx, execution); err != nil {
			return err
		}
		return e.executeNode(NewExecutionCtx(ctx, procinst, execution))
	})
}

func (e *executionAppImpl) ContinueExecution(ctx *ExecutionCtx) error {
	// 合并流程实例与执行流变量，执行流变量优先级高；再叠加本次操作产生的事实（如审批结果）：
	// 连线条件最常用的就是「通过走这条、驳回走那条」，而这些事实只存在于操作上下文里，
	// 不并进来后续判定就只能看到流程级的变量
	vars := collx.MapMerge(ctx.ProcinsVars, ctx.ExecutionVars, ctx.OpExtra)

	nextNode, err := ctx.GetNextNode(vars)
	if err != nil {
		return err
	}

	return e.MoveTo(ctx, nextNode)
}

// hisProcinstOpEndAction 节点结束操作留痕的固定动作值（写入 t_flow_his_procinst_op）
const hisProcinstOpEndAction = "completed"

func (e *executionAppImpl) MoveTo(ctx *ExecutionCtx, nextNode *entity.FlowNode) error {
	execution := ctx.Execution
	if execution == nil {
		return errorx.NewBizI(ctx.parent, imsg.ErrExecutionNil)
	}

	// 记录当前节点结束
	if err := e.hisProcinstOpApp.RecordEnd(ctx, hisProcinstOpEndAction); err != nil {
		return err
	}

	// 下一个节点为空：只有「当前就是结束节点」才是正常收尾。
	// 其它节点走到这里说明连线条件全不命中（或连线缺失），静默返回会让实例停在 Active、
	// 执行流停在旧节点、界面无待办且无任何日志——必须报错让审批人看到
	if nextNode == nil {
		if current := ctx.GetFlowNode(); current != nil && current.Type != FlowNodeTypeEnd {
			return errorx.NewBizI(ctx, imsg.ErrExecutionNoNextNode, "node", current.Name)
		}
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

		if execution.State == entity.ExectionStateActive {
			// 继续推进执行节点
			return e.executeNode(ctx)
		}

		return nil
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
	if node.IsAsync() {
		gox.Go(func() {
			if err := node.Execute(ctx); err != nil {
				logx.Errorf("async execute node error: %v, procinst_id: %d, node_key: %s", err, ctx.Procinst.Id, flowNode.Key)
			}
		})
		return nil
	}

	return node.Execute(ctx)
}
