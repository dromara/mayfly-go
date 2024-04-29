package application

import (
	"context"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"regexp"
)

type SaveMachineCmdConfParam struct {
	CmdConf   *entity.MachineCmdConf
	CodePaths []string
}

type MachineCmd struct {
	CmdRegexp *regexp.Regexp // 命令正则表达式
	Stratege  string         // 策略（拒绝或审批等）
}

type MachineCmdConf interface {
	base.App[*entity.MachineCmdConf]

	SaveCmdConf(ctx context.Context, cmdConf *SaveMachineCmdConfParam) error

	DeleteCmdConf(ctx context.Context, id uint64) error

	GetCmdConfsByMachineTags(tagPaths ...string) []*MachineCmd
}

type machineCmdConfAppImpl struct {
	base.AppImpl[*entity.MachineCmdConf, repository.MachineCmdConf]

	tagTreeRelateApp tagapp.TagTreeRelate `inject:"TagTreeRelateApp"`
}

var _ (MachineCmdConf) = (*machineCmdConfAppImpl)(nil)

// 注入MachineCmdConfRepo
func (m *machineCmdConfAppImpl) InjectMachineCmdConfRepo(repo repository.MachineCmdConf) {
	m.Repo = repo
}

func (m *machineCmdConfAppImpl) SaveCmdConf(ctx context.Context, cmdConfParam *SaveMachineCmdConfParam) error {
	cmdConf := cmdConfParam.CmdConf

	return m.Tx(ctx, func(ctx context.Context) error {
		return m.Save(ctx, cmdConf)
	}, func(ctx context.Context) error {
		return m.tagTreeRelateApp.RelateTag(ctx, tagentity.TagRelateTypeMachineCmd, cmdConf.Id, cmdConfParam.CodePaths...)
	})
}

func (m *machineCmdConfAppImpl) DeleteCmdConf(ctx context.Context, id uint64) error {
	_, err := m.GetById(new(entity.MachineCmdConf), id)
	if err != nil {
		return errorx.NewBiz("该命令配置不存在")
	}

	return m.Tx(ctx, func(ctx context.Context) error {
		return m.DeleteById(ctx, id)
	}, func(ctx context.Context) error {
		return m.tagTreeRelateApp.DeleteByCond(ctx, &tagentity.TagTreeRelate{
			RelateType: tagentity.TagRelateTypeMachineCmd,
			RelateId:   id,
		})
	})
}

func (m *machineCmdConfAppImpl) GetCmdConfsByMachineTags(tagPaths ...string) []*MachineCmd {
	var cmds []*MachineCmd
	cmdConfIds, err := m.tagTreeRelateApp.GetRelateIds(tagentity.TagRelateTypeMachineCmd, tagPaths...)
	if err != nil {
		logx.Errorf("获取命令配置信息失败: %s", err.Error())
		return cmds
	}
	if len(cmdConfIds) == 0 {
		return cmds
	}

	var cmdConfs []*entity.MachineCmdConf
	m.GetByIds(&cmdConfs, cmdConfIds)

	for _, cmdConf := range cmdConfs {
		for _, cmd := range cmdConf.Cmds {
			if p, err := regexp.Compile(cmd); err != nil {
				logx.Errorf("命令配置[%s]，正则编译失败", cmd)
			} else {
				cmds = append(cmds, &MachineCmd{CmdRegexp: p})
			}
		}
	}
	return cmds
}
