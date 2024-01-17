package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type teamRepoImpl struct {
	base.RepoImpl[*entity.Team]
}

func newTeamRepo() repository.Team {
	return &teamRepoImpl{}
}

func (p *teamRepoImpl) GetPageList(condition *entity.TeamQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(p.GetModel()).
		Like("name", condition.Name).
		WithOrderBy()
	return gormx.PageQuery(qd, pageParam, toEntity)
}
