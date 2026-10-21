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
	// 状态必须是显式过滤：启用(1)/停用(-1)，0 表示「不过滤」
	qd := model.NewCond().
		Like("name", condition.Name).
		Like("def_key", condition.DefKey).
		OrderBy(orderBy...)
	if condition.Status != 0 {
		qd = qd.EqStrict("status", condition.Status)
	}

	var res []*entity.ProcdefPagePO
	return gormx.PageByCond(p.GetModel(), qd, pageParam, res)
}
