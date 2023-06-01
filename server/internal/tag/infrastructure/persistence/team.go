package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type teamRepoImpl struct{}

func newTeamRepo() repository.Team {
	return new(teamRepoImpl)
}

func (p *teamRepoImpl) GetPageList(condition *entity.Team, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, condition, toEntity, orderBy...)
}

func (p *teamRepoImpl) Insert(team *entity.Team) {
	biz.ErrIsNil(model.Insert(team), "新增团队失败")
}

func (p *teamRepoImpl) UpdateById(team *entity.Team) {
	biz.ErrIsNil(model.UpdateById(team), "更新团队失败")
}

func (p *teamRepoImpl) Delete(id uint64) {
	model.DeleteById(new(entity.Team), id)
}

func (p *teamRepoImpl) DeleteBy(team *entity.Team) {
	model.DeleteByCondition(team)
}
