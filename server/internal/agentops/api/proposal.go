package api

import (
	"strings"

	"mayfly-go/internal/agentops/api/vo"
	"mayfly-go/internal/agentops/application"
	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"

	"mayfly-go/internal/agentops/imsg"

	"github.com/spf13/cast"
)

// Proposal 变更提案 API：自主运维闭环「提案 → 审批 → 受控执行 → 验证」的记录视图。
//
// 提案由只读诊断档经 propose_remediation 工具登记（非用户手建），故本 API 只读展示 + 删除终态，
// 不提供新建/编辑（审批动作在 flow 工单页完成）。列表读取时顺带做审批状态惰性对账：
// flowEventBus 为 flow 内部总线（agentops 不可订阅），驳回/取消不触发回调，故在此 best-effort
// 按工单终态校准提案（已取消/终止 → rejected，已完成 → approved），避免前端长期显示过期的 pending。
// 对账只改状态，不在读路径上触发受控执行（执行由停滞清扫推进，见 SweepStalledProposals）。
type Proposal struct {
	proposalApp application.ProposalGateway `inject:"T"`
}

func (p *Proposal) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 提案分页列表（读取时对账审批状态）
		req.NewGet("", p.GetPageList),

		// 提案详情（含完整 params/reason/execResult/verifyConclusion）
		req.NewGet("/:id", p.GetById),

		// 删除提案（批量，逗号分隔；仅终态可删）
		req.NewDelete(":ids", p.Delete).Log(req.NewLogSaveI(imsg.LogProposalDelete)).RequiredPermissionCode("agentops:proposal:del"),
	}

	return req.NewConfs("/agentops/proposals", reqs[:]...)
}

// GetPageList 分页查询提案：先惰性对账审批状态，再裁剪大字段返回列表视图
func (p *Proposal) GetPageList(rc *req.Ctx) {
	cond, pageParam := rc.BindQueryAndPage[entity.Proposal]()
	// 提案里含高危变更的完整参数（SQL/命令/目标资源），与任务/触发器同口径收窄：
	// 否则任何持列表权限的账号都能读到他人环境的变更参数与执行结论
	scopeOwnerToCaller(rc, func(accountId uint64) { cond.OwnerAccountId = accountId })

	pageRes, err := p.proposalApp.GetPageList(cond, pageParam, "id desc")
	biz.ErrIsNil(err)

	// 读路径惰性对账：按工单终态校准待审批提案（已取消/终止 → rejected，已完成 → approved）。
	// best-effort，失败不影响列表
	p.proposalApp.SyncProcStatus(rc.MetaCtx, pageRes.List)

	rc.ResData = model.PageResultConv[*entity.Proposal, *vo.ProposalVO](pageRes)
}

// GetById 提案详情（含锁定参数、理由、执行结果与校验结论，供详情抽屉展示）
func (p *Proposal) GetById(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	proposal, err := p.proposalApp.GetById(id)
	biz.ErrIsNil(err)
	biz.ErrIsNil(assertResourceOwned(rc, proposal.OwnerAccountId))

	resVo := model.PageResultConv[*entity.Proposal, *vo.ProposalVO](&model.PageResult[*entity.Proposal]{List: []*entity.Proposal{proposal}})
	rc.ResData = resVo.List[0]
}

// Delete 批量删除提案（逗号分隔 id）；仅终态提案可删（网关内校验，在途审批拒绝删除）
func (p *Proposal) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("ids")
	rc.ReqParam = idsStr
	for _, id := range strings.Split(idsStr, ",") {
		// 跳过空段：批量删除末尾空串会变成 id=0 的无意义查询
		if id == "" {
			continue
		}
		proposalId := cast.ToUint64(id)
		// 先取后断言：提案只有终态可删，但「谁的终态」仍要分人——否则任何持删除权限的账号
		// 都能抹掉他人的变更审计痕迹（提案里带着「谁批准了什么高危变更」的完整证据）
		proposal, err := p.proposalApp.GetById(proposalId)
		biz.ErrIsNil(err)
		biz.ErrIsNil(assertResourceOwned(rc, proposal.OwnerAccountId))
		biz.ErrIsNil(p.proposalApp.Delete(rc.MetaCtx, proposalId))
	}
}

var _ req.RouterApi = (*Proposal)(nil)
