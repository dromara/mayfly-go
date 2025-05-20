package application

import (
	"context"
	"errors"
	"mayfly-go/internal/flow/domain/entity"
)

/******************* 开始节点处理器 *******************/

const FlowNodeTypeStart entity.FlowNodeType = "start" // 开始节点

// StartNodeBehavior 开始节点处理器
type StartNodeBehavior struct {
	DefaultNodeBehavior
}

var _ NodeBehavior = (*StartNodeBehavior)(nil)

func (h *StartNodeBehavior) GetType() entity.FlowNodeType {
	return FlowNodeTypeStart
}

func (h *StartNodeBehavior) Validate(ctx context.Context, flowDef *entity.FlowDef, node *entity.FlowNode) error {
	// 检查开始节点是否有一个连线指出去
	edgesFromStartNode := flowDef.GetEdgeBySourceNode(node.Key)
	if len(edgesFromStartNode) == 0 {
		return errors.New("start node not has outgoing edge")
	}
	return nil
}
