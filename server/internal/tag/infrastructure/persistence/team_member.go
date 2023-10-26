package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type teamMemberRepoImpl struct {
	base.RepoImpl[*entity.TeamMember]
}

func newTeamMemberRepo() repository.TeamMember {
	return &teamMemberRepoImpl{base.RepoImpl[*entity.TeamMember]{M: new(entity.TeamMember)}}
}

func (p *teamMemberRepoImpl) ListMemeber(condition *entity.TeamMember, toEntity any, orderBy ...string) {
	gormx.ListByOrder(condition, toEntity, orderBy...)
}

func (p *teamMemberRepoImpl) GetPageList(condition *entity.TeamMember, pageParam *model.PageParam, toEntity any) (*model.PageResult[any], error) {
	qd := gormx.NewQueryWithTableName("t_team_member t").
		Select("t.*, a.name").
		Joins("JOIN t_sys_account a ON t.account_id = a.id AND a.status = 1").
		Eq("a.account_id", condition.AccountId).
		Eq0("a.is_deleted", model.ModelUndeleted).
		Eq("t.team_id", condition.TeamId).
		Eq0("t.is_deleted", model.ModelUndeleted).
		Like("a.username", condition.Username).
		OrderByDesc("t.id")
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (p *teamMemberRepoImpl) IsExist(teamId, accountId uint64) bool {
	return gormx.CountBy(&entity.TeamMember{TeamId: teamId, AccountId: accountId}) > 0
}
