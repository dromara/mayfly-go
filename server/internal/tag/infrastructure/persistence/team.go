package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type teamRepoImpl struct{}

func newTeamRepo() repository.Team {
	return new(teamRepoImpl)
}

func (p *teamRepoImpl) GetPageList(condition *entity.Team, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(condition).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (p *teamRepoImpl) Insert(team *entity.Team) {
	biz.ErrIsNil(gormx.Insert(team), "新增团队失败")
}

func (p *teamRepoImpl) UpdateById(team *entity.Team) {
	biz.ErrIsNil(gormx.UpdateById(team), "更新团队失败")
}

func (p *teamRepoImpl) Delete(id uint64) {
	gormx.DeleteById(new(entity.Team), id)
}

func (p *teamRepoImpl) DeleteBy(team *entity.Team) {
	gormx.DeleteByCondition(team)
}
