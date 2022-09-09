package application

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type MachineScript interface {
	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MachineScript, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 根据条件获取
	GetMachineScript(condition *entity.MachineScript, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.MachineScript

	Save(entity *entity.MachineScript)

	Delete(id uint64)
}

func newMachineScriptApp(machineScriptRepo repository.MachineScript, machineRepo repository.Machine) MachineScript {
	return &machineScriptAppImpl{machineRepo: machineRepo, machineScriptRepo: machineScriptRepo}

}

type machineScriptAppImpl struct {
	machineScriptRepo repository.MachineScript
	machineRepo       repository.Machine
}

const Common_Script_Machine_Id = 9999999

// // 实现类单例
// var MachineScriptApp MachineScript = &machineScriptAppImpl{
// 	machineRepo:       persistence.MachineDao,
// 	machineScriptRepo: persistence.MachineScriptDao}

// 分页获取机器脚本信息列表
func (m *machineScriptAppImpl) GetPageList(condition *entity.MachineScript, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return m.machineScriptRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 根据条件获取
func (m *machineScriptAppImpl) GetMachineScript(condition *entity.MachineScript, cols ...string) error {
	return m.machineScriptRepo.GetMachineScript(condition, cols...)
}

// 根据id获取
func (m *machineScriptAppImpl) GetById(id uint64, cols ...string) *entity.MachineScript {
	return m.machineScriptRepo.GetById(id, cols...)
}

// 保存机器脚本
func (m *machineScriptAppImpl) Save(entity *entity.MachineScript) {
	// 如果机器id不为公共脚本id，则校验机器是否存在
	if machineId := entity.MachineId; machineId != Common_Script_Machine_Id {
		biz.NotNil(m.machineRepo.GetById(machineId, "Name"), "该机器不存在")
	}

	if entity.Id != 0 {
		model.UpdateById(entity)
	} else {
		model.Insert(entity)
	}
}

// 根据id删除
func (m *machineScriptAppImpl) Delete(id uint64) {
	m.machineScriptRepo.Delete(id)
}
