package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type machineScriptRepoImpl struct{}

func newMachineScriptRepo() repository.MachineScript {
	return new(machineScriptRepoImpl)
}

// 分页获取机器信息列表
func (m *machineScriptRepoImpl) GetPageList(condition *entity.MachineScript, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(condition).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

// 根据条件获取账号信息
func (m *machineScriptRepoImpl) GetMachineScript(condition *entity.MachineScript, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

// 根据id获取
func (m *machineScriptRepoImpl) GetById(id uint64, cols ...string) *entity.MachineScript {
	ms := new(entity.MachineScript)
	if err := gormx.GetById(ms, id, cols...); err != nil {
		return nil

	}
	return ms
}

// 根据id获取
func (m *machineScriptRepoImpl) Delete(id uint64) {
	biz.ErrIsNil(gormx.DeleteById(new(entity.MachineScript), id), "删除失败")
}

func (m *machineScriptRepoImpl) Create(entity *entity.MachineScript) {
	gormx.Insert(entity)
}

func (m *machineScriptRepoImpl) UpdateById(entity *entity.MachineScript) {
	gormx.UpdateById(entity)
}
