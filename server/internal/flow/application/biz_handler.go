package application

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
)

// 流程业务处理函数（流程结束后会根据流程业务类型获取该函数进行处理）
// @param procinstStatus 流程实例状态
// @param bizKey 业务key，可为业务数据对应的主键
// type FlowBizHandlerFunc func(ctx context.Context, procinstStatus entity.ProcinstStatus, bizKey string) error

// 业务流程处理器（流程状态变更后会根据流程业务类型获取对应的处理器进行回调处理）
type FlowBizHandler interface {

	// 业务流程处理函数
	// @param procinstStatus 流程实例状态
	// @param bizKey 业务key，可为业务数据对应的主键
	FlowBizHandle(ctx context.Context, procinstStatus entity.ProcinstStatus, bizKey string) error
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
func FlowBizHandle(ctx context.Context, flowBizType string, bizKey string, procinstStatus entity.ProcinstStatus) error {
	if handler, ok := handlers[flowBizType]; !ok {
		logx.Warnf("flow biz handler not found: bizType=%s", flowBizType)
		return errorx.NewBiz("业务流程处理器不存在")
	} else {
		return handler.FlowBizHandle(ctx, procinstStatus, bizKey)
	}
}
