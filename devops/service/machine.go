package service

import (
	"mayfly-go/base/model"
	"mayfly-go/devops/apis/vo"
	"mayfly-go/devops/models"
)

type machineService struct {
}

func (m *machineService) GetMachineById(id uint64) *models.Machine {
	machine := new(models.Machine)
	machine.Id = id
	err := model.GetBy(machine)
	if err != nil {
		return nil
	}
	return machine
}

// 分页获取机器信息列表
func (m *machineService) GetMachineList(pageParam *model.PageParam) model.PageResult {
	return model.GetPage(pageParam, new(models.Machine), new([]vo.MachineVO), "Id desc")
}

// 获取所有需要监控的机器信息列表
func (m *machineService) GetNeedMonitorMachine() []map[string]interface{} {
	return model.GetListBySql("SELECT id FROM t_machine WHERE need_monitor = 1")
}
