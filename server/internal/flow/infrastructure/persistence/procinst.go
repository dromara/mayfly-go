package persistence

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
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
	qd := gormx.NewQueryWithTableName("t_flow_procinst_task t").
		Joins("JOIN t_flow_procinst tp ON t.procinst_id = tp.id ").
		WithCond(model.NewCond().Columns("t.*, tp.biz_key").
			Eq("tp.biz_key", condition.BizKey).
			Eq0("tp.is_deleted", model.ModelUndeleted).
			Eq("tp.biz_type", condition.BizType).
			Eq0("t.is_deleted", model.ModelUndeleted).
			Eq("t.status", condition.Status).
			OrderByDesc("t.id"))
	return gormx.PageQuery(qd, pageParam, toEntity)
}
