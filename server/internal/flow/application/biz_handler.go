package application

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
)

type BizHandleParam struct {
	BizType        string                //业务类型
	BizKey         string                // 业务key
	BizForm        string                // 业务表单信息
	ProcinstStatus entity.ProcinstStatus // 业务状态
}

// 业务流程处理器（流程状态变更后会根据流程业务类型获取对应的处理器进行回调处理）
type FlowBizHandler interface {

	// 业务流程处理函数
	// @param bizHandleParam 业务处理信息，可获取实例状态、关联业务key等信息
	FlowBizHandle(ctx context.Context, bizHandleParam *BizHandleParam) error
}

var (
	handlers map[string]FlowBizHandler = make(map[string]FlowBizHandler, 0)
)

// 注册流程业务处理函数
func RegisterBizHandler(flowBizType string, handler FlowBizHandler) {
	logx.Infof("flow register biz handelr: bizType=%s", flowBizType)
	handlers[flowBizType] = handler
}

// 流程业务处理
func FlowBizHandle(ctx context.Context, bizHandleParam *BizHandleParam) error {
	flowBizType := bizHandleParam.BizType
	if handler, ok := handlers[flowBizType]; !ok {
		logx.Warnf("flow biz handler not found: bizType=%s", flowBizType)
		return errorx.NewBiz("业务处理器不存在")
	} else {
		return handler.FlowBizHandle(ctx, bizHandleParam)
	}
}
