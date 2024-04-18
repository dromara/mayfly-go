package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
)

type tagTreeTeamRepoImpl struct {
	base.RepoImpl[*entity.TagTreeTeam]
}

func newTagTreeTeamRepo() repository.TagTreeTeam {
	return &tagTreeTeamRepoImpl{base.RepoImpl[*entity.TagTreeTeam]{M: new(entity.TagTreeTeam)}}
}

func (p *tagTreeTeamRepoImpl) SelectTagPathsByAccountId(accountId uint64) []string {
	var res []string
	sql := `
SELECT
	DISTINCT(t.code_path)
FROM
	t_tag_tree_team t1
JOIN t_team_member t2 ON
	t1.team_id = t2.team_id
JOIN t_tag_tree t ON
	t.id = t1.tag_id
WHERE
	t2.account_id = ?
	AND t1.is_deleted = 0
	AND t2.is_deleted = 0
	AND t.is_deleted = 0
ORDER BY
	t.code_path
	`
	gormx.GetListBySql2Model(sql, &res, accountId)
	return res
}
