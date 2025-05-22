package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/internal/machine/infrastructure/cache"
	"mayfly-go/internal/machine/mcm"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
)

type Machine interface {
	base.App[*entity.Machine]

	SaveMachine(ctx context.Context, param *dto.SaveMachine) error

	// 测试机器连接
	TestConn(ctx context.Context, me *entity.Machine, authCert *tagentity.ResourceAuthCert) error

	// 调整机器状态
	ChangeStatus(ctx context.Context, id uint64, status int8) error

	Delete(ctx context.Context, id uint64) error

	// 分页获取机器信息列表
	GetMachineList(condition *entity.MachineQuery, orderBy ...string) (*model.PageResult[*entity.Machine], error)

	// 新建机器客户端连接（需手动调用Close）
	NewCli(ctx context.Context, authCertName string) (*mcm.Cli, error)

	// 获取已缓存的机器连接，若不存在则新建客户端连接并缓存，主要用于定时获取状态等（避免频繁创建连接）
	GetCli(ctx context.Context, id uint64) (*mcm.Cli, error)

	// 根据授权凭证获取客户端连接
	GetCliByAc(ctx context.Context, authCertName string) (*mcm.Cli, error)

	// 获取ssh隧道机器连接
	GetSshTunnelMachine(ctx context.Context, id int) (*mcm.SshTunnelMachine, error)

	// 定时更新机器状态信息
	TimerUpdateStats()

	// 获取机器运行时状态信息
	GetMachineStats(machineId uint64) (*mcm.Stats, error)

	ToMachineInfoByAc(ac string) (*mcm.MachineInfo, error)
}

type machineAppImpl struct {
	base.AppImpl[*entity.Machine, repository.Machine]

	tagApp              tagapp.TagTree          `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`

	machineScriptApp MachineScript `inject:"T"`
	machineFileApp   MachineFile   `inject:"T"`
}

var _ (Machine) = (*machineAppImpl)(nil)

// 分页获取机器信息列表
func (m *machineAppImpl) GetMachineList(condition *entity.MachineQuery, orderBy ...string) (*model.PageResult[*entity.Machine], error) {
	return m.GetRepo().GetMachineList(condition, orderBy...)
}

func (m *machineAppImpl) SaveMachine(ctx context.Context, param *dto.SaveMachine) error {
	me := param.Machine
	tagCodePaths := param.TagCodePaths
	authCerts := param.AuthCerts
	resourceType := tagentity.TagTypeMachine

	if len(authCerts) == 0 {
		return errorx.NewBiz("ac cannot be empty")
	}

	oldMachine := &entity.Machine{
		Ip:                 me.Ip,
		Port:               me.Port,
		SshTunnelMachineId: me.SshTunnelMachineId,
	}

	if me.SshTunnelMachineId > 0 {
		if err := m.checkSshTunnelMachine(ctx, me.Ip, me.Port, me.SshTunnelMachineId, nil); err != nil {
			return err
		}
	}

	err := m.GetByCond(oldMachine)
	if me.Id == 0 {
		if err == nil {
			return errorx.NewBizI(ctx, imsg.ErrMachineExist)
		}

		// 新增机器，默认启用状态
		me.Status = entity.MachineStatusEnable
		// 生成随机编号
		me.Code = stringx.Rand(10)

		return m.Tx(ctx, func(ctx context.Context) error {
			return m.Insert(ctx, me)
		}, func(ctx context.Context) error {
			return m.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
				ResourceCode: me.Code,
				ResourceType: resourceType,
				AuthCerts:    authCerts,
			})

		}, func(ctx context.Context) error {
			return m.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
				ResourceTag:        m.genMachineResourceTag(me, authCerts),
				ParentTagCodePaths: tagCodePaths,
			})
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldMachine.Id != me.Id {
		return errorx.NewBizI(ctx, imsg.ErrMachineExist)
	}
	// 如果调整了SshTunnelMachineId Ip port等会查不到旧数据，故需要根据id获取旧信息将code赋值给标签进行关联
	if oldMachine.Code == "" {
		oldMachine, _ = m.GetById(me.Id)
	}

	// 关闭连接
	mcm.DeleteCli(me.Id)
	// 防止误传修改
	me.Code = ""
	return m.Tx(ctx, func(ctx context.Context) error {
		return m.UpdateById(ctx, me)
	}, func(ctx context.Context) error {
		return m.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
			ResourceCode: oldMachine.Code,
			ResourceType: resourceType,
			AuthCerts:    authCerts,
		})
	}, func(ctx context.Context) error {
		if oldMachine.Name != me.Name {
			if err := m.tagApp.UpdateTagName(ctx, tagentity.TagTypeMachine, oldMachine.Code, me.Name); err != nil {
				return err
			}
		}
		return m.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag:        m.genMachineResourceTag(oldMachine, authCerts),
			ParentTagCodePaths: tagCodePaths,
		})
	})
}

