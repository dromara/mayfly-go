package persistence

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type procinstImpl struct {
	base.RepoImpl[*entity.Procinst]
}

func newProcinstRepo() repository.Procinst {
	return &procinstImpl{}
}

func (p *procinstImpl) GetPageList(condition *entity.ProcinstQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewModelCond(condition)
	return p.PageByCondToAny(qd, pageParam, toEntity)
}

//-----------procinst task--------------

type procinstTaskImpl struct {
	base.RepoImpl[*entity.ProcinstTask]
}

func newProcinstTaskRepo() repository.ProcinstTask {
	return &procinstTaskImpl{}
}

func (p *procinstTaskImpl) GetPageList(condition *entity.ProcinstTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewModelCond(condition)
	return p.PageByCondToAny(qd, pageParam, toEntity)
}
