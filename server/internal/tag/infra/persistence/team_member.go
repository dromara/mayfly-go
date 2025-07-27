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
	return &teamMemberRepoImpl{}
}

func (p *teamMemberRepoImpl) ListMemeber(condition *entity.TeamMember, toEntity []any, orderBy ...string) {
	p.SelectByCondToAny(model.NewModelCond(condition).OrderBy(orderBy...), toEntity)
}

func (p *teamMemberRepoImpl) GetPageList(condition *entity.TeamMember, pageParam model.PageParam) (*model.PageResult[*entity.TeamMemberPO], error) {
	qd := gormx.NewQueryWithTableName("t_team_member t").
		Joins("JOIN t_sys_account a ON t.account_id = a.id AND a.status = 1").
		WithCond(model.NewCond().Columns("t.*, a.name").
			Eq("a.account_id", condition.AccountId).
			Eq0("a.is_deleted", model.ModelUndeleted).
			Eq("t.team_id", condition.TeamId).
			Eq0("t.is_deleted", model.ModelUndeleted).
			Like("a.username", condition.Username).
			OrderByDesc("t.id"))

	var res []*entity.TeamMemberPO
	return gormx.PageQuery(qd, pageParam, res)
}

func (p *teamMemberRepoImpl) IsExist(teamId, accountId uint64) bool {
	return p.CountByCond(&entity.TeamMember{TeamId: teamId, AccountId: accountId}) > 0
}
