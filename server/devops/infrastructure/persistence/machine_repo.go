package persistence

import (
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
)

type machineRepo struct{}

var MachineDao repository.Machine = &machineRepo{}

// 分页获取机器信息列表
func (m *machineRepo) GetMachineList(condition *entity.Machine, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult {
	return model.GetPage(pageParam, condition, toEntity, orderBy...)
}

// 根据条件获取账号信息
func (m *machineRepo) GetMachine(condition *entity.Machine, cols ...string) error {
	return model.GetBy(condition, cols...)
}

// 根据id获取
func (m *machineRepo) GetById(id uint64, cols ...string) *entity.Machine {
	machine := new(entity.Machine)
	if err := model.GetById(machine, id, cols...); err != nil {
		return nil

	}
	return machine
}

func (m *machineRepo) Create(entity *entity.Machine) {
	model.Insert(entity)
}

func (m *machineRepo) UpdateById(entity *entity.Machine) {
	model.UpdateById(entity)
}
