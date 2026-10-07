package application

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
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

// HasBizHandler 判断该业务场景是否有审批通过后的执行回调。
//
// 没有回调的场景意味着「需审批」无法落地（工单批了也没人把操作执行回去），
// 因此触发策略的可配置级别由它推导，而不是由各业务模块再声明一遍
func HasBizHandler(flowBizType string) bool {
	_, ok := handlers[flowBizType]
	return ok
}

// FlowBizHandle 流程业务处理。
//
// 身份统一在这里换成工单发起人，而不是由各业务模块自己换：回调一旦漏换就以审批人身份跑完整个操作
// （重放发生在最后一位审批人点通过的那一刻），新接入的业务场景极容易忘了这一步
func FlowBizHandle(ctx context.Context, bizHandleParam *BizHandleParam) (any, error) {
	bizHandler, err := GetFlowBizHandler(bizHandleParam)
	if err != nil {
		return nil, err
	}
	return bizHandler.FlowBizHandle(BizOperatorContext(ctx, bizHandleParam), bizHandleParam)
}

// BizOperatorContext 把审批回放的执行身份换成工单发起人，并保留原上下文里的请求链路信息。
//
// 审批人常常只有审批权、没有那台资源的生产运维权限：用审批人身份过资源鉴权会直接失败，
// 表现为「工单批过了但命令没执行」；即使他有权限，执行归属也会错记成审批人
func BizOperatorContext(ctx context.Context, bizHandleParam *BizHandleParam) context.Context {
	procinst := bizHandleParam.Procinst
	return contextx.WithLoginAccount(ctx, &model.LoginAccount{
		Id:       procinst.CreatorId,
		Username: procinst.Creator,
	})
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
