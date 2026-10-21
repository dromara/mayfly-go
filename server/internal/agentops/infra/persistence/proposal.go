package persistence

import (
	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type proposalRepoImpl struct {
	base.RepoImpl[*entity.Proposal]
}

var _ repository.Proposal = (*proposalRepoImpl)(nil)

func newProposalRepo() repository.Proposal {
	return &proposalRepoImpl{}
}

// GetPageList 分页查询提案（列名 camelCase 由基座统一蛇形化）
func (p *proposalRepoImpl) GetPageList(condition *entity.Proposal, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Proposal], error) {
	qd := model.NewCond().
		Like("title", condition.Title).
		Eq("status", condition.Status).
		Eq("risk", condition.Risk).
		Eq("action", condition.Action).
		Eq("ownerAccountId", condition.OwnerAccountId).
		Eq("runId", condition.RunId).
		Eq("procInstId", condition.ProcInstId).
		OrderBy(orderBy...)
	return p.PageByCond(qd, pageParam)
}

// GetByProcInstId 按审批流程实例 id 精确定位提案（procInstId 恒非空，用 EqStrict）
func (p *proposalRepoImpl) GetByProcInstId(procInstId uint64) (*entity.Proposal, error) {
	proposal := new(entity.Proposal)
	if err := p.GetByCond(model.NewCond().EqStrict("procInstId", procInstId).Dest(proposal)); err != nil {
		return nil, err
	}
	return proposal, nil
}