func (m *machineAppImpl) TestConn(ctx context.Context, me *entity.Machine, authCert *tagentity.ResourceAuthCert) error {
	me.Id = 0

	authCert, err := m.resourceAuthCertApp.GetRealAuthCert(authCert)
	if err != nil {
		return err
	}

	mi, err := m.toMi(me, authCert)
	if err != nil {
		return err
	}
	cli, err := mi.Conn(ctx)
	if err != nil {
		return err
	}
	cli.Close()
	return nil
}

func (m *machineAppImpl) ChangeStatus(ctx context.Context, id uint64, status int8) error {
	if status == entity.MachineStatusDisable {
		// 关闭连接
		mcm.DeleteCli(id)
	}
	machine := new(entity.Machine)
	machine.Id = id
	machine.Status = status
	return m.UpdateById(ctx, machine)
}

// 根据条件获取机器信息
func (m *machineAppImpl) Delete(ctx context.Context, id uint64) error {
	machine, err := m.GetById(id)
	if err != nil {
		return errorx.NewBiz("machine not found")
	}
	// 关闭连接
	mcm.DeleteCli(id)

	resourceType := tagentity.TagTypeMachine
	return m.Tx(ctx,
		func(ctx context.Context) error {
			if err := m.machineFileApp.DeleteByCond(ctx, &entity.MachineFile{MachineId: id}); err != nil {
				return err
			}
			if err := m.machineScriptApp.DeleteByCond(ctx, &entity.MachineScript{MachineId: id}); err != nil {
				return err
			}
			return m.DeleteById(ctx, id)
		}, func(ctx context.Context) error {
			return m.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
				ResourceTag: &tagdto.ResourceTag{
					Code: machine.Code,
					Type: tagentity.TagTypeMachine,
				},
			})
		}, func(ctx context.Context) error {
			return m.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
				ResourceCode: machine.Code,
				ResourceType: resourceType,
			})
		})
}

func (m *machineAppImpl) NewCli(ctx context.Context, authCertName string) (*mcm.Cli, error) {
	if mi, err := m.ToMachineInfoByAc(authCertName); err != nil {
		return nil, err
	} else {
		return mi.Conn(ctx)
	}
}

func (m *machineAppImpl) GetCliByAc(ctx context.Context, authCertName string) (*mcm.Cli, error) {
	return mcm.GetMachineCli(ctx, authCertName, func(ac string) (*mcm.MachineInfo, error) {
		return m.ToMachineInfoByAc(ac)
	})
}

func (m *machineAppImpl) GetCli(ctx context.Context, machineId uint64) (*mcm.Cli, error) {
	_, authCert, err := m.getMachineAndAuthCert(machineId)
	if err != nil {
		return nil, err
	}
	return m.GetCliByAc(ctx, authCert.Name)
}

func (m *machineAppImpl) GetSshTunnelMachine(ctx context.Context, machineId int) (*mcm.SshTunnelMachine, error) {
	return mcm.GetSshTunnelMachine(ctx, machineId, func(mid uint64) (*mcm.MachineInfo, error) {
		return m.ToMachineInfoById(mid)
	})
}

func (m *machineAppImpl) TimerUpdateStats() {
	logx.Debug("start collecting and caching machine state information periodically...")
	scheduler.AddFun("@every 2m", func() {
		machineIds, _ := m.ListByCond(model.NewModelCond(&entity.Machine{Status: entity.MachineStatusEnable, Protocol: entity.MachineProtocolSsh}).Columns("id"))
		for _, ma := range machineIds {
			go func(mid uint64) {
				defer func() {
					if err := recover(); err != nil {
						logx.ErrorTrace(fmt.Sprintf("failed to get machine [id=%d] status information on time", mid), err.(error))
					}
				}()
				logx.Debugf("time to get machine [id=%d] status information start", mid)
				ctx, cancelFunc := context.WithCancel(context.Background())
				defer cancelFunc()
				cli, err := m.GetCli(ctx, mid)
				if err != nil {
					logx.Errorf("failed to get machine [id=%d] status information periodically, failed to get machine cli: %s", mid, err.Error())
					return
				}
				cache.SaveMachineStats(mid, cli.GetAllStats())
				logx.Debugf("time to get the machine [id=%d] status information end", mid)
			}(ma.Id)
		}
	})
}

