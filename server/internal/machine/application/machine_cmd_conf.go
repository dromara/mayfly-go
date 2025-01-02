package application

import (
	"context"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"regexp"
)

type MachineCmd struct {
	CmdRegexp *regexp.Regexp // 命令正则表达式
	Stratege  string         // 策略（拒绝或审批等）
}

type MachineCmdConf interface {
	base.App[*entity.MachineCmdConf]

	SaveCmdConf(ctx context.Context, cmdConf *dto.SaveMachineCmdConf) error

	DeleteCmdConf(ctx context.Context, id uint64) error

	GetCmdConfsByMachineTags(ctx context.Context, tagPaths ...string) []*MachineCmd
}

type machineCmdConfAppImpl struct {
	base.AppImpl[*entity.MachineCmdConf, repository.MachineCmdConf]

	tagTreeRelateApp tagapp.TagTreeRelate `inject:"T"`
}

var _ (MachineCmdConf) = (*machineCmdConfAppImpl)(nil)

func (m *machineCmdConfAppImpl) SaveCmdConf(ctx context.Context, cmdConfParam *dto.SaveMachineCmdConf) error {
	cmdConf := cmdConfParam.CmdConf

	return m.Tx(ctx, func(ctx context.Context) error {
		return m.Save(ctx, cmdConf)
	}, func(ctx context.Context) error {
		return m.tagTreeRelateApp.RelateTag(ctx, tagentity.TagRelateTypeMachineCmd, cmdConf.Id, cmdConfParam.CodePaths...)
	})
}

func (m *machineCmdConfAppImpl) DeleteCmdConf(ctx context.Context, id uint64) error {
	_, err := m.GetById(id)
	if err != nil {
		return errorx.NewBiz("cmd config not found")
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

func (m *machineCmdConfAppImpl) GetCmdConfsByMachineTags(ctx context.Context, tagPaths ...string) []*MachineCmd {
	var cmds []*MachineCmd
	cmdConfIds, err := m.tagTreeRelateApp.GetRelateIds(ctx, tagentity.TagRelateTypeMachineCmd, tagPaths...)
	if err != nil {
		logx.Errorf("failed to get cmd config: %s", err.Error())
		return cmds
	}
	if len(cmdConfIds) == 0 {
		return cmds
	}

	cmdConfs, _ := m.GetByIds(cmdConfIds)
	for _, cmdConf := range cmdConfs {
		for _, cmd := range cmdConf.Cmds {
			if p, err := regexp.Compile(cmd); err != nil {
				logx.Errorf("cmd config [%s], regex compilation failed", cmd)
			} else {
				cmds = append(cmds, &MachineCmd{CmdRegexp: p})
			}
		}
	}
	return cmds
}
