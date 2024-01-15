package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type machineTermOpRepoImpl struct {
	base.RepoImpl[*entity.MachineTermOp]
}

func newMachineTermOpRepoImpl() repository.MachineTermOp {
	return &machineTermOpRepoImpl{base.RepoImpl[*entity.MachineTermOp]{M: new(entity.MachineTermOp)}}
}

func (m *machineTermOpRepoImpl) GetPageList(condition *entity.MachineTermOp, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(condition).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

// 根据条件获取记录列表
func (m *machineTermOpRepoImpl) SelectByQuery(cond *entity.MachineTermOpQuery) ([]*entity.MachineTermOp, error) {
	qd := gormx.NewQuery(m.GetModel()).Le("create_time", cond.StartCreateTime)
	var res []*entity.MachineTermOp
	err := gormx.ListByQueryCond(qd, &res)
	return res, err
}
