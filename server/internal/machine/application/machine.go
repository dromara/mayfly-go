package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/event"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/infrastructure/cache"
	"mayfly-go/internal/machine/mcm"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/collx"
)

type Machine interface {
	base.App[*entity.Machine]

	SaveMachine(ctx context.Context, param *dto.SaveMachine) error

	// 测试机器连接
	TestConn(me *entity.Machine, authCert *tagentity.ResourceAuthCert) error

	// 调整机器状态
	ChangeStatus(ctx context.Context, id uint64, status int8) error

	Delete(ctx context.Context, id uint64) error

	// 分页获取机器信息列表
	GetMachineList(condition *entity.MachineQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 新建机器客户端连接（需手动调用Close）
	NewCli(authCertName string) (*mcm.Cli, error)

	// 获取已缓存的机器连接，若不存在则新建客户端连接并缓存，主要用于定时获取状态等（避免频繁创建连接）
	GetCli(id uint64) (*mcm.Cli, error)

	// 根据授权凭证获取客户端连接
	GetCliByAc(authCertName string) (*mcm.Cli, error)

	// 获取ssh隧道机器连接
	GetSshTunnelMachine(id int) (*mcm.SshTunnelMachine, error)

	// 定时更新机器状态信息
	TimerUpdateStats()

	// 获取机器运行时状态信息
	GetMachineStats(machineId uint64) (*mcm.Stats, error)

	ToMachineInfoByAc(ac string) (*mcm.MachineInfo, error)
}

type machineAppImpl struct {
	base.AppImpl[*entity.Machine, repository.Machine]

	tagApp              tagapp.TagTree          `inject:"TagTreeApp"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"ResourceAuthCertApp"`
}

var _ (Machine) = (*machineAppImpl)(nil)

// 注入MachineRepo
func (m *machineAppImpl) InjectMachineRepo(repo repository.Machine) {
	m.Repo = repo
}

// 分页获取机器信息列表
func (m *machineAppImpl) GetMachineList(condition *entity.MachineQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return m.GetRepo().GetMachineList(condition, pageParam, toEntity, orderBy...)
}

func (m *machineAppImpl) SaveMachine(ctx context.Context, param *dto.SaveMachine) error {
	me := param.Machine
	tagCodePaths := param.TagCodePaths
	authCerts := param.AuthCerts
	resourceType := tagentity.TagTypeMachine

	if len(authCerts) == 0 {
		return errorx.NewBiz("授权凭证信息不能为空")
	}

	oldMachine := &entity.Machine{
		Ip:                 me.Ip,
		Port:               me.Port,
		SshTunnelMachineId: me.SshTunnelMachineId,
	}

	err := m.GetByCond(oldMachine)
	if me.Id == 0 {
		if err == nil {
			return errorx.NewBiz("该机器信息已存在")
		}
		if m.CountByCond(&entity.Machine{Code: me.Code}) > 0 {
			return errorx.NewBiz("该编码已存在")
		}

		// 新增机器，默认启用状态
		me.Status = entity.MachineStatusEnable

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
		return errorx.NewBiz("该机器信息已存在")
	}
	// 如果调整了ssh username等会查不到旧数据，故需要根据id获取旧信息将code赋值给标签进行关联
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

func (m *machineAppImpl) TestConn(me *entity.Machine, authCert *tagentity.ResourceAuthCert) error {
	me.Id = 0

	authCert, err := m.resourceAuthCertApp.GetRealAuthCert(authCert)
	if err != nil {
		return err
	}

	mi, err := m.toMi(me, authCert)
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
		return errorx.NewBiz("机器信息不存在")
	}
	// 关闭连接
	mcm.DeleteCli(id)

	// 发布机器删除事件
	global.EventBus.Publish(ctx, event.EventTopicDeleteMachine, machine)

	resourceType := tagentity.TagTypeMachine
	return m.Tx(ctx,
		func(ctx context.Context) error {
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

func (m *machineAppImpl) NewCli(authCertName string) (*mcm.Cli, error) {
	if mi, err := m.ToMachineInfoByAc(authCertName); err != nil {
		return nil, err
	} else {
		return mi.Conn()
	}
}

func (m *machineAppImpl) GetCliByAc(authCertName string) (*mcm.Cli, error) {
	return mcm.GetMachineCli(authCertName, func(ac string) (*mcm.MachineInfo, error) {
		return m.ToMachineInfoByAc(ac)
	})
}

func (m *machineAppImpl) GetCli(machineId uint64) (*mcm.Cli, error) {
	cli, err := mcm.GetMachineCliById(machineId)
	if err == nil {
		return cli, nil
	}

	_, authCert, err := m.getMachineAndAuthCert(machineId)
	if err != nil {
		return nil, err
	}
	return m.GetCliByAc(authCert.Name)
}

func (m *machineAppImpl) GetSshTunnelMachine(machineId int) (*mcm.SshTunnelMachine, error) {
	return mcm.GetSshTunnelMachine(machineId, func(mid uint64) (*mcm.MachineInfo, error) {
		return m.ToMachineInfoById(mid)
	})
}

func (m *machineAppImpl) TimerUpdateStats() {
	logx.Debug("开始定时收集并缓存服务器状态信息...")
	scheduler.AddFun("@every 2m", func() {
		machineIds, _ := m.ListByCond(model.NewModelCond(&entity.Machine{Status: entity.MachineStatusEnable, Protocol: entity.MachineProtocolSsh}).Columns("id"))
		for _, ma := range machineIds {
			go func(mid uint64) {
				defer func() {
					if err := recover(); err != nil {
						logx.ErrorTrace(fmt.Sprintf("定时获取机器[id=%d]状态信息失败", mid), err.(error))
					}
				}()
				logx.Debugf("定时获取机器[id=%d]状态信息开始", mid)
				cli, err := m.GetCli(mid)
				if err != nil {
					logx.Errorf("定时获取机器[id=%d]状态信息失败, 获取机器cli失败: %s", mid, err.Error())
					return
				}
				cache.SaveMachineStats(mid, cli.GetAllStats())
				logx.Debugf("定时获取机器[id=%d]状态信息结束", mid)
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
		return nil, errorx.NewBiz("该授权凭证关联的机器信息不存在")
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
		return nil, nil, errorx.NewBiz("[%d]机器信息不存在", machineId)
	}
	if me.Status != entity.MachineStatusEnable && me.Protocol == 1 {
		return nil, nil, errorx.NewBiz("[%s]该机器已被停用", me.Code)
	}

	authCert, err := m.resourceAuthCertApp.GetResourceAuthCert(tagentity.TagTypeMachine, me.Code)
	if err != nil {
		return nil, nil, err
	}

	return me, authCert, nil
}

func (m *machineAppImpl) toMi(me *entity.Machine, authCert *tagentity.ResourceAuthCert) (*mcm.MachineInfo, error) {
	mi := new(mcm.MachineInfo)
	mi.Id = me.Id
	mi.Code = me.Code
	mi.Name = me.Name
	mi.Ip = me.Ip
	mi.Port = me.Port
	mi.CodePath = m.tagApp.ListTagPathByTypeAndCode(int8(tagentity.TagTypeMachineAuthCert), authCert.Name)
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
			Type: tagentity.TagTypeMachineAuthCert,
		}
	})

	return &tagdto.ResourceTag{
		Code:     me.Code,
		Type:     tagentity.TagTypeMachine,
		Name:     me.Name,
		Children: authCertTags,
	}
}
