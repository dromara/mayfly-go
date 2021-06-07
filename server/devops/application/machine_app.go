package application

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
	"mayfly-go/server/devops/infrastructure/machine"
	"mayfly-go/server/devops/infrastructure/persistence"
)

type IMachine interface {
	// 根据条件获取账号信息
	GetMachine(condition *entity.Machine, cols ...string) error

	Save(entity *entity.Machine)

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Machine

	// 分页获取机器信息列表
	GetMachineList(condition *entity.Machine, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult

	// 获取机器连接
	GetCli(id uint64) *machine.Cli
}

type machineApp struct {
	machineRepo repository.Machine
}

var Machine IMachine = &machineApp{machineRepo: persistence.MachineDao}

// 分页获取机器信息列表
func (m *machineApp) GetMachineList(condition *entity.Machine, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult {
	return m.machineRepo.GetMachineList(condition, pageParam, toEntity, orderBy...)
}

// 根据条件获取机器信息
func (m *machineApp) Save(entity *entity.Machine) {
	biz.ErrIsNil(machine.TestConn(entity), "该机器无法连接")
	if entity.Id != 0 {
		m.machineRepo.UpdateById(entity)
	} else {
		m.machineRepo.Create(entity)
	}
}

// 根据条件获取机器信息
func (m *machineApp) GetMachine(condition *entity.Machine, cols ...string) error {
	return m.machineRepo.GetMachine(condition, cols...)
}

func (m *machineApp) GetById(id uint64, cols ...string) *entity.Machine {
	return m.machineRepo.GetById(id, cols...)
}

func (m *machineApp) GetCli(id uint64) *machine.Cli {
	cli, err := machine.GetCli(id, func(machineId uint64) *entity.Machine {
		return m.GetById(machineId)
	})
	biz.ErrIsNil(err, "获取客户端错误")
	return cli
}
