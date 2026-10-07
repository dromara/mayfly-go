package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/internal/flow/imsg"
	"sync"
	"time"

	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/collx"
)

/******************* 执行流上下文 *******************/

type ExecutionCtx struct {
	parent context.Context

	Procinst  *entity.Procinst  // 流程实例
	Execution *entity.Execution // 当前执行流

	ProcinsVars   collx.M // 流程实例变量
	ExecutionVars collx.M // 执行流变量
	OpExtra       collx.M // 操作额外信息，记录用户任务审批状态等

	HisProcinstOp *entity.HisProcinstOp // 当前节点操作记录，用于start这种立即完成的避免在开始节点时重复查询节点信息

	flowDef *entity.FlowDef
}

// GetProcinst 获取流程实例，若上下文不存在则从库中获取
func (e *ExecutionCtx) GetProcinst() *entity.Procinst {
	if e.Procinst == nil {
		pi, err := GetProcinstApp().GetById(e.Execution.ProcinstId)
		if err != nil {
			panic(err)
		}
		e.Procinst = pi
	}
	return e.Procinst
}

// GetFlowDef 获取流程定义
func (ec *ExecutionCtx) GetFlowDef() *entity.FlowDef {
	if ec.flowDef == nil {
		ec.flowDef = ec.Procinst.GetFlowDef()
	}
	return ec.flowDef

}

// GetNode 获取当前节点信息
func (ec *ExecutionCtx) GetFlowNode() *entity.FlowNode {
	nodes := ec.GetFlowDef().GetNodes(ec.Execution.NodeKey)
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

// GetNextNode 按连线跳转条件获取该执行流的下一个节点。
//
// 条件求值复用触发引擎：连线条件与触发策略是同一种条件树、同一套操作符，
// 只是字段字典换成流程实例变量，因此加一类可比较的流程变量不必改动这里
func (e *ExecutionCtx) GetNextNode(vars collx.M) (*entity.FlowNode, error) {
	procinst := e.GetProcinst()
	tc := newFlowConditionContext(e.parent, vars, procinst.ProcdefId)

	nextNodes, err := procinst.GetFlowDef().GetNextNodes(e.Execution.NodeKey, func(edge *entity.FlowEdge) (bool, error) {
		condition, err := edge.Condition()
		if err != nil {
			return false, flowConditionError(e.parent, flowEdgeLabel(edge), err)
		}
		matched, err := trigger.MatchCondition(condition, tc)
		if err != nil {
			return false, flowConditionError(e.parent, flowEdgeLabel(edge), err)
		}
		return matched, nil
	})
	if err != nil {
		return nil, err
	}
	if len(nextNodes) == 0 {
		return nil, nil
	}
	if len(nextNodes) > 1 {
		return nil, errorx.NewBiz("执行流的下一节点只允许单个节点")
	}
	return nextNodes[0], nil
}

// flowConditionError 把条件问题定位到具体连线。
//
// 判不出来时必须中断流转，而不是「按不成立处理」继续走默认分支：
// 条件写坏时最危险的后果不是报错，而是不声不响地跳过本该审批的那一步
func flowConditionError(ctx context.Context, source string, err error) error {
	return errorx.NewBizI(ctx, imsg.ErrFlowConditionInvalid, "source", source, "reason", errorReason(ctx, err))
}

// flowEdgeLabel 优先展示连线名称，未命名时退回节点 key，保证定位得到
func flowEdgeLabel(edge *entity.FlowEdge) string {
	if edge.Name != "" {
		return edge.Name
	}
	return fmt.Sprintf("%s->%s", edge.SourceNodeKey, edge.TargetNodeKey)
}

/*  context.Context 实现方法  */

// 实现Deadline方法（继承父上下文）
func (ec *ExecutionCtx) Deadline() (deadline time.Time, ok bool) {
	return ec.parent.Deadline()
}

// 实现Done方法（继承父上下文）
func (ec *ExecutionCtx) Done() <-chan struct{} {
	return ec.parent.Done()
}

// 实现Err方法（继承父上下文）
func (ec *ExecutionCtx) Err() error {
	return ec.parent.Err()
}

// 实现Value方法
func (ec *ExecutionCtx) Value(key interface{}) interface{} {
	return ec.parent.Value(key)
}

func NewExecutionCtx(ctx context.Context, procinst *entity.Procinst, execution *entity.Execution) *ExecutionCtx {
	return &ExecutionCtx{
		parent:    ctx,
		Procinst:  procinst,
		Execution: execution,

		ExecutionVars: collx.M(execution.Vars),
		ProcinsVars:   collx.M(procinst.Vars),
	}
}

/******************* 节点定义 *******************/

// NodeBehavior node handler
type NodeBehavior interface {
	// GetType 获取节点类型
	GetType() entity.FlowNodeType

	// Validate 验证节点信息
	Validate(context.Context, *entity.FlowDef, *entity.FlowNode) error

	// Execute 执行节点
	Execute(*ExecutionCtx) error

	// Leave 离开节点
	Leave(*ExecutionCtx) error

	// IsAsync 是否异步节点
	IsAsync() bool
}

type DefaultNodeBehavior struct {
}

func (h *DefaultNodeBehavior) Validate(ctx context.Context, flowDef *entity.FlowDef, node *entity.FlowNode) error {
	return nil
}

func (h *DefaultNodeBehavior) Execute(ctx *ExecutionCtx) error {
	return h.Leave(ctx)
}

func (h *DefaultNodeBehavior) Leave(ctx *ExecutionCtx) error {
	// 默认执行流推进下一节点
	return GetExecutionApp().ContinueExecution(ctx)
}

func (h *DefaultNodeBehavior) IsAsync() bool {
	return false
}

/******************* 节点注册器 *******************/

type NodeBehaviorRegistry struct {
	nodes sync.Map
}

func (r *NodeBehaviorRegistry) Register(node NodeBehavior) {
	if _, loaded := r.nodes.LoadOrStore(node.GetType(), node); loaded {
		panic(fmt.Sprintf("handler already registered: %s", node.GetType()))
	}
}

func (r *NodeBehaviorRegistry) GetNode(nodeType entity.FlowNodeType) (NodeBehavior, error) {
	val, ok := r.nodes.Load(nodeType)
	if !ok {
		return nil, fmt.Errorf("node handler not found: %s", nodeType)
	}
	return val.(NodeBehavior), nil
}

var nodeBehaviorRegistry = &NodeBehaviorRegistry{}
