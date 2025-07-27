package persistence

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type procinstTaskImpl struct {
	base.RepoImpl[*entity.ProcinstTask]
}

func newProcinstTaskRepo() repository.ProcinstTask {
	return &procinstTaskImpl{}
}

func (p *procinstTaskImpl) GetPageList(condition *entity.ProcinstTaskQuery, orderBy ...string) (*model.PageResult[*entity.ProcinstTaskPO], error) {
	qd := gormx.NewQueryWithTableName("t_flow_procinst_task t").
		Joins("JOIN t_flow_procinst tp ON t.procinst_id = tp.id JOIN t_flow_procinst_task_candidate tptc ON  tptc.task_id = t.id ").
		WithCond(model.NewCond().Columns("DISTINCT(t.id) id, t.procinst_id, t.execution_id, t.node_key, t.node_name, t.status, t.remark, t.create_time, tp.biz_key, tptc.status, tptc.duration, tptc.end_time, tptc.handler, tptc.candidate").
			Eq0("tp.is_deleted", model.ModelUndeleted).
			Eq0("t.is_deleted", model.ModelUndeleted).
			Eq0("tptc.is_deleted", model.ModelUndeleted).
			Eq("t.procinst_id", condition.ProcinstId).
			Eq("tp.biz_key", condition.BizKey).
			Eq("tp.biz_type", condition.BizType).
			Eq("tptc.handler", condition.Handler).
			Eq("tptc.status", condition.Status).
			In("tptc.candidate", condition.Candidates).
			OrderByDesc("t.id"))

	tasks := []*entity.ProcinstTaskPO{}
	return gormx.PageQuery(qd, condition.PageParam, tasks)
}
