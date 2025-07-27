package persistence

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type procdefImpl struct {
	base.RepoImpl[*entity.Procdef]
}

func newProcdefRepo() repository.Procdef {
	return &procdefImpl{}
}

func (p *procdefImpl) GetPageList(condition *entity.Procdef, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.ProcdefPagePO], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Like("def_key", condition.DefKey)

	var res []*entity.ProcdefPagePO
	return gormx.PageByCond(p.GetModel(), qd, pageParam, res)
}
