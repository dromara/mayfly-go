package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
)

type tagResourceRepoImpl struct {
	base.RepoImpl[*entity.TagResource]
}

func newTagResourceRepo() repository.TagResource {
	return &tagResourceRepoImpl{base.RepoImpl[*entity.TagResource]{M: new(entity.TagResource)}}
}

func (p *tagResourceRepoImpl) SelectByCondition(condition *entity.TagResourceQuery, toEntity any, orderBy ...string) {
	sql := "SELECT tr.resource_type, tr.resource_code, tr.tag_id, tr.tag_path FROM t_tag_resource tr WHERE tr.is_deleted = 0 "

	params := make([]any, 0)

	if condition.ResourceType != 0 {
		sql = sql + " AND tr.resource_type = ?"
		params = append(params, condition.ResourceType)
	}

	if condition.ResourceCode != "" {
		sql = sql + " AND tr.resource_code = ?"
		params = append(params, condition.ResourceCode)
	}

	if len(condition.ResourceCodes) > 0 {
		sql = sql + " AND tr.resource_code IN (?)"
		params = append(params, condition.ResourceCodes)
	}

	if condition.TagId != 0 {
		sql = sql + " AND tr.tag_id = ?"
		params = append(params, condition.TagId)
	}
	if condition.TagPath != "" {
		sql = sql + " AND tr.tag_path = ?"
		params = append(params, condition.TagPath)
	}
	if condition.TagPathLike != "" {
		sql = sql + " AND tr.tag_path LIKE ?"
		params = append(params, condition.TagPathLike+"%")
	}
	if len(condition.TagPathLikes) > 0 {
		sql = sql + " AND ("
		for i, v := range condition.TagPathLikes {
			if i == 0 {
				sql = sql + "tr.tag_path LIKE ?"
			} else {
				sql = sql + " OR tr.tag_path LIKE ?"
			}
			params = append(params, v+"%")
		}
		sql = sql + ")"
	}

	sql = sql + " ORDER BY tr.tag_path"
	gormx.GetListBySql2Model(sql, toEntity, params...)
}
