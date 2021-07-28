package application

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
	"mayfly-go/server/devops/infrastructure/machine"
	"mayfly-go/server/devops/infrastructure/persistence"

	"gorm.io/gorm"
)

type Machine interface {
	// 根据条件获取账号信息
	GetMachine(condition *entity.Machine, cols ...string) error

	Save(entity *entity.Machine)

	Delete(id uint64)

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Machine

	// 分页获取机器信息列表
	GetMachineList(condition *entity.Machine, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 获取机器连接
	GetCli(id uint64) *machine.Cli
}

type machineAppImpl struct {
	machineRepo repository.Machine
}

var MachineApp Machine = &machineAppImpl{machineRepo: persistence.MachineDao}

// 分页获取机器信息列表
func (m *machineAppImpl) GetMachineList(condition *entity.Machine, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return m.machineRepo.GetMachineList(condition, pageParam, toEntity, orderBy...)
}

// 根据条件获取机器信息
func (m *machineAppImpl) Save(me *entity.Machine) {
	biz.ErrIsNilAppendErr(machine.TestConn(me), "该机器无法连接: %s")

	oldMachine := &entity.Machine{Ip: me.Ip, Port: me.Port, Username: me.Username}
	err := m.GetMachine(oldMachine)

	if me.Id != 0 {
		// 如果存在该库，则校验修改的库是否为该库
		if err == nil {
			biz.IsTrue(oldMachine.Id == me.Id, "该机器信息已存在")
		}
		// 关闭连接
		machine.Close(me.Id)
		m.machineRepo.UpdateById(me)
	} else {
		biz.IsTrue(err != nil, "该机器信息已存在")
		m.machineRepo.Create(me)
	}
}

// 根据条件获取机器信息
func (m *machineAppImpl) Delete(id uint64) {
	// 关闭连接
	machine.Close(id)
	model.Tx(
		func(db *gorm.DB) error {
			// 删除machine表信息
			return db.Delete(new(entity.Machine), "id = ?", id).Error
		},
		func(db *gorm.DB) error {
			// 删除machine_file
			machineFile := &entity.MachineFile{MachineId: id}
			return db.Where(machineFile).Delete(machineFile).Error
		},
		func(db *gorm.DB) error {
			// 删除machine_script
			machineScript := &entity.MachineScript{MachineId: id}
			return db.Where(machineScript).Delete(machineScript).Error
		},
	)
}

// 根据条件获取机器信息
func (m *machineAppImpl) GetMachine(condition *entity.Machine, cols ...string) error {
	return m.machineRepo.GetMachine(condition, cols...)
}

func (m *machineAppImpl) GetById(id uint64, cols ...string) *entity.Machine {
	return m.machineRepo.GetById(id, cols...)
}

func (m *machineAppImpl) GetCli(id uint64) *machine.Cli {
	cli, err := machine.GetCli(id, func(machineId uint64) *entity.Machine {
		return m.GetById(machineId)
	})
	biz.ErrIsNilAppendErr(err, "获取客户端错误: %s")
	return cli
}
