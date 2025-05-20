package application

import (
	"context"
	"errors"
	"mayfly-go/internal/flow/domain/entity"
)

/******************* 结束节点 *******************/

const FlowNodeTypeEnd entity.FlowNodeType = "end" // 开始节点

// EndNodeBehavior 结束节点
type EndNodeBehavior struct {
	DefaultNodeBehavior
}

var _ NodeBehavior = (*EndNodeBehavior)(nil)

func (h *EndNodeBehavior) GetType() entity.FlowNodeType {
	return FlowNodeTypeEnd
}

func (h *EndNodeBehavior) Validate(ctx context.Context, flowDef *entity.FlowDef, node *entity.FlowNode) error {
	// 检查是否有连线指向结束节点
	hasIncomingEdge := false
	for _, edge := range flowDef.Edges {
		if edge.TargetNodeKey == node.Key {
			hasIncomingEdge = true
			break
		}
	}
	if !hasIncomingEdge {
		return errors.New("end node not has incoming edge")
	}

	return nil
}

func (h *EndNodeBehavior) Execute(ctx *ExecutionCtx) error {
	// 执行流状态变为已完成
	execution := ctx.Execution
	execution.State = entity.ExectionStateCompleted

	procinstApp := GetProcinstApp()
	exectuionApp := GetExecutionApp()

	if err := procinstApp.Tx(ctx, func(c context.Context) error {
		if ctx.GetProcinst().Status == entity.ProcinstStatusActive {
			if err := procinstApp.CompletedProc(c, ctx.Procinst.Id); err != nil {
				return err
			}
		}

		return exectuionApp.Save(c, execution)
	}); err != nil {
		return err
	}

	return h.Leave(ctx)
}
