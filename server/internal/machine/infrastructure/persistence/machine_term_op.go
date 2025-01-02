package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type machineTermOpRepoImpl struct {
	base.RepoImpl[*entity.MachineTermOp]
}

func newMachineTermOpRepoImpl() repository.MachineTermOp {
	return &machineTermOpRepoImpl{}
}

func (m *machineTermOpRepoImpl) GetPageList(condition *entity.MachineTermOp, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	pd := model.NewModelCond(condition).OrderBy(orderBy...)
	return m.PageByCondToAny(pd, pageParam, toEntity)
}

// 根据条件获取记录列表
func (m *machineTermOpRepoImpl) SelectByQuery(cond *entity.MachineTermOpQuery) ([]*entity.MachineTermOp, error) {
	qd := model.NewCond().Le("create_time", cond.StartCreateTime)
	return m.SelectByCond(qd)
}
