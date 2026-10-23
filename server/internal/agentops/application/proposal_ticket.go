package application

import (
	"context"
	"strconv"
	"strings"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/imsg"
	flowapp "mayfly-go/internal/flow/application"
	flowdto "mayfly-go/internal/flow/application/dto"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/jsonx"
)

func (g *proposalGatewayImpl) recordTicketResult(ctx context.Context, proposalId uint64) {
	proposal, err := g.GetById(proposalId)
	if err != nil || proposal == nil || proposal.ProcInstId == 0 {
		return
	}
	bizStatus := flowentity.ProcinstBizStatusSuccess
	if proposal.Status != entity.ProposalStatusExecuted {
		bizStatus = flowentity.ProcinstBizStatusFail
	}
	app := flowProcinstFn()
	if app == nil {
		// flow 应用未就绪（启动早期/单测未装配）：丢一次回写不影响提案自身事实
		return
	}
	res := &ticketHandleResult{
		ProposalId:       proposal.Id,
		Status:           proposal.Status,
		ExecRunId:        proposal.ExecRunId,
		ExecResult:       truncateTicketText(proposal.ExecResult),
		VerifyRunId:      proposal.VerifyRunId,
		VerifyConclusion: truncateTicketText(proposal.VerifyConclusion),
		Err:              proposal.Err,
	}
	if uerr := app.RecordBizHandleResult(ctx, proposal.ProcInstId, bizStatus, jsonx.ToStr(res)); uerr != nil {
		logx.WarnfContext(ctx, "[agentops-proposal] record ticket result failed (non-fatal): proposalId=%d, procInstId=%d, err=%v",
			proposal.Id, proposal.ProcInstId, uerr)
	}
}

// resolveProcDef 按 DefKey 定位生效的审批流程定义（不存在/未启用返回 nil）
func (g *proposalGatewayImpl) resolveProcDef(ctx context.Context) *flowentity.Procdef {
	procdefApp := flowProcdefFn()
	if procdefApp == nil {
		return nil
	}
	def := &flowentity.Procdef{DefKey: remediationProcDefKey}
	if err := procdefApp.GetByCond(def); err != nil {
		if !model.IsNotFound(err) {
			logx.WarnfContext(ctx, "[agentops-proposal] resolve procdef failed: defKey=%s, err=%v", remediationProcDefKey, err)
		}
		return nil
	}
	if def.Status != flowentity.ProcdefStatusEnable {
		return nil
	}
	return def
}

// startApprovalProc 以提案详情为业务表单发起 flow 审批工单（BizType=AgentRemediationFlowBizType）
func (g *proposalGatewayImpl) startApprovalProc(ctx context.Context, def *flowentity.Procdef, proposal *entity.Proposal) (*flowentity.Procinst, error) {
	procinstApp := flowProcinstFn()
	if procinstApp == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrProposalFlowNotReady)
	}
	bizForm := &AgentRemediationBizForm{
		ProposalId:     proposal.Id,
		Title:          proposal.Title,
		Action:         proposal.Action,
		Params:         proposal.Params,
		Risk:           proposal.Risk,
		Reason:         proposal.Reason,
		RunId:          proposal.RunId,
		ConversationId: proposal.ConversationId,
		OwnerUsername:  proposal.OwnerUsername,
	}
	return procinstApp.StartProc(ctx, def.Id, &flowdto.StarProc{
		BizType: AgentRemediationFlowBizType,
		BizKey:  bizKeyPrefix + strconv.FormatUint(proposal.Id, 10),
		BizForm: jsonx.ToStr(bizForm),
		Remark:  "[AI变更提案] " + proposal.Title,
	})
}

// FlowBizHandle 审批通过回调：领取执行权 → 投递受控执行异步体 → 立即返回。
//
// 仅在流程「已完成」（审批通过）时触发执行；驳回/取消不经此回调（由 SyncProcStatus 惰性对账）。
// 幂等：以 pending→approved 的 CAS 领取执行权，flow 清扫重驱动/并发回调时只有一方真正领取。
//
// 为什么不在这里直接跑执行：本回调由审批请求事务提交后的动作在同一 goroutine 内同步执行（也曾被
// 清扫任务串行重驱动）。里面直接跑「执行档 + 校验档」两次无头运行（默认各 5min）会把审批接口挂到
// 网关超时——而 SQL/命令其实已经执行（不可回滚），审批人看到「请求失败」极易再操作一次；同时
// RunSync 绕过有界工作池，本特性唯一的并发上限在这里会失效。
func (g *proposalGatewayImpl) FlowBizHandle(ctx context.Context, param *flowapp.BizHandleParam) (any, error) {
	if param == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrProposalBizParamNil)
	}
	procinst := param.Procinst
	if procinst.Status != flowentity.ProcinstStatusCompleted {
		// 非完成状态不处理（与 db 的 FlowBizHandle 口径一致）
		return nil, nil
	}

	proposal, err := g.locateByProcInst(ctx, procinst)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		// 两种途径都定位不到（非本模块工单或提案已删）：忽略，不报错影响 flow
		return nil, nil
	}
	if entity.IsProposalTerminal(proposal.Status) {
		return map[string]any{AckKeyProposalId: proposal.Id, AckKeyStatus: proposal.Status, AckKeyMessage: AckMessageAlreadyProcessed}, nil
	}

	// CAS 领取执行权：仅 pending → approved；落败说明已被其它路径领取（并发/重驱动）
	rows, err := g.GetRepo().UpdateColsByCond(ctx,
		base.NewPatch().Set("status", entity.ProposalStatusApproved),
		model.NewCond().EqStrict("id", proposal.Id).EqStrict("status", entity.ProposalStatusPending))
	if err != nil {
		return nil, err
	}
	if rows != 1 {
		return g.handleClaimLost(ctx, proposal.Id)
	}

	if err := g.submitForExecution(ctx, proposal.Id); err != nil {
		return nil, err
	}
	logx.InfofContext(ctx, "[agentops-proposal] approved, execution queued: proposalId=%d, action=%s", proposal.Id, proposal.Action)
	return map[string]any{
		AckKeyProposalId: proposal.Id,
		AckKeyStatus:     entity.ProposalStatusApproved,
		AckKeyExecution:  AckExecutionQueued,
	}, nil
}

