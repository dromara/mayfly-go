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

func (p *procinstImpl) GetPageList(condition *entity.ProcinstQuery, orderBy ...string) (*model.PageResult[*entity.Procinst], error) {
	qd := model.NewModelCond(condition)
	return p.PageByCond(qd, condition.PageParam)
}
