package repository

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Procinst interface {
	base.Repo[*entity.Procinst]

	GetPageList(condition *entity.ProcinstQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

type ProcinstTask interface {
	base.Repo[*entity.ProcinstTask]

	GetPageList(condition *entity.ProcinstTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}
