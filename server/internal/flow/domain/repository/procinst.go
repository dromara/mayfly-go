package repository

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Procinst interface {
	base.Repo[*entity.Procinst]

	GetPageList(condition *entity.ProcinstQuery, orderBy ...string) (*model.PageResult[*entity.Procinst], error)
}
