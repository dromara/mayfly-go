package persistence

import (
	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type agentTaskRepoImpl struct {
	base.RepoImpl[*entity.AgentTask]
}

var _ repository.AgentTask = (*agentTaskRepoImpl)(nil)

func newAgentTaskRepo() repository.AgentTask {
	return &agentTaskRepoImpl{}
}

// GetPageList 分页查询任务（列名 camelCase 由基座统一蛇形化）
func (a *agentTaskRepoImpl) GetPageList(condition *entity.AgentTask, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.AgentTask], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Eq("status", condition.Status).
		Eq("profileId", condition.ProfileId).
		Eq("ownerAccountId", condition.OwnerAccountId).
		OrderBy(orderBy...)
	return a.PageByCond(qd, pageParam)
}
