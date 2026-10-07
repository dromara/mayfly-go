package application

import (
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/pkg/logx"
	"sync"
)

// RegisterTriggerBiz 注册一个业务场景的触发策略能力（字段字典、内置检查项、试算解析）。
//
// 业务模块应在包级 init 中调用，使元数据早于任何 API 请求与保存校验就绪；
// 求值函数内部如需应用服务，通过 ioc.Get 惰性获取
func RegisterTriggerBiz(meta trigger.BizMeta) {
	trigger.RegisterBiz(meta)
}

// RegisterTriggerOp 注册场景专属的操作符
func RegisterTriggerOp(def trigger.OpDef) {
	trigger.RegisterOp(def)
}

// RegisterTriggerType 注册自定义字段类型及其默认操作符与前端编辑器
func RegisterTriggerType(def trigger.FieldTypeDef) {
	trigger.RegisterType(def)
}

var approvalChannelCheck sync.Once

// checkTriggerApprovalChannels 校验「声明了审批通道」与「注册了业务回调」两件事是否一致。
//
// 这个检查不能在 flow 的 Init 里跑：各模块的 RegisterBizHandler 与 flow 的 Init 一样被 starter
// 以 goroutine 并发执行，那时 db/redis 的回调往往还没注册上，合法配置也会被报成缺回调（实测每次启动
// 都会刷两条 ERROR）。改到第一次真实触发校验时执行一次：那时服务已在处理请求，所有模块早已就绪
func checkTriggerApprovalChannels() {
	approvalChannelCheck.Do(func() {
		for _, meta := range trigger.BizMetas() {
			checkGovernanceDeclaration(meta)
			if meta.ConditionOnly {
				// 流程内部条件的字段字典既不审批也不触发，没有一致性可言
				continue
			}
			hasHandler := HasBizHandler(meta.BizType)
			switch {
			case meta.Approvable && !hasHandler:
				logx.Errorf("flow trigger: bizType=%s declares an approval channel but no biz handler is registered, approved work orders will not be executed", meta.BizType)
			case !meta.Approvable && hasHandler:
				logx.Warnf("flow trigger: bizType=%s registers a biz handler but does not declare itself approvable, the required severity stays unavailable", meta.BizType)
			}
		}
	})
}

// checkGovernanceDeclaration 把治理路径的声明问题报进日志（判据见域层 GovernanceDeclarationProblems）。
//
// 与审批通道一致性检查同一时机执行：那时各模块的 init 早已跑完，注册表是完整的
func checkGovernanceDeclaration(meta trigger.BizMeta) {
	for _, problem := range trigger.GovernanceDeclarationProblems(meta) {
		logx.Errorf("flow trigger: bizType=%s %s", meta.BizType, problem)
	}
}
