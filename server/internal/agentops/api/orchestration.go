package api

import (
	"mayfly-go/internal/agentops/application"
	"mayfly-go/internal/agentops/imsg"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

// Orchestration 多 Agent 编排 API（计划阶段 4.1，Supervisor 模式）：
// 发起「Planner 拆解 → 步骤执行（写步骤转提案审批）→ Reviewer 审查」的编排流程，
// 立即返回父运行（trigger_ref=orchestration），编排轨迹在执行历史中以子运行呈现。
type Orchestration struct {
	orchestrator application.Orchestrator `inject:"T"`
	// accountApp 用于把表单指定的 ownerAccountId 解析为账号名（运行身份展示/审计）；
	// 未指定 owner 时以当前登录账号为运行身份（判定均在 resolveRunOwner 内）
	accountApp sysapp.Account `inject:"T"`
}

// OrchestrationForm 编排发起表单
type OrchestrationForm struct {
	// Goal 运维目标（必填）
	Goal string `json:"goal" binding:"required"`
	// Instruction 额外编排要求（可空）
	Instruction string `json:"instruction"`
	// OwnerAccountId 编排运行身份账号（可空，缺省当前登录账号；指定他人需超管或 owner-impersonate 权限点）
	OwnerAccountId uint64 `json:"ownerAccountId"`
	// TimeoutSeconds 单步运行超时秒数（0=用服务默认）
	TimeoutSeconds int `json:"timeoutSeconds"`
	// MaxStep 单步运行最大步数（0=用 Profile 默认）
	MaxStep int `json:"maxStep"`
	// TokenBudget 单步运行 token 预算熔断（0=不熔断）
	TokenBudget int64 `json:"tokenBudget"`
}

func (o *Orchestration) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 发起编排（异步执行，返回父运行 id/runNo 供前端跳转执行历史）
		req.NewPost("", o.Orchestrate).Log(req.NewLogSaveI(imsg.LogOrchestrationRun)).RequiredPermissionCode("agentops:orchestration:run"),
	}
	return req.NewConfs("/agentops/orchestrations", reqs[:]...)
}

// Orchestrate 发起一次多 Agent 编排
func (o *Orchestration) Orchestrate(rc *req.Ctx) {
	body := rc.BindJson[OrchestrationForm]()
	rc.ReqParam = body

	owner, err := resolveRunOwner(rc, o.accountApp, body.OwnerAccountId)
	biz.ErrIsNil(err)

	run, err := o.orchestrator.Orchestrate(rc.MetaCtx, &application.OrchestrateReq{
		Goal:           body.Goal,
		Instruction:    body.Instruction,
		OwnerAccountId: owner.AccountId,
		OwnerUsername:  owner.Username,
		TimeoutSeconds: body.TimeoutSeconds,
		MaxStep:        body.MaxStep,
		TokenBudget:    body.TokenBudget,
	})
	biz.ErrIsNil(err)
	rc.ResData = map[string]any{"runId": run.Id, "runNo": run.RunNo}
}

var _ req.RouterApi = (*Orchestration)(nil)
