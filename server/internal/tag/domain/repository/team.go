package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Team interface {
	base.Repo[*entity.Team]

	GetPageList(condition *entity.Team, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}
