package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type teamMemberRepoImpl struct{}

func newTeamMemberRepo() repository.TeamMember {
	return new(teamMemberRepoImpl)
}

func (p *teamMemberRepoImpl) ListMemeber(condition *entity.TeamMember, toEntity interface{}, orderBy ...string) {
	model.ListByOrder(condition, toEntity, orderBy...)
}

func (p *teamMemberRepoImpl) Save(pm *entity.TeamMember) {
	biz.ErrIsNilAppendErr(model.Insert(pm), "保存团队成员失败：%s")
}

func (p *teamMemberRepoImpl) GetPageList(condition *entity.TeamMember, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, condition, toEntity, orderBy...)
}

func (p *teamMemberRepoImpl) DeleteBy(condition *entity.TeamMember) {
	model.DeleteByCondition(condition)
}

func (p *teamMemberRepoImpl) IsExist(teamId, accountId uint64) bool {
	return model.CountBy(&entity.TeamMember{TeamId: teamId, AccountId: accountId}) > 0
}
