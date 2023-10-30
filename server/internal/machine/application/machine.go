package application

import (
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"

	"gorm.io/gorm"
)

type Machine interface {
	base.App[*entity.Machine]

	Save(*entity.Machine) error

	// 测试机器连接
	TestConn(me *entity.Machine) error

	// 调整机器状态
	ChangeStatus(id uint64, status int8) error

	Count(condition *entity.MachineQuery) int64

	Delete(id uint64) error

	// 分页获取机器信息列表
	GetMachineList(condition *entity.MachineQuery, pageParam *model.PageParam, toEntity *[]*vo.MachineVO, orderBy ...string) (*model.PageResult[*[]*vo.MachineVO], error)

	// 获取机器连接
	GetCli(id uint64) (*mcm.Cli, error)

	// 获取ssh隧道机器连接
	GetSshTunnelMachine(id int) (*mcm.SshTunnelMachine, error)
}

func newMachineApp(machineRepo repository.Machine, authCertApp AuthCert) Machine {
	app := &machineAppImpl{
		authCertApp: authCertApp,
	}
	app.Repo = machineRepo
	return app
}

type machineAppImpl struct {
	base.AppImpl[*entity.Machine, repository.Machine]

	authCertApp AuthCert
}

// 分页获取机器信息列表
func (m *machineAppImpl) GetMachineList(condition *entity.MachineQuery, pageParam *model.PageParam, toEntity *[]*vo.MachineVO, orderBy ...string) (*model.PageResult[*[]*vo.MachineVO], error) {
	return m.GetRepo().GetMachineList(condition, pageParam, toEntity, orderBy...)
}

func (m *machineAppImpl) Count(condition *entity.MachineQuery) int64 {
	return m.GetRepo().Count(condition)
}

func (m *machineAppImpl) Save(me *entity.Machine) error {
	oldMachine := &entity.Machine{Ip: me.Ip, Port: me.Port, Username: me.Username}
	if me.SshTunnelMachineId > 0 {
		oldMachine.SshTunnelMachineId = me.SshTunnelMachineId
	}
	err := m.GetBy(oldMachine)

	me.PwdEncrypt()
	if me.Id == 0 {
		if err == nil {
			return errorx.NewBiz("该机器信息已存在")
		}
		// 新增机器，默认启用状态
		me.Status = entity.MachineStatusEnable
		return m.Insert(me)
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldMachine.Id != me.Id {
		return errorx.NewBiz("该机器信息已存在")
	}

	// 关闭连接
	mcm.DeleteCli(me.Id)
	return m.UpdateById(me)
}

func (m *machineAppImpl) TestConn(me *entity.Machine) error {
	me.Id = 0
	mi, err := m.toMachineInfo(me)
	if err != nil {
		return err
	}
	cli, err := mi.Conn()
	if err != nil {
		return err
	}
	cli.Close()
	return nil
}

func (m *machineAppImpl) ChangeStatus(id uint64, status int8) error {
	if status == entity.MachineStatusDisable {
		// 关闭连接
		mcm.DeleteCli(id)
	}
	machine := new(entity.Machine)
	machine.Id = id
	machine.Status = status
	return m.UpdateById(machine)
}

// 根据条件获取机器信息
func (m *machineAppImpl) Delete(id uint64) error {
	// 关闭连接
	mcm.DeleteCli(id)
	return gormx.Tx(
		func(db *gorm.DB) error {
			// 删除machine表信息
			return gormx.DeleteByIdWithDb(db, new(entity.Machine), id)
		},
		func(db *gorm.DB) error {
			// 删除machine_file
			return gormx.DeleteByWithDb(db, &entity.MachineFile{MachineId: id})
		},
		func(db *gorm.DB) error {
			// 删除machine_script
			return gormx.DeleteByWithDb(db, &entity.MachineScript{MachineId: id})
		},
	)
}

func (m *machineAppImpl) GetCli(machineId uint64) (*mcm.Cli, error) {
	return mcm.GetMachineCli(machineId, func(mid uint64) (*mcm.MachineInfo, error) {
		return m.toMachineInfoById(mid)
	})
}

func (m *machineAppImpl) GetSshTunnelMachine(machineId int) (*mcm.SshTunnelMachine, error) {
	return mcm.GetSshTunnelMachine(machineId, func(mid uint64) (*mcm.MachineInfo, error) {
		return m.toMachineInfoById(mid)
	})
}

// 生成机器信息，根据授权凭证id填充用户密码等
func (m *machineAppImpl) toMachineInfoById(machineId uint64) (*mcm.MachineInfo, error) {
	me, err := m.GetById(new(entity.Machine), machineId)
	if err != nil {
		return nil, errorx.NewBiz("机器信息不存在")
	}
	if me.Status != entity.MachineStatusEnable {
		return nil, errorx.NewBiz("该机器已被停用")
	}

	if mi, err := m.toMachineInfo(me); err != nil {
		return nil, err
	} else {
		return mi, nil
	}
}

func (m *machineAppImpl) toMachineInfo(me *entity.Machine) (*mcm.MachineInfo, error) {
	mi := new(mcm.MachineInfo)
	mi.Id = me.Id
	mi.Name = me.Name
	mi.Ip = me.Ip
	mi.Port = me.Port
	mi.Username = me.Username
	mi.TagPath = me.TagPath
	mi.EnableRecorder = me.EnableRecorder

	if me.UseAuthCert() {
		ac, err := m.authCertApp.GetById(new(entity.AuthCert), uint64(me.AuthCertId))
		if err != nil {
			return nil, errorx.NewBiz("授权凭证信息已不存在，请重新关联")
		}
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

	// 使用了ssh隧道，则将隧道机器信息也附上
	if me.SshTunnelMachineId > 0 {
		sshTunnelMe, err := m.GetById(new(entity.Machine), uint64(me.SshTunnelMachineId))
		if err != nil {
			return nil, errorx.NewBiz("隧道机器信息不存在")
		}
		sshTunnelMi, err := m.toMachineInfo(sshTunnelMe)
		if err != nil {
			return nil, err
		}
		mi.SshTunnelMachine = sshTunnelMi
	}
	return mi, nil
}
