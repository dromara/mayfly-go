package repository

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type ProcinstTask interface {
	base.Repo[*entity.ProcinstTask]

	GetPageList(condition *entity.ProcinstTaskQuery, orderBy ...string) (*model.PageResult[*entity.ProcinstTaskPO], error)
}
