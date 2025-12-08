package application

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
)

type BizHandleParam struct {
	Procinst entity.Procinst // 流程实例
}

// 业务流程处理器（流程状态变更后会根据流程业务类型获取对应的处理器进行回调处理）
type FlowBizHandler interface {

	// FlowBizHandle 业务流程处理函数
	//  bizHandleParam 业务处理信息，可获取实例状态、关联业务key等信息
	// return any 返回业务处理结果
	FlowBizHandle(ctx context.Context, bizHandleParam *BizHandleParam) (any, error)
}

var (
	handlers map[string]FlowBizHandler = make(map[string]FlowBizHandler, 0)
)

// RegisterBizHandler 注册流程业务处理函数
func RegisterBizHandler(flowBizType string, handler FlowBizHandler) {
	logx.Infof("flow register biz handelr: bizType=%s", flowBizType)
	handlers[flowBizType] = handler
}

// FlowBizHandle 流程业务处理
func FlowBizHandle(ctx context.Context, bizHandleParam *BizHandleParam) (any, error) {
	if bizHandler, err := GetFlowBizHandler(bizHandleParam); err != nil {
		return nil, err
	} else {
		return bizHandler.FlowBizHandle(ctx, bizHandleParam)
	}
}

// GetFlowBizHandler 获取流程业务处理函数
func GetFlowBizHandler(bizHandleParam *BizHandleParam) (FlowBizHandler, error) {
	flowBizType := bizHandleParam.Procinst.BizType
	if handler, ok := handlers[flowBizType]; !ok {
		logx.Warnf("flow biz handler not found: bizType=%s", flowBizType)
		return nil, errorx.NewBiz("flow biz handler not found")
	} else {
		return handler, nil
	}
}
