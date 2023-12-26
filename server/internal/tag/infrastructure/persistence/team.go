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
	return &teamRepoImpl{base.RepoImpl[*entity.Team]{M: new(entity.Team)}}
}

func (p *teamRepoImpl) GetPageList(condition *entity.Team, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(condition).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}
