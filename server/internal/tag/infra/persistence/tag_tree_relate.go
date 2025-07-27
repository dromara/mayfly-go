package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"time"
)

type tagTreeRelateRepoImpl struct {
	base.RepoImpl[*entity.TagTreeRelate]
}

func newTagTreeRelateRepo() repository.TagTreeRelate {
	return &tagTreeRelateRepoImpl{}
}

// SelectRelateIdsByTagPaths 根据标签路径查询相关联的id
func (tr *tagTreeRelateRepoImpl) SelectRelateIdsByTagPaths(relateType entity.TagRelateType, tagPaths ...string) ([]uint64, error) {
	var res []uint64
	sql := `
SELECT
	t1.relate_id
FROM
	t_tag_tree_relate t1
JOIN t_tag_tree t ON
	t.id = t1.tag_id
WHERE
	t1.relate_type = ?
	AND t.code_path in ?
	AND t.is_deleted = 0
	AND t1.is_deleted = 0
ORDER BY
	t.code_path
	`
	if err := tr.SelectBySql(sql, &res, relateType, tagPaths); err != nil {
		return res, err
	}
	return res, nil
}

func (tr *tagTreeRelateRepoImpl) SelectTagPathsByAccountId(accountId uint64) []string {
	var res []string
	sql := `
SELECT
	DISTINCT(t.code_path)
FROM t_tag_tree_relate t1
JOIN t_team_member t2 ON t1.relate_id = t2.team_id
JOIN t_team t3 ON t3.id = t2.team_id AND t3.validity_start_date < ? AND t3.validity_end_date > ?
JOIN t_tag_tree t ON t.id = t1.tag_id
WHERE
	t1.relate_type = ?
	AND t2.account_id = ?
	AND t1.is_deleted = 0
	AND t2.is_deleted = 0
	AND t.is_deleted = 0
ORDER BY
	t.code_path
	`
	now := time.Now()
	tr.SelectBySql(sql, &res, now, now, entity.TagRelateTypeTeam, accountId)
	return res
}

// SelectTagPathsByRelate 根据关联信息查询对应的关联的标签路径
func (tr *tagTreeRelateRepoImpl) SelectTagPathsByRelate(relateType entity.TagRelateType, relateId uint64) []string {
	var res []string
	sql := `
SELECT
	DISTINCT(t.code_path)
FROM
	t_tag_tree_relate t1
JOIN t_tag_tree t ON
	t.id = t1.tag_id
WHERE
	t1.relate_id = ?
	AND t1.relate_type = ?
	AND t.is_deleted = 0
	AND t1.is_deleted = 0
ORDER BY
	t.code_path
	`
	tr.SelectBySql(sql, &res, relateId, relateType)
	return res
}