// handleClaimLost CAS 落败后的收敛：
//   - 已终态或非本次能接的状态：幂等返回，不重复执行
//   - 仍停在 approved 且 exec_run_id=0：说明上一次领取后崩在执行之前 → 重新投递执行。
//     执行运行的幂等键 proposal-exec:<id> 保证「已真正执行过」时不会把变更再做第二遍
//     （命中幂等键短路返回既有运行）。
//     注意这条兜底只在 FlowBizHandle 被再次调用时才生效，而 flow 只重驱动「业务待处理」的工单，
//     本模块的工单在回调返回那一刻就已记为处理成功——所以它拦的是并发竞态，
//     真正的丢单自愈靠 SweepStalledProposals
func (g *proposalGatewayImpl) handleClaimLost(ctx context.Context, proposalId uint64) (any, error) {
	reloaded, err := g.GetById(proposalId)
	if err != nil || reloaded == nil {
		return map[string]any{AckKeyProposalId: proposalId, AckKeyMessage: AckMessageAlreadyClaimed}, nil
	}
	if reloaded.Status == entity.ProposalStatusApproved && reloaded.ExecRunId == 0 {
		if err := g.submitForExecution(ctx, reloaded.Id); err != nil {
			return nil, err
		}
		logx.WarnfContext(ctx, "[agentops-proposal] requeue orphaned approved proposal (previous executor crashed before start): proposalId=%d", proposalId)
		return map[string]any{AckKeyProposalId: proposalId, AckKeyStatus: reloaded.Status, AckKeyExecution: AckExecutionRequeued}, nil
	}
	return map[string]any{AckKeyProposalId: proposalId, AckKeyStatus: reloaded.Status, AckKeyMessage: AckMessageAlreadyClaimed}, nil
}

// locateByProcInst 按工单定位提案：先走 proc_inst_id；关联未落库（Record 回写失败）时按工单
// 业务键 aiops-proposal:<id> 反查兜底，并补回关联。
//
// 为什么必须有兜底：发起工单与回写 procInstId 是两次写（提案落库即视为登记成功），回写失败时若只按
// procInstId 找，工单批完也找不到提案 → 提案永久 pending、工单显示「已完成」、模型收到「已发起审批」
// 的回复，三处都说没问题而变更永不执行，是最难发现的一类失效。
// BizKey 与工单在同一次发起里生成（不依赖那次回写），所以它能作为可靠的反向定位键。
func (g *proposalGatewayImpl) locateByProcInst(ctx context.Context, procinst flowentity.Procinst) (*entity.Proposal, error) {
	proposal, err := g.GetRepo().GetByProcInstId(procinst.Id)
	if err == nil {
		return proposal, nil
	}
	if !model.IsNotFound(err) {
		return nil, err
	}
	id, ok := proposalIdFromBizKey(procinst.BizKey)
	if !ok {
		return nil, nil
	}
	repaired, getErr := g.GetById(id)
	if getErr != nil || repaired == nil {
		return nil, nil
	}
	// 补回关联（仅当仍未绑定，避免覆盖并发写入的其它工单关联）
	if _, uerr := g.GetRepo().UpdateColsByCond(ctx,
		base.NewPatch().Set("procInstId", procinst.Id).Set("procDefId", procinst.ProcdefId),
		model.NewCond().EqStrict("id", repaired.Id).EqStrict("procInstId", 0)); uerr != nil {
		logx.ErrorfContext(ctx, "[agentops-proposal] repair procinst binding failed (non-fatal): proposalId=%d, err=%v", repaired.Id, uerr)
	}
	logx.WarnfContext(ctx, "[agentops-proposal] proposal located by biz key (proc_inst_id was not bound): proposalId=%d, procInstId=%d", repaired.Id, procinst.Id)
	repaired.ProcInstId = procinst.Id
	return repaired, nil
}

// proposalIdFromBizKey 从工单业务键解析提案 id（非本模块键返回 false）
func proposalIdFromBizKey(bizKey string) (uint64, bool) {
	if !strings.HasPrefix(bizKey, bizKeyPrefix) {
		return 0, false
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(bizKey, bizKeyPrefix), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}
