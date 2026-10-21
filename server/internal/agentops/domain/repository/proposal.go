package repository

import (
	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

// Proposal 变更提案仓储
type Proposal interface {
	base.Repo[*entity.Proposal]

	// GetPageList 分页查询提案（按状态、风险级、归属账号、来源运行、审批实例、标题模糊过滤）
	GetPageList(condition *entity.Proposal, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Proposal], error)

	// GetByProcInstId 按审批流程实例 id 定位提案（审批回调据 BizKey/procinst 反查用）；
	// 不存在返回 model.ErrNotFound
	GetByProcInstId(procInstId uint64) (*entity.Proposal, error)
}
