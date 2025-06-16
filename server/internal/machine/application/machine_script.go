package application

import (
	"context"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

type MachineScript interface {
	base.App[*entity.MachineScript]

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MachineScript, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineScript], error)

	// GetScriptCategorys 获取脚本分类
	GetScriptCategorys(ctx context.Context) ([]string, error)

	Save(ctx context.Context, entity *entity.MachineScript) error

	Delete(ctx context.Context, id uint64)
}

var _ (MachineScript) = (*machineScriptAppImpl)(nil)

type machineScriptAppImpl struct {
	base.AppImpl[*entity.MachineScript, repository.MachineScript]

	machineApp Machine `inject:"T"`
}

const Common_Script_Machine_Id = 9999999

// 分页获取机器脚本信息列表
func (m *machineScriptAppImpl) GetPageList(condition *entity.MachineScript, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineScript], error) {
	return m.GetRepo().GetPageList(condition, pageParam, orderBy...)
}

func (m *machineScriptAppImpl) GetScriptCategorys(ctx context.Context) ([]string, error) {
	scripts, err := m.ListByCond(new(entity.MachineScript), "category")
	if err != nil {
		return nil, err
	}

	return collx.ArrayRemoveBlank(collx.ArrayDeduplicate(collx.ArrayMap(scripts, func(script *entity.MachineScript) string { return script.Category }))), nil
}

// 保存机器脚本
func (m *machineScriptAppImpl) Save(ctx context.Context, ms *entity.MachineScript) error {
	// 如果机器id不为公共脚本id，则校验机器是否存在
	if machineId := ms.MachineId; machineId != Common_Script_Machine_Id {
		_, err := m.machineApp.GetById(machineId, "Name")
		if err != nil {
			return errorx.NewBiz("machine not found")
		}
	}

	if ms.Id != 0 {
		return m.UpdateById(ctx, ms)
	}
	return m.Insert(ctx, ms)
}

// 根据id删除
func (m *machineScriptAppImpl) Delete(ctx context.Context, id uint64) {
	m.DeleteById(ctx, id)
}
