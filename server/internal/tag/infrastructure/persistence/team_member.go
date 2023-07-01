package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type teamMemberRepoImpl struct{}

func newTeamMemberRepo() repository.TeamMember {
	return new(teamMemberRepoImpl)
}

func (p *teamMemberRepoImpl) ListMemeber(condition *entity.TeamMember, toEntity any, orderBy ...string) {
	gormx.ListByOrder(condition, toEntity, orderBy...)
}

func (p *teamMemberRepoImpl) Save(pm *entity.TeamMember) {
	biz.ErrIsNilAppendErr(gormx.Insert(pm), "保存团队成员失败：%s")
}

func (p *teamMemberRepoImpl) GetPageList(condition *entity.TeamMember, pageParam *model.PageParam, toEntity any) *model.PageResult[any] {
	qd := gormx.NewQuery(new(entity.TeamMember)).
		Select("t_team_member.*, a.name").
		Joins("JOIN t_sys_account a ON t_team_member.account_id = a.id AND a.status = 1").
		Eq("account_id", condition.AccountId).
		Eq("team_id", condition.TeamId).
		Like("a.username", condition.Username).
		OrderByDesc("t_team_member.id")
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (p *teamMemberRepoImpl) DeleteBy(condition *entity.TeamMember) {
	gormx.DeleteByCondition(condition)
}

func (p *teamMemberRepoImpl) IsExist(teamId, accountId uint64) bool {
	return gormx.CountBy(&entity.TeamMember{TeamId: teamId, AccountId: accountId}) > 0
}
