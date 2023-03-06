package application

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/infrastructure/machine"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"

	"gorm.io/gorm"
)

type Machine interface {
	// 根据条件获取账号信息
	GetMachine(condition *entity.Machine, cols ...string) error

	Save(*entity.Machine)

	// 测试机器连接
	TestConn(me *entity.Machine)

	// 调整机器状态
	ChangeStatus(id uint64, status int8)

	Count(condition *entity.MachineQuery) int64

	Delete(id uint64)

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Machine

	// 分页获取机器信息列表
	GetMachineList(condition *entity.MachineQuery, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 获取机器连接
	GetCli(id uint64) *machine.Cli

	// 获取ssh隧道机器连接
	GetSshTunnelMachine(id int) *machine.SshTunnelMachine
}

func newMachineApp(machineRepo repository.Machine, authCertApp AuthCert) Machine {
	return &machineAppImpl{
		machineRepo: machineRepo,
		authCertApp: authCertApp,
	}
}

type machineAppImpl struct {
	machineRepo repository.Machine
	authCertApp AuthCert
}

// 分页获取机器信息列表
func (m *machineAppImpl) GetMachineList(condition *entity.MachineQuery, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return m.machineRepo.GetMachineList(condition, pageParam, toEntity, orderBy...)
}

func (m *machineAppImpl) Count(condition *entity.MachineQuery) int64 {
	return m.machineRepo.Count(condition)
}

func (m *machineAppImpl) Save(me *entity.Machine) {
	oldMachine := &entity.Machine{Ip: me.Ip, Port: me.Port, Username: me.Username}
	err := m.GetMachine(oldMachine)

	me.PwdEncrypt()
	if me.Id == 0 {
		biz.IsTrue(err != nil, "该机器信息已存在")
		// 新增机器，默认启用状态
		me.Status = entity.MachineStatusEnable
		m.machineRepo.Create(me)
		return
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil {
		biz.IsTrue(oldMachine.Id == me.Id, "该机器信息已存在")
	}

	// 关闭连接
	machine.DeleteCli(me.Id)
	m.machineRepo.UpdateById(me)
}

func (m *machineAppImpl) TestConn(me *entity.Machine) {
	me.Id = 0
	// 测试连接
	biz.ErrIsNilAppendErr(machine.TestConn(*m.toMachineInfo(me), func(u uint64) *machine.Info {
		return m.toMachineInfoById(u)
	}), "该机器无法连接: %s")
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

func (m *machineAppImpl) GetCli(machineId uint64) *machine.Cli {
	cli, err := machine.GetCli(machineId, func(mid uint64) *machine.Info {
		return m.toMachineInfoById(mid)
	})
	biz.ErrIsNilAppendErr(err, "获取客户端错误: %s")
	return cli
}

func (m *machineAppImpl) GetSshTunnelMachine(machineId int) *machine.SshTunnelMachine {
	sshTunnel, err := machine.GetSshTunnelMachine(machineId, func(mid uint64) *machine.Info {
		return m.toMachineInfoById(mid)
	})
	biz.ErrIsNilAppendErr(err, "获取ssh隧道连接失败: %s")
	return sshTunnel
}

// 生成机器信息，根据授权凭证id填充用户密码等
func (m *machineAppImpl) toMachineInfoById(machineId uint64) *machine.Info {
	me := m.GetById(machineId)
	biz.IsTrue(me.Status == entity.MachineStatusEnable, "该机器已被停用")

	return m.toMachineInfo(me)
}

func (m *machineAppImpl) toMachineInfo(me *entity.Machine) *machine.Info {
	mi := new(machine.Info)
	mi.Id = me.Id
	mi.Name = me.Name
	mi.Ip = me.Ip
	mi.Port = me.Port
	mi.Username = me.Username
	mi.TagId = me.TagId
	mi.TagPath = me.TagPath
	mi.EnableRecorder = me.EnableRecorder
	mi.SshTunnelMachineId = me.SshTunnelMachineId

	if me.UseAuthCert() {
		ac := m.authCertApp.GetById(uint64(me.AuthCertId))
		biz.NotNil(ac, "授权凭证信息已不存在，请重新关联")
		mi.AuthMethod = ac.AuthMethod
		ac.PwdDecrypt()
		mi.Password = ac.Password
		mi.Passphrase = ac.Passphrase
	} else {
		mi.AuthMethod = entity.AuthCertAuthMethodPassword
		if me.Id != 0 {
			me.PwdDecrypt()
		}
		mi.Password = me.Password
	}
	return mi
}
