package persistence

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
)

type machineScriptRepo struct{}

var MachineScriptDao repository.MachineScript = &machineScriptRepo{}

// 分页获取机器信息列表
func (m *machineScriptRepo) GetPageList(condition *entity.MachineScript, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, toEntity, orderBy...)
}

// 根据条件获取账号信息
func (m *machineScriptRepo) GetMachineScript(condition *entity.MachineScript, cols ...string) error {
	return model.GetBy(condition, cols...)
}

// 根据id获取
func (m *machineScriptRepo) GetById(id uint64, cols ...string) *entity.MachineScript {
	ms := new(entity.MachineScript)
	if err := model.GetById(ms, id, cols...); err != nil {
		return nil

	}
	return ms
}

// 根据id获取
func (m *machineScriptRepo) Delete(id uint64) {
	biz.ErrIsNil(model.DeleteById(new(entity.MachineScript), id), "删除失败")
}

func (m *machineScriptRepo) Create(entity *entity.MachineScript) {
	model.Insert(entity)
}

func (m *machineScriptRepo) UpdateById(entity *entity.MachineScript) {
	model.UpdateById(entity)
}
