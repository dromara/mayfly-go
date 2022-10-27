package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type tagTreeTeamRepoImpl struct{}

func newTagTreeTeamRepo() repository.TagTreeTeam {
	return new(tagTreeTeamRepoImpl)
}

func (p *tagTreeTeamRepoImpl) ListProject(condition *entity.TagTreeTeam, toEntity interface{}, orderBy ...string) {
	model.ListByOrder(condition, toEntity, orderBy...)
}

func (p *tagTreeTeamRepoImpl) Save(pm *entity.TagTreeTeam) {
	biz.ErrIsNilAppendErr(model.Insert(pm), "保存团队项目信息失败：%s")
}

func (p *tagTreeTeamRepoImpl) DeleteBy(condition *entity.TagTreeTeam) {
	model.DeleteByCondition(condition)
}

func (p *tagTreeTeamRepoImpl) SelectTagPathsByAccountId(accountId uint64) []string {
	var res []string
	model.GetListBySql2Model("SELECT DISTINCT(t1.tag_path) FROM t_tag_tree_team t1 JOIN t_team_member t2 ON t1.team_id = t2.team_id WHERE t2.account_id = ?", &res, accountId)
	return res
}