func (m *machineAppImpl) GetMachineStats(machineId uint64) (*mcm.Stats, error) {
	return cache.GetMachineStats(machineId)
}

// 根据授权凭证,生成机器信息
func (m *machineAppImpl) ToMachineInfoByAc(authCertName string) (*mcm.MachineInfo, error) {
	authCert, err := m.resourceAuthCertApp.GetAuthCert(authCertName)
	if err != nil {
		return nil, err
	}

	machine := &entity.Machine{
		Code: authCert.ResourceCode,
	}
	if err := m.GetByCond(machine); err != nil {
		return nil, errorx.NewBiz("the machine information associated with the authorization credential does not exist")
	}

	return m.toMi(machine, authCert)
}

// 生成机器信息，根据授权凭证id填充用户密码等
func (m *machineAppImpl) ToMachineInfoById(machineId uint64) (*mcm.MachineInfo, error) {
	me, authCert, err := m.getMachineAndAuthCert(machineId)
	if err != nil {
		return nil, err
	}

	return m.toMi(me, authCert)
}

func (m *machineAppImpl) getMachineAndAuthCert(machineId uint64) (*entity.Machine, *tagentity.ResourceAuthCert, error) {
	me, err := m.GetById(machineId)
	if err != nil {
		return nil, nil, errorx.NewBiz("[%d] machine not found", machineId)
	}
	if me.Status != entity.MachineStatusEnable && me.Protocol == 1 {
		return nil, nil, errorx.NewBiz("[%s] machine has been disable", me.Code)
	}

	authCert, err := m.resourceAuthCertApp.GetResourceAuthCert(tagentity.TagTypeMachine, me.Code)
	if err != nil {
		return nil, nil, err
	}

	return me, authCert, nil
}

func (m *machineAppImpl) toMi(me *entity.Machine, authCert *tagentity.ResourceAuthCert) (*mcm.MachineInfo, error) {
	mi := new(mcm.MachineInfo)
	mi.ExtraData = me.ExtraData
	mi.Id = me.Id
	mi.Code = me.Code
	mi.Name = me.Name
	mi.Ip = me.Ip
	mi.Port = me.Port
	mi.CodePath = m.tagApp.ListTagPathByTypeAndCode(int8(tagentity.TagTypeAuthCert), authCert.Name)
	mi.EnableRecorder = me.EnableRecorder
	mi.Protocol = me.Protocol

	mi.AuthCertName = authCert.Name
	mi.Username = authCert.Username
	mi.Password = authCert.Ciphertext
	mi.Passphrase = authCert.GetExtraString(tagentity.ExtraKeyPassphrase)
	mi.AuthMethod = int8(authCert.CiphertextType)

	// 使用了ssh隧道，则将隧道机器信息也附上
	if me.SshTunnelMachineId > 0 {
		sshTunnelMi, err := m.ToMachineInfoById(uint64(me.SshTunnelMachineId))
		if err != nil {
			return nil, err
		}
		mi.SshTunnelMachine = sshTunnelMi
	}
	return mi, nil
}

func (m *machineAppImpl) genMachineResourceTag(me *entity.Machine, authCerts []*tagentity.ResourceAuthCert) *tagdto.ResourceTag {
	authCertTags := collx.ArrayMap[*tagentity.ResourceAuthCert, *tagdto.ResourceTag](authCerts, func(val *tagentity.ResourceAuthCert) *tagdto.ResourceTag {
		return &tagdto.ResourceTag{
			Code: val.Name,
			Name: val.Username,
			Type: tagentity.TagTypeAuthCert,
		}
	})

	return &tagdto.ResourceTag{
		Code:     me.Code,
		Type:     tagentity.TagTypeMachine,
		Name:     me.Name,
		Children: authCertTags,
	}
}

// checkSshTunnelMachine 校验ssh隧道机器是否存在循环隧道
func (m *machineAppImpl) checkSshTunnelMachine(ctx context.Context, ip string, port int, sshTunnelMachineId int, visited map[string]bool) error {
	if visited == nil {
		visited = make(map[string]bool)
	}
	visited[fmt.Sprintf("%s:%d", ip, port)] = true

	stm, err := m.GetById(uint64(sshTunnelMachineId))
	if err != nil {
		return err
	}

	if visited[fmt.Sprintf("%s:%d", stm.Ip, stm.Port)] {
		return errorx.NewBizI(ctx, imsg.ErrSshTunnelCircular)
	}

	if stm.SshTunnelMachineId > 0 {
		return m.checkSshTunnelMachine(ctx, stm.Ip, stm.Port, stm.SshTunnelMachineId, visited)
	}
	return nil
}
