package application

import (
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/internal/devops/infrastructure/machine"
	"mayfly-go/internal/devops/infrastructure/persistence"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"

	"gorm.io/gorm"
)

type Machine interface {
	// 根据条件获取账号信息
	GetMachine(condition *entity.Machine, cols ...string) error

	Save(entity *entity.Machine)

	// 调整机器状态
	ChangeStatus(id uint64, status int8)

	Count(condition *entity.Machine) int64

	Delete(id uint64)

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Machine

	// 分页获取机器信息列表
	GetMachineList(condition *entity.Machine, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 获取机器连接
	GetCli(id uint64) *machine.Cli

	// 获取ssh隧道机器连接
	GetSshTunnelMachine(id uint64) *machine.SshTunnelMachine
}

type machineAppImpl struct {
	machineRepo repository.Machine
}

var MachineApp Machine = &machineAppImpl{machineRepo: persistence.MachineDao}

// 分页获取机器信息列表
func (m *machineAppImpl) GetMachineList(condition *entity.Machine, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return m.machineRepo.GetMachineList(condition, pageParam, toEntity, orderBy...)
}

func (m *machineAppImpl) Count(condition *entity.Machine) int64 {
	return m.machineRepo.Count(condition)
}

// 根据条件获取机器信息
func (m *machineAppImpl) Save(me *entity.Machine) {
	// ’修改机器信息且密码不为空‘ or ‘新增’需要测试是否可连接
	if (me.Id != 0 && me.Password != "") || me.Id == 0 {
		biz.ErrIsNilAppendErr(machine.TestConn(*me, func(u uint64) *entity.Machine { return m.GetById(u) }), "该机器无法连接: %s")
	}

	oldMachine := &entity.Machine{Ip: me.Ip, Port: me.Port, Username: me.Username}
	err := m.GetMachine(oldMachine)

	if me.Id != 0 {
		// 如果存在该库，则校验修改的库是否为该库
		if err == nil {
			biz.IsTrue(oldMachine.Id == me.Id, "该机器信息已存在")
		}
		// 关闭连接
		machine.DeleteCli(me.Id)
		me.PwdEncrypt()
		m.machineRepo.UpdateById(me)
	} else {
		biz.IsTrue(err != nil, "该机器信息已存在")
		// 新增机器，默认启用状态
		me.Status = entity.MachineStatusEnable
		me.PwdEncrypt()
		m.machineRepo.Create(me)
	}
}

func (m *machineAppImpl) ChangeStatus(id uint64, status int8) {
	if status == entity.MachineStatusDisable {
		// 关闭连接
		machine.DeleteCli(id)
	}
	machine := new(entity.Machine)
	machine.Id = id
	machine.Status = status
	m.machineRepo.UpdateById(machine)
}

// 根据条件获取机器信息
func (m *machineAppImpl) Delete(id uint64) {
	// 关闭连接
	machine.DeleteCli(id)
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
		machine := m.GetById(machineId)
		machine.PwdDecrypt()
		biz.IsTrue(machine.Status == entity.MachineStatusEnable, "该机器已被停用")
		return machine
	})
	biz.ErrIsNilAppendErr(err, "获取客户端错误: %s")
	return cli
}

func (m *machineAppImpl) GetSshTunnelMachine(id uint64) *machine.SshTunnelMachine {
	sshTunnel, err := machine.GetSshTunnelMachine(id, func(machineId uint64) *entity.Machine {
		machine := m.GetById(machineId)
		machine.PwdDecrypt()
		biz.IsTrue(machine.Status == entity.MachineStatusEnable, "该机器已被停用")
		return machine
	})
	biz.ErrIsNilAppendErr(err, "获取ssh隧道连接失败: %s")
	return sshTunnel
}
