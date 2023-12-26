package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
)

type TagTree interface {
	base.Repo[*entity.TagTree]

	// 根据条件查询
	SelectByCondition(condition *entity.TagTreeQuery, toEntity any, orderBy ...string)
}
