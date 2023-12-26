package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
)

type TagResource interface {
	base.Repo[*entity.TagResource]

	SelectByCondition(condition *entity.TagResourceQuery, toEntity any, orderBy ...string)
}
