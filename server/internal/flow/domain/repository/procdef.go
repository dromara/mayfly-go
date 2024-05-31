package repository

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Procdef interface {
	base.Repo[*entity.Procdef]

	GetPageList(condition *entity.Procdef, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}
