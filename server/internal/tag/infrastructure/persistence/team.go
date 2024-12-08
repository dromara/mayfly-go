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
	return &teamRepoImpl{base.RepoImpl[*entity.Team]{M: new(entity.Team)}}
}

func (p *teamRepoImpl) GetPageList(condition *entity.TeamQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		OrderBy(orderBy...)
	return p.PageByCondToAny(qd, pageParam, toEntity)
}
