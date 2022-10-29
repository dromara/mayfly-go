package persistence

import (
	"fmt"
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

func (p *teamMemberRepoImpl) GetPageList(condition *entity.TeamMember, pageParam *model.PageParam, toEntity interface{}) *model.PageResult {
	sql := "SELECT d.*, a.name FROM t_team_member d JOIN t_sys_account a ON d.account_id = a.id WHERE a.status = 1  "

	if condition.AccountId != 0 {
		sql = fmt.Sprintf("%s AND d.account_id = %d", sql, condition.AccountId)
	}
	if condition.TeamId != 0 {
		sql = fmt.Sprintf("%s AND d.team_id = %d", sql, condition.TeamId)
	}
	if condition.Username != "" {
		sql = sql + " AND d.Username LIKE '%" + condition.Username + "%'"
	}
	sql = sql + " ORDER BY d.id DESC"
	return model.GetPageBySql(sql, pageParam, toEntity)
}

func (p *teamMemberRepoImpl) DeleteBy(condition *entity.TeamMember) {
	model.DeleteByCondition(condition)
}

func (p *teamMemberRepoImpl) IsExist(teamId, accountId uint64) bool {
	return model.CountBy(&entity.TeamMember{TeamId: teamId, AccountId: accountId}) > 0
}
