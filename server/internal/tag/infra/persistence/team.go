package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type teamRepoImpl struct {
	base.RepoImpl[*entity.Team]
}

func newTeamRepo() repository.Team {
	return &teamRepoImpl{}
}

func (p *teamRepoImpl) GetPageList(condition *entity.TeamQuery, orderBy ...string) (*model.PageResult[*entity.Team], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		OrderBy(orderBy...)
	return p.PageByCond(qd, condition.PageParam)
}
