package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/flow/domain/entity"
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

// GetNextNodes 获取该执行流的下一个节点
func (e *ExecutionCtx) GetNextNode(vars collx.M) (*entity.FlowNode, error) {
	nextNodes := e.GetProcinst().GetFlowDef().GetNextNodes(e.Execution.NodeKey, vars)
	if len(nextNodes) == 0 {
		return nil, nil
	}
	if len(nextNodes) > 1 {
		return nil, errorx.NewBiz("执行流的下一节点只允许单个节点")
	}
	return nextNodes[0], nil
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
