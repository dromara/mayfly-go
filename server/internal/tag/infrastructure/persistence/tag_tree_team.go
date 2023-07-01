package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
)

type tagTreeTeamRepoImpl struct{}

func newTagTreeTeamRepo() repository.TagTreeTeam {
	return new(tagTreeTeamRepoImpl)
}

func (p *tagTreeTeamRepoImpl) ListTag(condition *entity.TagTreeTeam, toEntity any, orderBy ...string) {
	gormx.ListByOrder(condition, toEntity, orderBy...)
}

func (p *tagTreeTeamRepoImpl) Save(pm *entity.TagTreeTeam) {
	biz.ErrIsNilAppendErr(gormx.Insert(pm), "保存团队项目信息失败：%s")
}

func (p *tagTreeTeamRepoImpl) DeleteBy(condition *entity.TagTreeTeam) {
	gormx.DeleteByCondition(condition)
}

func (p *tagTreeTeamRepoImpl) SelectTagPathsByAccountId(accountId uint64) []string {
	var res []string
	gormx.GetListBySql2Model("SELECT DISTINCT(t1.tag_path) FROM t_tag_tree_team t1 JOIN t_team_member t2 ON t1.team_id = t2.team_id WHERE t2.account_id = ? ORDER BY t1.tag_path", &res, accountId)
	return res
}
