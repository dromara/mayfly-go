package application

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/devops/domain/entity"
	"mayfly-go/devops/domain/repository"
	"mayfly-go/devops/infrastructure/persistence"
)

type IMachineScript interface {
	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MachineScript, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult

	// 根据条件获取
	GetMachineScript(condition *entity.MachineScript, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.MachineScript

	Save(entity *entity.MachineScript)

	Delete(id uint64)
}

type machineScriptApp struct {
	machineScriptRepo repository.MachineScript
	machineRepo       repository.Machine
}

const Common_Script_Machine_Id = 9999999

// 实现类单例
var MachineScript IMachineScript = &machineScriptApp{
	machineRepo:       persistence.MachineDao,
	machineScriptRepo: persistence.MachineScriptDao}

// 分页获取机器脚本信息列表
func (m *machineScriptApp) GetPageList(condition *entity.MachineScript, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult {
	return m.machineScriptRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 根据条件获取
func (m *machineScriptApp) GetMachineScript(condition *entity.MachineScript, cols ...string) error {
	return m.machineScriptRepo.GetMachineScript(condition, cols...)
}

// 根据id获取
func (m *machineScriptApp) GetById(id uint64, cols ...string) *entity.MachineScript {
	return m.machineScriptRepo.GetById(id, cols...)
}

// 保存机器脚本
func (m *machineScriptApp) Save(entity *entity.MachineScript) {
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
func (m *machineScriptApp) Delete(id uint64) {
	m.machineScriptRepo.Delete(id)
}
